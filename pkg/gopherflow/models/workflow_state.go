package models

// WorkflowState declares one state of a workflow. Every state named in
// StateTransitions must appear in GetAllStates, and at least one state must be
// of type StateEnd so the workflow can finish.
type WorkflowState struct {
	Name      string    // Name of the state
	StateType StateType // Type of the state (e.g., Start, Normal, End)
}
