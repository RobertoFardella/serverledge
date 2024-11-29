package scheduling

import (
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
		containerID, err := node.AcquireRunningContainer(r.Fun)
		if err == nil {
			execLocally(r, containerID, true)
		} else if handleUnavailableRunningContainer(r) {
			return
		}
	}

	dropRequest(r)
}
