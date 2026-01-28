package controllers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/RealZimboGuy/gopherflow/internal/engine"
	"github.com/RealZimboGuy/gopherflow/internal/repository"
	"github.com/RealZimboGuy/gopherflow/pkg/gopherflow/domain"
	"github.com/RealZimboGuy/gopherflow/pkg/gopherflow/models"
)

// Mock repos for Controller tests (implementing engine.WorkflowRepo/ActionRepo and engine.DefinitionRepo)

type MockWorkflowRepo struct {
	FindByIDFunc func(id int64) (*domain.Workflow, error)
	// Add other methods if needed by controller
	GetWorkflowOverviewFunc        func() ([]repository.WorkflowOverviewRow, error)
	GetDefinitionStateOverviewFunc func(workflowType string) ([]repository.DefinitionStateRow, error)
	GetTopExecutingFunc            func(limit int) (*[]domain.Workflow, error)
	GetNextToExecuteFunc           func(limit int) (*[]domain.Workflow, error)
}

// Implement engine.WorkflowRepo - using panic or no-op for unused methods
func (m *MockWorkflowRepo) FindByID(id int64) (*domain.Workflow, error) {
	if m.FindByIDFunc != nil {
		return m.FindByIDFunc(id)
	}
	return nil, nil
}
func (m *MockWorkflowRepo) GetWorkflowOverview() ([]repository.WorkflowOverviewRow, error) {
	if m.GetWorkflowOverviewFunc != nil {
		return m.GetWorkflowOverviewFunc()
	}
	return nil, nil
}
func (m *MockWorkflowRepo) GetDefinitionStateOverview(workflowType string) ([]repository.DefinitionStateRow, error) {
	if m.GetDefinitionStateOverviewFunc != nil {
		return m.GetDefinitionStateOverviewFunc(workflowType)
	}
	return nil, nil
}
func (m *MockWorkflowRepo) GetTopExecuting(limit int) (*[]domain.Workflow, error) {
	if m.GetTopExecutingFunc != nil {
		return m.GetTopExecutingFunc(limit)
	}
	return nil, nil
}
func (m *MockWorkflowRepo) GetNextToExecute(limit int) (*[]domain.Workflow, error) {
	if m.GetNextToExecuteFunc != nil {
		return m.GetNextToExecuteFunc(limit)
	}
	return nil, nil
}

// Stubs for others
func (m *MockWorkflowRepo) GetChildrenByParentID(parentID int64, onlyActive bool) (*[]domain.Workflow, error) {
	return nil, nil
}
func (m *MockWorkflowRepo) UpdateWorkflowStatus(id int64, status string) error          { return nil }
func (m *MockWorkflowRepo) UpdateWorkflowStartingTime(id int64) error                   { return nil }
func (m *MockWorkflowRepo) UpdateState(id int64, state string) error                    { return nil }
func (m *MockWorkflowRepo) SaveWorkflowVariables(id int64, vars string) error           { return nil }
func (m *MockWorkflowRepo) WakeParentWorkflow(parentID int64) error                     { return nil }
func (m *MockWorkflowRepo) Save(wf *domain.Workflow) (int64, error)                     { return 1, nil }
func (m *MockWorkflowRepo) UpdateNextActivationSpecific(id int64, next time.Time) error { return nil }
func (m *MockWorkflowRepo) UpdateNextActivationOffset(id int64, offset string) error    { return nil }
func (m *MockWorkflowRepo) ClearExecutorId(id int64) error                              { return nil }
func (m *MockWorkflowRepo) IncrementRetryCounterAndSetNextActivation(id int64, activation time.Time) error {
	return nil
}
func (m *MockWorkflowRepo) FindPendingWorkflows(size int, executorGroup string) (*[]domain.Workflow, error) {
	return nil, nil
}
func (m *MockWorkflowRepo) MarkWorkflowAsScheduledForExecution(id int64, executorId int64, modified time.Time) bool {
	return true
}
func (m *MockWorkflowRepo) FindStuckWorkflows(minutesRepair string, executorGroup string, limit int) (*[]domain.Workflow, error) {
	return nil, nil
}
func (m *MockWorkflowRepo) LockWorkflowByModified(id int64, modified time.Time) bool { return true }
func (m *MockWorkflowRepo) SearchWorkflows(req models.SearchWorkflowRequest) (*[]domain.Workflow, error) {
	return nil, nil
}
func (m *MockWorkflowRepo) FindByExternalId(id string) (*domain.Workflow, error)      { return nil, nil }
func (m *MockWorkflowRepo) SaveWorkflowVariablesAndTouch(id int64, vars string) error { return nil }

type MockWorkflowActionRepo struct{
	FindAllByWorkflowIDFunc func(workflowID int64) (*[]domain.WorkflowAction, error)
	SaveFunc func(a *domain.WorkflowAction) (int64, error)
}

func (m *MockWorkflowActionRepo) Save(a *domain.WorkflowAction) (int64, error) { 
	if m.SaveFunc != nil {
		return m.SaveFunc(a)
	}
	return 1, nil 
}
func (m *MockWorkflowActionRepo) FindAllByWorkflowID(workflowID int64) (*[]domain.WorkflowAction, error) {
	if m.FindAllByWorkflowIDFunc != nil {
		return m.FindAllByWorkflowIDFunc(workflowID)
	}
	return nil, nil
}

type MockDefinitionRepo struct {
	FindAllFunc    func() (*[]domain.WorkflowDefinition, error)
	FindByNameFunc func(name string) (*domain.WorkflowDefinition, error)
}

func (m *MockDefinitionRepo) FindAll() (*[]domain.WorkflowDefinition, error) {
	if m.FindAllFunc != nil {
		return m.FindAllFunc()
	}
	return nil, nil
}
func (m *MockDefinitionRepo) FindByName(name string) (*domain.WorkflowDefinition, error) {
	if m.FindByNameFunc != nil {
		return m.FindByNameFunc(name)
	}
	return nil, nil
}
func (m *MockDefinitionRepo) Save(def *domain.WorkflowDefinition) error { return nil }

type MockExecutorRepo struct{
	GetExecutorsByLastActiveFunc func(limit int) ([]*domain.Executor, error)
	SaveFunc func(e *domain.Executor) (int64, error)
	UpdateLastActiveFunc func(id int64, ts time.Time) error
}

func (m *MockExecutorRepo) Save(e *domain.Executor) (int64, error) {
	if m.SaveFunc != nil {
		return m.SaveFunc(e)
	}
	return 1, nil 
}
func (m *MockExecutorRepo) UpdateLastActive(id int64, ts time.Time) error { 
	if m.UpdateLastActiveFunc != nil {
		return m.UpdateLastActiveFunc(id, ts)
	}
	return nil 
}
func (m *MockExecutorRepo) GetExecutorsByLastActive(limit int) ([]*domain.Executor, error) {
	if m.GetExecutorsByLastActiveFunc != nil {
		return m.GetExecutorsByLastActiveFunc(limit)
	}
	return nil, nil
}

func TestWorkflowsController_ListWorkflowDefinitions(t *testing.T) {
	defRepo := &MockDefinitionRepo{
		FindAllFunc: func() (*[]domain.WorkflowDefinition, error) {
			return &[]domain.WorkflowDefinition{
				{Name: "W1", Description: "Desc1"},
			}, nil
		},
	}
	wm := engine.NewWorkflowManager(
		&MockWorkflowRepo{},
		&MockWorkflowActionRepo{},
		&MockExecutorRepo{},
		defRepo,
		nil, nil,
	)

	c := NewWorkflowsController(&MockWorkflowRepo{}, &MockWorkflowActionRepo{}, wm, nil)

	req := httptest.NewRequest("GET", "/workflows", nil)
	w := httptest.NewRecorder()

	c.handleListWorkflowDefinitions(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var defs []domain.WorkflowDefinition
	if err := json.NewDecoder(resp.Body).Decode(&defs); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	if len(defs) != 1 {
		t.Errorf("Expected 1 definition, got %d", len(defs))
	}
	if defs[0].Name != "W1" {
		t.Errorf("Expected name W1, got %s", defs[0].Name)
	}
}
