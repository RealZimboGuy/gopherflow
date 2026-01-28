package engine

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/RealZimboGuy/gopherflow/pkg/gopherflow/core"
	"github.com/RealZimboGuy/gopherflow/pkg/gopherflow/domain"
)

// Reusing Mocks from workflow_executor_test.go where possible,
// but redefining MockExecutorRepo and MockDefinitionRepo here as they are unique to this test
// (or could act as shared mocks if I organizing them better, but local definition is fine)

type MockExecutorRepo struct {
	SaveFunc                     func(e *domain.Executor) (int64, error)
	UpdateLastActiveFunc         func(id int64, ts time.Time) error
	GetExecutorsByLastActiveFunc func(limit int) ([]*domain.Executor, error)
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

type MockDefinitionRepo struct {
	FindAllFunc    func() (*[]domain.WorkflowDefinition, error)
	FindByNameFunc func(name string) (*domain.WorkflowDefinition, error)
	SaveFunc       func(def *domain.WorkflowDefinition) error
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
	return nil, nil // Not found
}
func (m *MockDefinitionRepo) Save(def *domain.WorkflowDefinition) error {
	if m.SaveFunc != nil {
		return m.SaveFunc(def)
	}
	return nil
}

func TestWorkflowManager_ListWorkflowDefinitions(t *testing.T) {
	expectedDefs := []domain.WorkflowDefinition{
		{Name: "Def1"},
		{Name: "Def2"},
	}
	defRepo := &MockDefinitionRepo{
		FindAllFunc: func() (*[]domain.WorkflowDefinition, error) {
			return &expectedDefs, nil
		},
	}

	wm := NewWorkflowManager(nil, nil, nil, defRepo, nil, nil)
	defs, err := wm.ListWorkflowDefinitions()
	if err != nil {
		t.Fatalf("ListWorkflowDefinitions returned error: %v", err)
	}
	if len(*defs) != 2 {
		t.Errorf("Expected 2 definitions, got %d", len(*defs))
	}
}

func TestWorkflowManager_PollAndRunWorkflows(t *testing.T) {
	// Setup config
	os.Setenv("ENGINE_BATCH_SIZE", "10")
	os.Setenv("ENGINE_EXECUTOR_GROUP", "default")
	defer os.Unsetenv("ENGINE_BATCH_SIZE")
	defer os.Unsetenv("ENGINE_EXECUTOR_GROUP")

	// Initialize workflowQueue global variable
	workflowQueue = make(chan core.Workflow, 10)
	defer func() { close(workflowQueue) }()

	// Mocks
	wfRepo := &MockWorkflowRepo{
		FindPendingWorkflowsFunc: func(size int, executorGroup string) (*[]domain.Workflow, error) {
			// Return one pending workflow
			return &[]domain.Workflow{
				{ID: 1, WorkflowType: "MockWorkflow", BusinessKey: "key1"},
			}, nil
		},
		MarkWorkflowAsScheduledForExecutionFunc: func(id int64, executorId int64, modified time.Time) bool {
			return true
		},
	}
	waRepo := &MockWorkflowActionRepo{
		SaveFunc: func(a *domain.WorkflowAction) (int64, error) {
			return 1, nil
		},
	}

	// Mock Registry
	registry := map[string]func() core.Workflow{
		"MockWorkflow": func() core.Workflow {
			return &MockWorkflow{
				WorkflowData: domain.Workflow{ID: 1}, // Setup basic data
			}
		},
	}

	wm := NewWorkflowManager(wfRepo, waRepo, nil, nil, &registry, nil)
	wm.executorID = 123

	// Run poll
	wm.pollAndRunWorkflows(context.Background())

	// Check if workflow was added to queue
	select {
	case wf := <-workflowQueue:
		if wf.GetWorkflowData().ID != 1 {
			t.Errorf("Expected workflow ID 1, got %d", wf.GetWorkflowData().ID)
		}
	case <-time.After(1 * time.Second):
		t.Error("Timed out waiting for workflow in queue")
	}
}

func TestRegisterWorkflowDefinitions(t *testing.T) {
	// Setup registry with valid workflow
	registry := map[string]func() core.Workflow{
		"TestWorkflow": func() core.Workflow {
			return &MockWorkflow{
				WorkflowData: domain.Workflow{},
			}
		},
	}

	saveCalled := false
	defRepo := &MockDefinitionRepo{
		FindByNameFunc: func(name string) (*domain.WorkflowDefinition, error) {
			return nil, nil // Not found
		},
		SaveFunc: func(def *domain.WorkflowDefinition) error {
			if def.Name == "TestWorkflow" {
				saveCalled = true
			}
			return nil
		},
	}

	wm := NewWorkflowManager(nil, nil, nil, defRepo, &registry, nil)

	registerWorkflowDefinitions(context.Background(), wm)

	if !saveCalled {
		t.Error("Expected definition to be saved")
	}
}

// Ensure MockWorkflow satisfies core.Workflow (it was improved in workflow_executor_test.go)
// We rely on MockWorkflow being available in the package test build.
