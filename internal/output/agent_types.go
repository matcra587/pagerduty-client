package output

// AgentRef is a compact reference to a PagerDuty resource.
type AgentRef struct {
	ID      string `json:"id"`
	Summary string `json:"summary"`
}
