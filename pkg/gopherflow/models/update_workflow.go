package models

import "time"

// UpdateStateVarRequest sets a single state variable on a workflow.
type UpdateStateVarRequest struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// UpdateStateVarResponse reports whether a state variable update succeeded.
type UpdateStateVarResponse struct {
	OK bool `json:"ok"`
}

// UpdateWorkflowStateRequest moves a workflow to a new state, optionally
// scheduling when it should next run.
type UpdateWorkflowStateRequest struct {
	State          string     `json:"state"`
	NextActivation *time.Time `json:"nextActivation,omitempty"`
}

// UpdateWorkflowStateAndWaitRequest moves a workflow to a new state and then
// blocks until it reaches one of WaitForStates or WaitSeconds elapses. FromStates,
// when set, requires the workflow to currently be in one of those states,
// which makes the update safe against a concurrent transition.
type UpdateWorkflowStateAndWaitRequest struct {
	UpdateWorkflowStateRequest UpdateWorkflowStateRequest `json:"updateWorkflowStateRequest"`
	UpdateStateVarRequest      UpdateStateVarRequest      `json:"updateStateVarRequest"`
	WaitSeconds                int                        `json:"waitSeconds"`
	CheckSeconds               int                        `json:"checkSeconds"`
	FromStates                 []string                   `json:"fromStates"`
	WaitForStates              []string                   `json:"waitForStates"`
}

// UpdateWorkflowStateResponse reports whether a state update succeeded.
type UpdateWorkflowStateResponse struct {
	OK bool `json:"ok"`
}
