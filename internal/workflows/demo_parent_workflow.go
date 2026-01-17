package workflows

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/RealZimboGuy/gopherflow/pkg/gopherflow"
	"github.com/RealZimboGuy/gopherflow/pkg/gopherflow/core"
	domain "github.com/RealZimboGuy/gopherflow/pkg/gopherflow/domain"
	"github.com/RealZimboGuy/gopherflow/pkg/gopherflow/models"
)

// State constants for parent workflow
const (
	ParentInit            = "ParentInit"
	ParentSpawnChildren   = "ParentSpawnChildren"
	ParentWaitForChildren = "ParentWaitForChildren"
	ParentFinish          = "ParentFinish"
)

// DemoParentWorkflow is a demonstration of a parent workflow that creates child workflows
type DemoParentWorkflow struct {
	core.BaseWorkflow
	Clock      core.Clock
	numCreated int
}

func (w *DemoParentWorkflow) Setup(wf *domain.Workflow) {
	w.BaseWorkflow.Setup(wf)
}

func (w *DemoParentWorkflow) GetWorkflowData() *domain.Workflow {
	return w.WorkflowState
}

func (w *DemoParentWorkflow) GetStateVariables() map[string]string {
	return w.StateVariables
}

func (w *DemoParentWorkflow) InitialState() string {
	return ParentInit
}

func (w *DemoParentWorkflow) Description() string {
	return "Demo parent workflow that spawns and coordinates child workflows"
}

func (w *DemoParentWorkflow) StateTransitions() map[string][]string {
	return map[string][]string{
		ParentInit:            {ParentSpawnChildren},
		ParentSpawnChildren:   {ParentWaitForChildren},
		ParentWaitForChildren: {ParentFinish, ParentWaitForChildren},
	}
}

func (w *DemoParentWorkflow) GetAllStates() []models.WorkflowState {
	return []models.WorkflowState{
		{Name: ParentInit, StateType: models.StateStart},
		{Name: ParentSpawnChildren, StateType: models.StateNormal},
		{Name: ParentWaitForChildren, StateType: models.StateNormal},
		{Name: ParentFinish, StateType: models.StateEnd},
	}
}

func (w *DemoParentWorkflow) GetRetryConfig() models.RetryConfig {
	return models.RetryConfig{
		MaxRetryCount:    3,
		RetryIntervalMin: time.Second * 1,
		RetryIntervalMax: time.Second * 5,
	}
}

// GetChildWorkflows implements the ParentChildCapable interface
func (w *DemoParentWorkflow) GetChildWorkflows(ctx context.Context) ([]domain.Workflow, error) {
	slog.InfoContext(ctx, "Getting child workflows for parent", "parent_id", w.WorkflowState.ID, "count", len(w.ChildWorkflows))
	return w.ChildWorkflows, nil
}

// WakeParent implements the ParentChildCapable interface
func (w *DemoParentWorkflow) WakeParent(ctx context.Context) error {
	// This shouldn't be called on a parent workflow, but we implement it for interface compliance
	slog.InfoContext(ctx, "WakeParent called on parent workflow, this is unexpected")
	return nil
}

// ParentInit initializes the parent workflow
func (w *DemoParentWorkflow) ParentInit(ctx context.Context) (*models.NextState, error) {
	slog.InfoContext(ctx, "Initializing parent workflow", "workflow_id", w.WorkflowState.ID)

	return &models.NextState{
		Name:      ParentSpawnChildren,
		ActionLog: "Parent workflow initialized",
	}, nil
}

// ParentSpawnChildren creates two child workflows
func (w *DemoParentWorkflow) ParentSpawnChildren(ctx context.Context) (*models.NextState, error) {
	slog.InfoContext(ctx, "Spawning child workflows", "workflow_id", w.WorkflowState.ID)

	// Create child workflow requests - two children with different task durations
	childRequests := []models.ChildWorkflowRequest{
		gopherflow.CreateChildWorkflowRequest(
			"DemoChildWorkflow", // Workflow type
			fmt.Sprintf("child-1-of-%d", w.WorkflowState.ID), // Business key
			map[string]string{"sleepTime": "3"},              // State variables - 3 second sleep
		),
		gopherflow.CreateChildWorkflowRequest(
			"DemoChildWorkflow", // Workflow type
			fmt.Sprintf("child-2-of-%d", w.WorkflowState.ID), // Business key
			map[string]string{"sleepTime": "15"},             // State variables - 5 second sleep
		),
	}

	// Store the number of children we created for later reference
	w.numCreated = len(childRequests)
	w.StateVariables["children_count"] = fmt.Sprintf("%d", w.numCreated)

	return &models.NextState{
		Name:                ParentWaitForChildren,
		ActionLog:           fmt.Sprintf("Spawned %d child workflows", w.numCreated),
		ChildWorkflows:      childRequests,
		NextExecutionOffset: "10 minutes",
	}, nil
}

// ParentWaitForChildren checks if all children are complete
func (w *DemoParentWorkflow) ParentWaitForChildren(ctx context.Context) (*models.NextState, error) {
	slog.InfoContext(ctx, "Waiting for child workflows to complete", "workflow_id", w.WorkflowState.ID)

	// Get all child workflows
	children, err := w.GetChildWorkflows(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get child workflows: %w", err)
	}

	// Count how many children are complete
	completedCount := 0
	for _, child := range children {
		if child.Status == "FINISHED" {
			completedCount++
			slog.InfoContext(ctx, "Child workflow is complete",
				"child_id", child.ID,
				"business_key", child.BusinessKey)
		} else {
			slog.InfoContext(ctx, "Child workflow is still running",
				"child_id", child.ID,
				"business_key", child.BusinessKey,
				"status", child.Status)
		}
	}

	// Parse children_count from state variables
	expectedChildren := 2 // Default if we can't parse
	if countStr, ok := w.StateVariables["children_count"]; ok {
		if count, err := fmt.Sscanf(countStr, "%d", &expectedChildren); err != nil || count != 1 {
			// If there's an error parsing, just use our default
			expectedChildren = 2
		}
	}

	// If not all children are complete, wait and check again later
	if completedCount < expectedChildren {
		slog.InfoContext(ctx, "Not all children complete, waiting",
			"completed", completedCount,
			"expected", expectedChildren)

		// We'll check again in 5 seconds
		return &models.NextState{
			Name:                ParentWaitForChildren,
			ActionLog:           fmt.Sprintf("Waiting for children: %d/%d complete", completedCount, expectedChildren),
			NextExecutionOffset: "10 minutes",
		}, nil
	}

	// All children are complete, collect their results
	slog.InfoContext(ctx, "All children complete, proceeding to finish")

	// In a real workflow, you might want to process results from children here
	// For this demo, we just move to finish

	return &models.NextState{
		Name:      ParentFinish,
		ActionLog: "All child workflows complete",
	}, nil
}

// ParentFinish finishes the workflow
func (w *DemoParentWorkflow) ParentFinish(ctx context.Context) (*models.NextState, error) {
	slog.InfoContext(ctx, "Parent workflow finishing", "workflow_id", w.WorkflowState.ID)

	// In a real workflow, you might want to do some final processing here

	return &models.NextState{
		Name:      "END", // Special state name that marks workflow completion
		ActionLog: "Parent workflow complete",
	}, nil
}
