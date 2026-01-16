package core

import (
	"context"

	"github.com/RealZimboGuy/gopherflow/pkg/gopherflow/domain"
	models "github.com/RealZimboGuy/gopherflow/pkg/gopherflow/models"
)

// Define a named string type

// Workflow is the interface that all workflows must implement.
type Workflow interface {
	StateTransitions() map[string][]string // map of state name -> list of next state names
	InitialState() string
	Description() string
	Setup(wf *domain.Workflow)
	GetWorkflowData() *domain.Workflow
	GetStateVariables() map[string]string
	GetAllStates() []models.WorkflowState // where to start
	GetRetryConfig() models.RetryConfig
}

// ParentChildCapable is an optional interface that workflows can implement
// to interact with child workflows.
type ParentChildCapable interface {
	// GetChildWorkflows retrieves all child workflows for this workflow
	GetChildWorkflows(ctx context.Context) ([]domain.Workflow, error)
	
	// WakeParent wakes up the parent workflow if this workflow is a child
	WakeParent(ctx context.Context) error
}
