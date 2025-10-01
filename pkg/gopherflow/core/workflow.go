package core

import (
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
