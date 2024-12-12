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
	if err == nil {
		p.queue.Dequeue()
		log.Printf("[%s] running container start from the queue (length=%d)\n", req, p.queue.Len())
		execLocally(req, containerID, true) // use a running container
		return
	}

	if errors.Is(err, node.NoRunningContErr) {
		// If there are no running containers executing functions, take one from the warm pool
		if node.AcquireResources(req.Fun.CPUDemand, req.Fun.MemoryMB, false) {
			p.queue.Dequeue()
			warmContainer, err := node.WarmContainerWithAcquiredResources(req.Fun)
			if err != nil { // cold start
				go func(req *scheduledRequest) {
					newContainer, err := node.NewContainerWithAcquiredResources(req.Fun)
					if err != nil {
						dropRequest(req)
					} else {
						log.Printf("[%s] cold start from the queue (length=%d)\n", req, p.queue.Len())
						execLocally(req, newContainer, false)
					}
				}(req)
				return

			} else { // warm container is ready
				log.Printf("[%s] warm start from the queue (length=%d)\n", req, p.queue.Len())
				execLocally(req, warmContainer, true)
			}

		}
	} else if errors.Is(err, node.OutOfResourcesErr) {
		// pass
	} else {
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

	if errors.Is(err, node.OutOfResourcesErr) {
		// pass
	}

	if errors.Is(err, node.NoRunningContErr) {
		if handleUnavailableRunningContainer(r) {
			return
		}
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

	dropRequest(r) //if the Enqueue operation is not succeed, drop the request

}
