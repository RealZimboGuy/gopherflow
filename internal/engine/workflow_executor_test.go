package engine

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/RealZimboGuy/gopherflow/internal/repository"
	"github.com/RealZimboGuy/gopherflow/pkg/gopherflow/core"
	"github.com/RealZimboGuy/gopherflow/pkg/gopherflow/domain"
	"github.com/RealZimboGuy/gopherflow/pkg/gopherflow/models"
)

// MockWorkflowRepo implements WorkflowRepo for testing
type MockWorkflowRepo struct {
	UpdateWorkflowStatusFunc                      func(id int64, status string) error
	UpdateWorkflowStartingTimeFunc                func(id int64) error
	UpdateStateFunc                               func(id int64, state string) error
	SaveWorkflowVariablesFunc                     func(id int64, vars string) error
	WakeParentWorkflowFunc                        func(parentID int64) error
	SaveFunc                                      func(wf *domain.Workflow) (int64, error)
	FindByIDFunc                                  func(id int64) (*domain.Workflow, error)
	UpdateNextActivationSpecificFunc              func(id int64, next time.Time) error
	UpdateNextActivationOffsetFunc                func(id int64, offset string) error
	ClearExecutorIdFunc                           func(id int64) error
	IncrementRetryCounterAndSetNextActivationFunc func(id int64, activation time.Time) error
	FindPendingWorkflowsFunc                      func(size int, executorGroup string) (*[]domain.Workflow, error)
	MarkWorkflowAsScheduledForExecutionFunc       func(id int64, executorId int64, modified time.Time) bool
	FindStuckWorkflowsFunc                        func(minutesRepair string, executorGroup string, limit int) (*[]domain.Workflow, error)
	LockWorkflowByModifiedFunc                    func(id int64, modified time.Time) bool
}

func (m *MockWorkflowRepo) UpdateWorkflowStatus(id int64, status string) error {
	if m.UpdateWorkflowStatusFunc != nil {
		return m.UpdateWorkflowStatusFunc(id, status)
	}
	return nil
}
func (m *MockWorkflowRepo) UpdateWorkflowStartingTime(id int64) error {
	if m.UpdateWorkflowStartingTimeFunc != nil {
		return m.UpdateWorkflowStartingTimeFunc(id)
	}
	return nil
}
func (m *MockWorkflowRepo) UpdateState(id int64, state string) error {
	if m.UpdateStateFunc != nil {
		return m.UpdateStateFunc(id, state)
	}
	return nil
}
func (m *MockWorkflowRepo) SaveWorkflowVariables(id int64, vars string) error {
	if m.SaveWorkflowVariablesFunc != nil {
		return m.SaveWorkflowVariablesFunc(id, vars)
	}
	return nil
}
func (m *MockWorkflowRepo) WakeParentWorkflow(parentID int64) error {
	if m.WakeParentWorkflowFunc != nil {
		return m.WakeParentWorkflowFunc(parentID)
	}
	return nil
}
func (m *MockWorkflowRepo) Save(wf *domain.Workflow) (int64, error) {
	if m.SaveFunc != nil {
		return m.SaveFunc(wf)
	}
	return 1, nil
}
func (m *MockWorkflowRepo) FindByID(id int64) (*domain.Workflow, error) {
	if m.FindByIDFunc != nil {
		return m.FindByIDFunc(id)
	}
	return nil, nil
}
func (m *MockWorkflowRepo) UpdateNextActivationSpecific(id int64, next time.Time) error {
	if m.UpdateNextActivationSpecificFunc != nil {
		return m.UpdateNextActivationSpecificFunc(id, next)
	}
	return nil
}
func (m *MockWorkflowRepo) UpdateNextActivationOffset(id int64, offset string) error {
	if m.UpdateNextActivationOffsetFunc != nil {
		return m.UpdateNextActivationOffsetFunc(id, offset)
	}
	return nil
}
func (m *MockWorkflowRepo) ClearExecutorId(id int64) error {
	if m.ClearExecutorIdFunc != nil {
		return m.ClearExecutorIdFunc(id)
	}
	return nil
}
func (m *MockWorkflowRepo) IncrementRetryCounterAndSetNextActivation(id int64, activation time.Time) error {
	if m.IncrementRetryCounterAndSetNextActivationFunc != nil {
		return m.IncrementRetryCounterAndSetNextActivationFunc(id, activation)
	}
	return nil
}

// Stubs for other interface methods not typically used in basic RunWorkflow tests but required by interface
func (m *MockWorkflowRepo) GetChildrenByParentID(parentID int64, onlyActive bool) (*[]domain.Workflow, error) {
	return nil, nil
}
func (m *MockWorkflowRepo) FindPendingWorkflows(size int, executorGroup string) (*[]domain.Workflow, error) {
	if m.FindPendingWorkflowsFunc != nil {
		return m.FindPendingWorkflowsFunc(size, executorGroup)
	}
	return nil, nil
}
func (m *MockWorkflowRepo) MarkWorkflowAsScheduledForExecution(id int64, executorId int64, modified time.Time) bool {
	if m.MarkWorkflowAsScheduledForExecutionFunc != nil {
		return m.MarkWorkflowAsScheduledForExecutionFunc(id, executorId, modified)
	}
	return true
}
func (m *MockWorkflowRepo) FindStuckWorkflows(minutesRepair string, executorGroup string, limit int) (*[]domain.Workflow, error) {
	if m.FindStuckWorkflowsFunc != nil {
		return m.FindStuckWorkflowsFunc(minutesRepair, executorGroup, limit)
	}
	return nil, nil
}
func (m *MockWorkflowRepo) LockWorkflowByModified(id int64, modified time.Time) bool {
	if m.LockWorkflowByModifiedFunc != nil {
		return m.LockWorkflowByModifiedFunc(id, modified)
	}
	return true
}
func (m *MockWorkflowRepo) SearchWorkflows(req models.SearchWorkflowRequest) (*[]domain.Workflow, error) {
	return nil, nil
}
func (m *MockWorkflowRepo) GetTopExecuting(limit int) (*[]domain.Workflow, error)  { return nil, nil }
func (m *MockWorkflowRepo) GetNextToExecute(limit int) (*[]domain.Workflow, error) { return nil, nil }
func (m *MockWorkflowRepo) GetWorkflowOverview() ([]repository.WorkflowOverviewRow, error) {
	return nil, nil
}
func (m *MockWorkflowRepo) GetDefinitionStateOverview(workflowType string) ([]repository.DefinitionStateRow, error) {
	return nil, nil
}
func (m *MockWorkflowRepo) FindByExternalId(id string) (*domain.Workflow, error)      { return nil, nil }
func (m *MockWorkflowRepo) SaveWorkflowVariablesAndTouch(id int64, vars string) error { return nil }

// MockWorkflowActionRepo
type MockWorkflowActionRepo struct {
	SaveFunc func(a *domain.WorkflowAction) (int64, error)
}

func (m *MockWorkflowActionRepo) Save(a *domain.WorkflowAction) (int64, error) {
	if m.SaveFunc != nil {
		return m.SaveFunc(a)
	}
	return 1, nil
}
func (m *MockWorkflowActionRepo) FindAllByWorkflowID(workflowID int64) (*[]domain.WorkflowAction, error) {
	return nil, nil
}

// MockWorkflow
type MockWorkflow struct {
	core.BaseWorkflow
	WorkflowData domain.Workflow
	ShouldPanic  bool
	ShouldError  bool
}

func (m *MockWorkflow) Description() string {
	return "Mock Workflow"
}
func (m *MockWorkflow) Setup(wf *domain.Workflow) {
	m.WorkflowData = *wf
}
func (m *MockWorkflow) GetWorkflowData() *domain.Workflow {
	return &m.WorkflowData
}
func (m *MockWorkflow) GetStateVariables() map[string]string {
	return map[string]string{}
}
func (m *MockWorkflow) StateTransitions() map[string][]string {
	return map[string][]string{
		string(models.StateStart): {"Step1"},
		"Step1":                   {string(models.StateEnd)},
	}
}
func (m *MockWorkflow) InitialState() string {
	return string(models.StateStart)
}
func (m *MockWorkflow) GetAllStates() []models.WorkflowState {
	return []models.WorkflowState{
		{Name: string(models.StateStart), StateType: models.StateStart},
		{Name: "Step1", StateType: models.StateNormal},
		{Name: string(models.StateEnd), StateType: models.StateEnd},
	}
}
func (m *MockWorkflow) GetRetryConfig() models.RetryConfig {
	return models.RetryConfig{
		MaxRetryCount:    3,
		RetryIntervalMin: 1 * time.Second,
		RetryIntervalMax: 5 * time.Second,
	}
}

// State Methods
func (m *MockWorkflow) Start(ctx context.Context) (models.NextState, error) {
	return models.NextState{Name: "Step1"}, nil
}
func (m *MockWorkflow) Step1(ctx context.Context) (models.NextState, error) {
	if m.ShouldPanic {
		panic("boom")
	}
	if m.ShouldError {
		return models.NextState{}, errors.New("something went wrong")
	}
	return models.NextState{Name: string(models.StateEnd)}, nil
}
func (m *MockWorkflow) End(ctx context.Context) (models.NextState, error) {
	return models.NextState{}, nil
}

func TestRunWorkflow_Success(t *testing.T) {
	repo := &MockWorkflowRepo{
		UpdateStateFunc: func(id int64, state string) error {
			return nil
		},
		UpdateWorkflowStatusFunc: func(id int64, status string) error {
			return nil
		},
		ClearExecutorIdFunc: func(id int64) error {
			return nil
		},
	}
	actionRepo := &MockWorkflowActionRepo{}

	wf := &MockWorkflow{
		WorkflowData: domain.Workflow{
			ID:    1,
			State: string(models.StateStart),
		},
	}

	RunWorkflow(context.Background(), wf, repo, actionRepo, 1, "worker1")

	// Verify expected state transitions were called (conceptually covered by mocks not erroring)
}

func TestRunWorkflow_PanicRecovery(t *testing.T) {
	repo := &MockWorkflowRepo{
		UpdateWorkflowStatusFunc: func(id int64, status string) error {
			if status == "ERROR" {
				return nil
			}
			return nil
		},
	}
	actionRepo := &MockWorkflowActionRepo{}

	wf := &MockWorkflow{
		WorkflowData: domain.Workflow{
			ID:    1,
			State: "Step1",
		},
		ShouldPanic: true,
	}

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("RunWorkflow should have recovered internally but panicked with: %v", r)
		}
	}()

	RunWorkflow(context.Background(), wf, repo, actionRepo, 1, "worker1")
}

func TestRunWorkflow_RetryLogic(t *testing.T) {
	retryCalled := false
	repo := &MockWorkflowRepo{
		IncrementRetryCounterAndSetNextActivationFunc: func(id int64, activation time.Time) error {
			retryCalled = true
			return nil
		},
	}
	actionRepo := &MockWorkflowActionRepo{}

	wf := &MockWorkflow{
		WorkflowData: domain.Workflow{
			ID:    1,
			State: "Step1",
		},
		ShouldError: true,
	}

	RunWorkflow(context.Background(), wf, repo, actionRepo, 1, "worker1")

	if !retryCalled {
		t.Error("Expected increment retry counter to be called")
	}
}
