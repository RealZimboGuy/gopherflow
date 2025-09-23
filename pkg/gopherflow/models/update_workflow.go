package models

import "time"

type UpdateStateVarRequest struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type UpdateStateVarResponse struct {
	OK bool `json:"ok"`
}

type UpdateWorkflowStateRequest struct {
	State          string     `json:"state"`
	NextActivation *time.Time `json:"nextActivation,omitempty"`
}
type UpdateWorkflowStateAndWaitRequest struct {
	UpdateWorkflowStateRequest UpdateWorkflowStateRequest `json:"updateWorkflowStateRequest"`
	UpdateStateVarRequest      UpdateStateVarRequest      `json:"updateStateVarRequest"`
	WaitSeconds                int                        `json:"waitSeconds"`
	CheckSeconds               int                        `json:"checkSeconds"`
	FromStates                 []string                   `json:"fromStates"`
	WaitForStates              []string                   `json:"waitForStates"`
}

type UpdateWorkflowStateResponse struct {
	OK bool `json:"ok"`
}
