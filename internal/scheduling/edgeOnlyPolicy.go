package scheduling

import (
	"log"

	"github.com/grussorusso/serverledge/internal/function"

	"github.com/grussorusso/serverledge/internal/node"
)

// EdgePolicy supports only Edge-Edge offloading
type EdgePolicy struct{}

func (p *EdgePolicy) Init() {
}

func (p *EdgePolicy) OnCompletion(_ *function.Function, _ *function.ExecutionReport) {

}

func (p *EdgePolicy) OnArrival(r *scheduledRequest) {
	if r.CanDoOffloading {
		url := pickEdgeNodeForOffloading(r)
		if url != "" {
			handleOffload(r, url)
			return
		}
	} else {
		containerID, err := node.AcquireRunningContainer(r.Fun, r.Istance_number)
		if err == nil {
			log.Printf("Using a warm container for: %v\n", r)
			execLocally(r, containerID, true)
		} else if handleColdStart(r) {
			return
		}
	}

	dropRequest(r)
}
