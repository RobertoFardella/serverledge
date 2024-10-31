package scheduling

import (
	"github.com/grussorusso/serverledge/internal/function"
	"github.com/grussorusso/serverledge/internal/node"
)

// CloudEdgePolicy supports only Edge-Cloud Offloading
type CloudEdgePolicy struct{}

func (p *CloudEdgePolicy) Init() {
}

func (p *CloudEdgePolicy) OnCompletion(_ *function.Function, _ *function.ExecutionReport) {

}

func (p *CloudEdgePolicy) OnArrival(r *scheduledRequest) { //TODO: qui va aggiunta la nuova logica
	containerID, err := node.AcquireRunningContainer(r.Fun, r.Istance_number)
	if err == nil {
		execLocally(r, containerID, false)
	} else if handleColdStart(r) {
		return
	} else if r.CanDoOffloading {
		handleCloudOffload(r)
	} else {
		dropRequest(r)
	}
}
