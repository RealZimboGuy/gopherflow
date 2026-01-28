package engine

import (
	"database/sql"
	"time"

	"github.com/RealZimboGuy/gopherflow/internal/repository"
	"github.com/RealZimboGuy/gopherflow/pkg/gopherflow/domain"
	"github.com/RealZimboGuy/gopherflow/pkg/gopherflow/models"
)

// WorkflowRepo defines the interface for workflow persistence, matching repository.WorkflowRepository.
type WorkflowRepo interface {
	GetChildrenByParentID(parentID int64, onlyActive bool) (*[]domain.Workflow, error)
	UpdateWorkflowStatus(id int64, status string) error
	UpdateWorkflowStartingTime(id int64) error
	UpdateState(id int64, state string) error
	SaveWorkflowVariables(id int64, vars string) error
	WakeParentWorkflow(parentID int64) error
	Save(wf *domain.Workflow) (int64, error)
	FindByID(id int64) (*domain.Workflow, error)
	UpdateNextActivationSpecific(id int64, next time.Time) error
	UpdateNextActivationOffset(id int64, offset string) error
	ClearExecutorId(id int64) error
	IncrementRetryCounterAndSetNextActivation(id int64, activation time.Time) error
	FindPendingWorkflows(size int, executorGroup string) (*[]domain.Workflow, error)
	MarkWorkflowAsScheduledForExecution(id int64, executorId int64, modified time.Time) bool
	FindStuckWorkflows(minutesRepair string, executorGroup string, limit int) (*[]domain.Workflow, error)
	LockWorkflowByModified(id int64, modified time.Time) bool
	SearchWorkflows(req models.SearchWorkflowRequest) (*[]domain.Workflow, error)
	GetTopExecuting(limit int) (*[]domain.Workflow, error)
	GetNextToExecute(limit int) (*[]domain.Workflow, error)
	GetWorkflowOverview() ([]repository.WorkflowOverviewRow, error)
	GetDefinitionStateOverview(workflowType string) ([]repository.DefinitionStateRow, error)
	FindByExternalId(id string) (*domain.Workflow, error)
	SaveWorkflowVariablesAndTouch(id int64, vars string) error
}

// WorkflowActionRepo defines the interface for workflow action persistence.
type WorkflowActionRepo interface {
	Save(a *domain.WorkflowAction) (int64, error)
	FindAllByWorkflowID(workflowID int64) (*[]domain.WorkflowAction, error)
}

// ExecutorRepo defines the interface for executor persistence.
type ExecutorRepo interface {
	Save(e *domain.Executor) (int64, error)
	UpdateLastActive(id int64, ts time.Time) error
	GetExecutorsByLastActive(limit int) ([]*domain.Executor, error)
}

// DefinitionRepo defines the interface for workflow definition persistence.
type DefinitionRepo interface {
	FindAll() (*[]domain.WorkflowDefinition, error)
	FindByName(name string) (*domain.WorkflowDefinition, error)
	Save(def *domain.WorkflowDefinition) error
}

// UserRepo defines the interface for user persistence.
type UserRepo interface {
	FindBySessionID(sessionID string, now time.Time) (*domain.User, error)
	FindByApiKey(apiKey string) (*domain.User, error)
	FindAll() (*[]domain.User, error)
	Save(user *domain.User) (int64, error)
	FindById(id int64) (*domain.User, error)
	DeleteById(id int64) error
	FindByUsername(username string) (*domain.User, error)
	UpdateSession(userID int64, sessionID string, expiry time.Time) error
	ClearSessionBySessionID(sessionID string) error
	UpdateUser(id int64, username string, apiKey sql.NullString, enabled sql.NullBool) error
}
