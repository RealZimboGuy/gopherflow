package core

import (
	"encoding/json"
	"log/slog"

	domain "github.com/RealZimboGuy/gopherflow/pkg/gopherflow/domain"
)

// BaseWorkflow holds common workflow state and provides shared setup logic.
type BaseWorkflow struct {
	StateVariables map[string]string
	WorkflowState  *domain.Workflow
	ChildWorkflows []domain.Workflow
}

// Setup initializes the base workflow with the given workflow instance and parses state variables from JSON, if present.
func (b *BaseWorkflow) Setup(wf *domain.Workflow) {
	b.WorkflowState = wf
	if b.StateVariables == nil {
		b.StateVariables = make(map[string]string)
	} else {
		// ensure we start from existing vars but don't nil panic
	}
	// if there are state vars then try parse them to have loaded in
	if wf.StateVars.Valid && wf.StateVars.String != "" && wf.StateVars.String != "null" {
		// Reset map before loading to avoid carrying stale values when reusing instance
		b.StateVariables = make(map[string]string)
		if err := json.Unmarshal([]byte(wf.StateVars.String), &b.StateVariables); err != nil {
			slog.Error("Error parsing state vars", "error", err)
		}
	}
}
func (b *BaseWorkflow) SetChildWorkflows(children []domain.Workflow) {
	b.ChildWorkflows = children
}
