package scheduling

import (
	"errors"
	"log"

	"github.com/grussorusso/serverledge/internal/function"

	"github.com/grussorusso/serverledge/internal/config"
	"github.com/grussorusso/serverledge/internal/node"
)

type DefaultLocalPolicy struct {
	queue queue
}

func (p *DefaultLocalPolicy) Init() {
	queueCapacity := config.GetInt(config.SCHEDULER_QUEUE_CAPACITY, 0)
	log.Printf("queue capacity: %d", queueCapacity)
	if queueCapacity > 0 {
		log.Printf("Configured queue with capacity %d\n", queueCapacity)
		p.queue = NewFIFOQueue(queueCapacity)
	} else {
		p.queue = nil
	}
}

func (p *DefaultLocalPolicy) OnCompletion(_ *function.Function, _ *function.ExecutionReport) {
	if p.queue == nil {
		return
	}

	p.queue.Lock()
	defer p.queue.Unlock()
	if p.queue.Len() == 0 {
		return
	}

	req := p.queue.Front()

	containerID, err := node.AcquireRunningContainer(req.Fun)
	switch {
	case err == nil:
		p.queue.Dequeue()
		log.Printf("[%s] start from the queue (length=%d)\n", req, p.queue.Len())
		execLocally(req, containerID, false) // use a running container
		return

	case errors.Is(err, node.NoRunningContErr):
		log.Printf("attempt to acquire resources\n")
		// If there are no running containers executing functions, take one from the warm pool
		if node.AcquireResources(req.Fun.CPUDemand, req.Fun.MemoryMB, true) {
			log.Printf("[%s] warm start from the queue\n", req)
			p.queue.Dequeue()

			// Get a warm container
			// This avoids blocking the thread during the cold
			// start, but also allows us to check for resource
			// availability before dequeueing
			go func() {
				warmContainer, err := node.WarmContainerWithAcquiredResources(req.Fun)
				if err != nil {
					log.Printf("there is an error when we are aqcuiring a warm container, the request will be dropped\n")
					dropRequest(req)
				} else {
					log.Printf("using a warm container")
					execLocally(req, warmContainer, true) // use a warm container
				}
			}()
			return
		}

	case errors.Is(err, node.OutOfResourcesErr):
		log.Printf("out of resources\n")
		// pass

	default:
		// Other error
		log.Printf("there is an error\n")
		p.queue.Dequeue()
		dropRequest(req)
	}
}

func (p *DefaultLocalPolicy) OnArrival(r *scheduledRequest) {

	containerID, err := node.AcquireRunningContainer(r.Fun)
	if err == nil {
		execLocally(r, containerID, false)
		return
	}

	if errors.Is(err, node.NoRunningContErr) {
		log.Printf("attempt to acquire warm container after there are no running container\n")
		// If there are no running containers executing functions, take one from the warm pool
		containerID, err := node.AcquireWarmContainer(r.Fun)

		if err == nil {
			execLocally(r, containerID, true)
			return
		}

		if errors.Is(err, node.OutOfResourcesErr) {
			// pass
			goto OutRes
		}

		if errors.Is(err, node.NoWarmFoundErr) {
			if handleColdStart(r) {
				return
			}
		}

		// other error
		dropRequest(r)
		return
	}

OutRes:
	if errors.Is(err, node.OutOfResourcesErr) {
		// pass
		log.Printf("not enough resources for function execution, the request will be enqueue if possible\n")
	}

	// enqueue if possible
	if p.queue != nil {
		p.queue.Lock()
		defer p.queue.Unlock()
		if p.queue.Enqueue(r) {
			log.Printf("[%s] Added to queue (length=%d)\n", r, p.queue.Len())
			return
		}
	}

	dropRequest(r)
}
