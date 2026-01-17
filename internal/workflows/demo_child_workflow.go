package workflows

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/RealZimboGuy/gopherflow/pkg/gopherflow/core"
	domain "github.com/RealZimboGuy/gopherflow/pkg/gopherflow/domain"
	"github.com/RealZimboGuy/gopherflow/pkg/gopherflow/models"
)

// State constants for child workflow
const (
	ChildInit       = "ChildInit"
	ChildProcessing = "ChildProcessing"
	ChildWakeParent = "ChildWakeParent"
	ChildFinish     = "ChildFinish"
)

// DemoChildWorkflow is a demonstration of a child workflow that wakes its parent
type DemoChildWorkflow struct {
	core.BaseWorkflow
	Clock core.Clock
}

func (w *DemoChildWorkflow) Setup(wf *domain.Workflow) {
	w.BaseWorkflow.Setup(wf)
}

func (w *DemoChildWorkflow) GetWorkflowData() *domain.Workflow {
	return w.WorkflowState
}

func (w *DemoChildWorkflow) GetStateVariables() map[string]string {
	return w.StateVariables
}

func (w *DemoChildWorkflow) InitialState() string {
	return ChildInit
}

func (w *DemoChildWorkflow) Description() string {
	return "Demo child workflow that performs a task and wakes its parent"
}

func (w *DemoChildWorkflow) StateTransitions() map[string][]string {
	return map[string][]string{
		ChildInit:       {ChildProcessing},
		ChildProcessing: {ChildWakeParent},
		ChildWakeParent: {ChildFinish},
	}
}

func (w *DemoChildWorkflow) GetAllStates() []models.WorkflowState {
	return []models.WorkflowState{
		{Name: ChildInit, StateType: models.StateStart},
		{Name: ChildProcessing, StateType: models.StateNormal},
		{Name: ChildWakeParent, StateType: models.StateNormal},
		{Name: ChildFinish, StateType: models.StateEnd},
	}
}

func (w *DemoChildWorkflow) GetRetryConfig() models.RetryConfig {
	return models.RetryConfig{
		MaxRetryCount:    3,
		RetryIntervalMin: time.Second * 1,
		RetryIntervalMax: time.Second * 5,
	}
}

// GetChildWorkflows implements the ParentChildCapable interface
func (w *DemoChildWorkflow) GetChildWorkflows(ctx context.Context) ([]domain.Workflow, error) {
	// Child workflows don't have children, but we implement this for interface compliance
	slog.InfoContext(ctx, "GetChildWorkflows called on child workflow, this is unexpected")
	return []domain.Workflow{}, nil
}

//// WakeParent implements the ParentChildCapable interface
//func (w *DemoChildWorkflow) WakeParent(ctx context.Context) error {
//	// In a real implementation, this would call the repository to wake the parent
//	// But for this demo, we'll just log
//	slog.InfoContext(ctx, "Child workflow waking parent",
//		"child_id", w.WorkflowState.ID,
//		"parent_id", w.WorkflowState.ParentWorkflowID.Int64)
//
//	// In actual implementation, this would be:
//	// return repo.WakeParentWorkflow(w.WorkflowState.ParentWorkflowID.Int64)
//
//	// For demo we just pretend it worked
//	return nil
//}

// ChildInit initializes the child workflow
func (w *DemoChildWorkflow) ChildInit(ctx context.Context) (*models.NextState, error) {
	slog.InfoContext(ctx, "Initializing child workflow", "workflow_id", w.WorkflowState.ID)

	// Log the parent ID if available
	if w.WorkflowState.ParentWorkflowID.Valid {
		slog.InfoContext(ctx, "Child has parent", "parent_id", w.WorkflowState.ParentWorkflowID.Int64)
	}

	return &models.NextState{
		Name:      ChildProcessing,
		ActionLog: "Child workflow initialized",
	}, nil
}

// ChildProcessing performs the child workflow's task (simulated with a delay)
func (w *DemoChildWorkflow) ChildProcessing(ctx context.Context) (*models.NextState, error) {
	slog.InfoContext(ctx, "Child workflow processing", "workflow_id", w.WorkflowState.ID)

	// Get the sleep time from state variables, default to 3 seconds
	sleepTime := 3
	if sleepTimeStr, ok := w.StateVariables["sleepTime"]; ok {
		if parsed, err := strconv.Atoi(sleepTimeStr); err == nil && parsed > 0 {
			sleepTime = parsed
		}
	}

	slog.InfoContext(ctx, "Child workflow will sleep",
		"workflow_id", w.WorkflowState.ID,
		"sleep_seconds", sleepTime)

	// Store result and add sleep time for demonstration
	w.StateVariables["result"] = fmt.Sprintf("Processed after %d seconds", sleepTime)

	// In a real workflow, you would do actual processing here
	// For demo, we'll just set the next execution to be in the future
	return &models.NextState{
		Name:          ChildWakeParent,
		ActionLog:     fmt.Sprintf("Child workflow processed for %d seconds", sleepTime),
		NextExecution: w.Clock.Now().Add(time.Duration(sleepTime) * time.Second),
	}, nil
}

// ChildWakeParent wakes the parent workflow
func (w *DemoChildWorkflow) ChildWakeParent(ctx context.Context) (*models.NextState, error) {
	slog.InfoContext(ctx, "Child workflow attempting to wake parent", "workflow_id", w.WorkflowState.ID)

	// Only try to wake parent if we have a valid parent ID
	if !w.WorkflowState.ParentWorkflowID.Valid {
		slog.InfoContext(ctx, "Child workflow has no parent to wake", "workflow_id", w.WorkflowState.ID)
	}

	return &models.NextState{
		Name:       ChildFinish,
		ActionLog:  "Parent workflow awakened",
		WakeParent: true,
	}, nil
}

// ChildFinish finishes the child workflow
func (w *DemoChildWorkflow) ChildFinish(ctx context.Context) (*models.NextState, error) {
	slog.InfoContext(ctx, "Child workflow finishing", "workflow_id", w.WorkflowState.ID)

	return &models.NextState{
		Name:      "END", // Special state name that marks workflow completion
		ActionLog: "Child workflow complete",
	}, nil
}
