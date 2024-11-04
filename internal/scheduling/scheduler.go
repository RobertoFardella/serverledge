package scheduling

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"runtime"
	"time"

	"github.com/grussorusso/serverledge/internal/metrics"

	"github.com/grussorusso/serverledge/internal/node"

	"github.com/grussorusso/serverledge/internal/config"

	"github.com/grussorusso/serverledge/internal/container"
	"github.com/grussorusso/serverledge/internal/function"
)

var requests chan *scheduledRequest
var completions chan *completionNotification
var remoteServerUrl string
var offloadingClient *http.Client

func Run(p Policy) {
	requests = make(chan *scheduledRequest, 500)
	completions = make(chan *completionNotification, 500)

	// initialize Resources
	availableCores := runtime.NumCPU()
	node.Resources.AvailableMemMB = int64(config.GetInt(config.POOL_MEMORY_MB, 1024))
	node.Resources.AvailableCPUs = config.GetFloat(config.POOL_CPUS, float64(availableCores))
	node.Resources.ContainerPools = make(map[string]*node.ContainerPool)
	log.Printf("Current resources: %v\n", &node.Resources)

	container.InitDockerContainerFactory()

	//janitor periodically remove expired warm container
	node.GetJanitorInstance()

	tr := &http.Transport{
		MaxIdleConns:        2500,
		MaxIdleConnsPerHost: 2500,
		MaxConnsPerHost:     0,
		IdleConnTimeout:     30 * time.Minute,
	}
	offloadingClient = &http.Client{Transport: tr}

	// initialize scheduling policy
	p.Init()

	remoteServerUrl = config.GetString(config.CLOUD_URL, "") //this is unused!

	log.Println("Scheduler started.")

	var r *scheduledRequest
	var c *completionNotification
	for {
		select {
		case r = <-requests:
			go p.OnArrival(r)
		case c = <-completions:
			node.ReleaseResources(c.contID, r.Istance_number, c.fun)
			p.OnCompletion(c.fun, c.executionReport)

			if metrics.Enabled && c.executionReport != nil {
				metrics.AddCompletedInvocation(c.fun.Name)
				if c.executionReport.SchedAction != SCHED_ACTION_OFFLOAD {
					metrics.AddFunctionDurationValue(c.fun.Name, c.executionReport.Duration)
				}
			}
		}
	}

}

// SubmitRequest submits a newly arrived request for scheduling and execution
func SubmitRequest(r *function.Request) (function.ExecutionReport, error) {
	schedRequest := scheduledRequest{
		Request:         r,
		decisionChannel: make(chan schedDecision, 1)}

	requests <- &schedRequest

	// wait on channel for scheduling action
	schedDecision, ok := <-schedRequest.decisionChannel
	if !ok {
		return function.ExecutionReport{}, fmt.Errorf("could not schedule the request")
	}
	//log.Printf("[%s] Scheduling decision: %v", r, schedDecision)

	if schedDecision.action == DROP {
		log.Printf("[%s] Dropping request", r)
		return function.ExecutionReport{}, node.OutOfResourcesErr
	} else if schedDecision.action == EXEC_REMOTE {
		log.Printf("Offloading request")
		return Offload(r, schedDecision.remoteHost)
	} else {
		return Execute(schedDecision.contID, &schedRequest, schedDecision.useWarm)
	}
}

// SubmitAsyncRequest submits a newly arrived async request for scheduling and execution
func SubmitAsyncRequest(r *function.Request) {
	schedRequest := scheduledRequest{
		Request:         r,
		decisionChannel: make(chan schedDecision, 1)}
	requests <- &schedRequest

	// wait on channel for scheduling action
	schedDecision, ok := <-schedRequest.decisionChannel
	if !ok {
		publishAsyncResponse(r.ReqId, function.Response{Success: false})
		return
	}

	var err error
	if schedDecision.action == DROP {
		publishAsyncResponse(r.ReqId, function.Response{Success: false})
	} else if schedDecision.action == EXEC_REMOTE {
		log.Printf("Offloading request")
		err = OffloadAsync(r, schedDecision.remoteHost)
		if err != nil {
			publishAsyncResponse(r.ReqId, function.Response{Success: false})
		}
	} else {
		report, err := Execute(schedDecision.contID, &schedRequest, schedDecision.useWarm)
		if err != nil {
			publishAsyncResponse(r.ReqId, function.Response{Success: false})
		}
		publishAsyncResponse(r.ReqId, function.Response{Success: true, ExecutionReport: report})
	}
}

func handleColdStart(r *scheduledRequest) (isSuccess bool) {
	newContainer, err := node.NewContainer(r.Fun, r.Istance_number)
	if errors.Is(err, node.OutOfResourcesErr) {
		return false
	} else if err != nil {
		log.Printf("Cold start failed: %v\n", err)
		return false
	} else {
		execLocally(r, newContainer, false)
		return true
	}
}

func dropRequest(r *scheduledRequest) {
	r.decisionChannel <- schedDecision{action: DROP}
}

func execLocally(r *scheduledRequest, c container.ContainerID, warmStart bool) {
	decision := schedDecision{action: EXEC_LOCAL, contID: c, useWarm: warmStart}
	r.decisionChannel <- decision
}

func handleOffload(r *scheduledRequest, serverHost string) {
	r.CanDoOffloading = false // the next server can't offload this request
	r.decisionChannel <- schedDecision{
		action:     EXEC_REMOTE,
		contID:     "",
		remoteHost: serverHost,
	}
}

func handleCloudOffload(r *scheduledRequest) {
	cloudAddress := config.GetString(config.CLOUD_URL, "")
	handleOffload(r, cloudAddress)
}

func handleUnavailableRunningContainer(r *scheduledRequest) (isSuccess bool) {

	log.Printf("attempt to acquire warm container after there are no running container\n")

	// If there are no running containers executing functions, take one from the warm pool (if any)
	containerID, err := node.AcquireWarmContainer(r.Fun, r.Istance_number, r.Fun.MaxFunctionInstances)
	if err == nil {
		execLocally(r, containerID, true)
		return true
	}

	if errors.Is(err, node.OutOfResourcesErr) {
		log.Printf("not enough resources for function execution into a warm container, the request will be enqueue if possible\n")
		return false
	}

	if errors.Is(err, node.NoWarmFoundErr) {
		return handleColdStart(r)
	}

	// other error
	return false
}
