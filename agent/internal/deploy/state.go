package deploy

import "time"

type Phase string

const (
	PhasePending           Phase = "pending"
	PhaseBuilding          Phase = "building"
	PhaseContainersStarted Phase = "containers_started"
	PhaseHealthcheck       Phase = "healthcheck"
	PhaseSwapped           Phase = "swapped"
	PhaseDraining          Phase = "draining"
	PhaseSucceeded         Phase = "succeeded"
	PhaseFailed            Phase = "failed"
)

type ContainerRecord struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Service string `json:"service"`
}

type ServiceBuild struct {
	Name  string `json:"name"`
	Image string `json:"image"`
	Port  int    `json:"port"`
}

type State struct {
	ID              string            `json:"id"`
	Tenant          string            `json:"tenant"`
	Project         string            `json:"project"`
	BuildID         string            `json:"build_id"`
	Phase           Phase             `json:"phase"`
	Message         string            `json:"message,omitempty"`
	CurrentBuildID  string            `json:"current_build_id,omitempty"`
	PreviousBuildID string            `json:"previous_build_id,omitempty"`
	Services        []ServiceBuild    `json:"services,omitempty"`
	NewContainers   []ContainerRecord `json:"new_containers,omitempty"`
	DrainContainers []ContainerRecord `json:"drain_containers,omitempty"`
	UpdatedAt       time.Time         `json:"updated_at"`
}

func (s State) IsTerminal() bool {
	return s.Phase == PhaseSucceeded || s.Phase == PhaseFailed
}

func (s State) IsInFlight() bool {
	return !s.IsTerminal() && s.Phase != PhasePending
}
