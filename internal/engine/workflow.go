package engine

import (
	"github.com/RealZimboGuy/gopherflow/internal/domain"
	"github.com/RealZimboGuy/gopherflow/internal/models"
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
