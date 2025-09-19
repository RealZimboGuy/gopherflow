package models

type WorkflowState struct {
	Name      string    // Name of the state
	StateType StateType // Type of the state (e.g., Start, Normal, End)
}
