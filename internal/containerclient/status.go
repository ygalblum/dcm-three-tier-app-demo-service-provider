package containerclient

import (
	"strings"

	"github.com/dcm-project/3-tier-demo-service-provider/api/v1alpha1"
)

// WorstStatusFromPodmanStates returns the worst status given 3 Podman container states.
// States: "running", "exited", "created", "paused", "dead", "removing", etc.
// Order: FAILED > PENDING > RUNNING. Returns FAILED if len(states) != 3.
func WorstStatusFromPodmanStates(states []string) (v1alpha1.StackStatus, bool) {
	if len(states) != 3 {
		return v1alpha1.FAILED, true
	}
	worst := v1alpha1.RUNNING
	for _, state := range states {
		switch strings.TrimSpace(strings.ToLower(state)) {
		case "running":
			// keep current worst
		case "created", "paused":
			if worst == v1alpha1.RUNNING {
				worst = v1alpha1.PENDING
			}
		default:
			// exited, dead, removing, etc.
			return v1alpha1.FAILED, true
		}
	}
	return worst, true
}
