package common

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"github.com/RealZimboGuy/gopherflow/pkg/gopherflow"
	"github.com/RealZimboGuy/gopherflow/pkg/gopherflow/core"
	domain "github.com/RealZimboGuy/gopherflow/pkg/gopherflow/domain"
	models "github.com/RealZimboGuy/gopherflow/pkg/gopherflow/models"
)

// State constants for parent workflow
const (
	ParentInit                    = "ParentInit"
	ParentSpawnChildren           = "ParentSpawnChildren"
	ParentWaitForChildren         = "ParentWaitForChildren"
	ParentProcessChildrenResults  = "ParentProcessChildrenResults"
	ParentFinish                  = "ParentFinish"
)

// State constants for child workflow
const (
	ChildInit                     = "ChildInit"
	ChildProcessing               = "ChildProcessing"
	ChildWakeParent               = "ChildWakeParent"
	ChildFinish                   = "ChildFinish"
)

// ParentWorkflow represents a workflow that spawns child workflows
type ParentWorkflow struct {
	core.BaseWorkflow
	repo       *TestRepository
	numCreated int
}

// ChildWorkflow represents a child workflow
type ChildWorkflow struct {
	core.BaseWorkflow
	repo       *TestRepository
}

// TestRepository is a mock repository for testing purposes
type TestRepository struct {
	ParentWoken  bool
	ChildCreated bool
}

// NewParentWorkflow creates a new instance of ParentWorkflow
func NewParentWorkflow(repo *TestRepository) *ParentWorkflow {
	return &ParentWorkflow{
		repo: repo,
	}
}

// NewChildWorkflow creates a new instance of ChildWorkflow
func NewChildWorkflow(repo *TestRepository) *ChildWorkflow {
	return &ChildWorkflow{
		repo: repo,
	}
}

// Setup initializes the workflow with domain data
func (w *ParentWorkflow) Setup(wf *domain.Workflow) {
	w.BaseWorkflow.Setup(wf)
}

func (w *ParentWorkflow) GetWorkflowData() *domain.Workflow {
	return w.WorkflowState
}

func (w *ParentWorkflow) GetStateVariables() map[string]string {
	return w.StateVariables
}

func (w *ParentWorkflow) InitialState() string {
	return ParentInit
}

func (w *ParentWorkflow) Description() string {
	return "Parent workflow that spawns and coordinates child workflows"
}

func (w *ParentWorkflow) StateTransitions() map[string][]string {
	return map[string][]string{
		ParentInit:                   {ParentSpawnChildren},
		ParentSpawnChildren:          {ParentWaitForChildren},
		ParentWaitForChildren:        {ParentProcessChildrenResults},
		ParentProcessChildrenResults: {ParentFinish},
	}
}

func (w *ParentWorkflow) GetAllStates() []models.WorkflowState {
	return []models.WorkflowState{
		{Name: ParentInit, StateType: models.StateStart},
		{Name: ParentSpawnChildren, StateType: models.StateNormal},
		{Name: ParentWaitForChildren, StateType: models.StateNormal},
		{Name: ParentProcessChildrenResults, StateType: models.StateNormal},
		{Name: ParentFinish, StateType: models.StateEnd},
	}
}

func (w *ParentWorkflow) GetRetryConfig() models.RetryConfig {
	return models.RetryConfig{
		MaxRetryCount:    3,
		RetryIntervalMin: time.Second * 1,
		RetryIntervalMax: time.Second * 5,
	}
}

// GetChildWorkflows implements the ParentChildCapable interface
func (w *ParentWorkflow) GetChildWorkflows(ctx context.Context) ([]domain.Workflow, error) {
	// In a real implementation, this would call the repository
	// For testing, we'll return a mock response
	slog.InfoContext(ctx, "Getting child workflows")
	
	if w.repo != nil {
		w.repo.ChildCreated = true
	}
	
	// Return mock child workflow for testing
	return []domain.Workflow{
		{
			ID:               2,
			Status:           "FINISHED",
			WorkflowType:     "ChildWorkflow",
			BusinessKey:      "child-1",
			ParentWorkflowID: sql.NullInt64{Int64: w.WorkflowState.ID, Valid: true},
			StateVars:        sql.NullString{String: `{"result":"success"}`, Valid: true},
		},
	}, nil
}

// ParentInit initializes the parent workflow
func (w *ParentWorkflow) ParentInit(ctx context.Context) (*models.NextState, error) {
	slog.InfoContext(ctx, "Initializing parent workflow")
	
	return &models.NextState{
		Name:      ParentSpawnChildren,
		ActionLog: "Parent workflow initialized",
	}, nil
}

// ParentSpawnChildren spawns child workflows
func (w *ParentWorkflow) ParentSpawnChildren(ctx context.Context) (*models.NextState, error) {
	slog.InfoContext(ctx, "Spawning child workflows")
	
	// Create child workflow requests
	childRequests := []models.ChildWorkflowRequest{
		gopherflow.CreateChildWorkflowRequest(
			"ChildWorkflow",
			"child-1",
			ChildInit,
			map[string]string{"input": "value1"},
		),
		gopherflow.CreateChildWorkflowRequest(
			"ChildWorkflow",
			"child-2",
			ChildInit,
			map[string]string{"input": "value2"},
		),
	}
	
	w.numCreated = len(childRequests)
	w.StateVariables["children_count"] = fmt.Sprintf("%d", w.numCreated)
	
	return &models.NextState{
		Name:           ParentWaitForChildren,
		ActionLog:      fmt.Sprintf("Spawned %d child workflows", w.numCreated),
		ChildWorkflows: childRequests,
	}, nil
}

// ParentWaitForChildren waits for child workflows to complete
func (w *ParentWorkflow) ParentWaitForChildren(ctx context.Context) (*models.NextState, error) {
	slog.InfoContext(ctx, "Waiting for child workflows to complete")
	
	// Check if all children are done
	// In a real implementation, this would check the status of children
	children, err := w.GetChildWorkflows(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get child workflows: %w", err)
	}
	
	completedCount := 0
	for _, child := range children {
		if child.Status == "FINISHED" {
			completedCount++
		}
	}
	
	// If not all children are complete, wait more
	if completedCount < w.numCreated {
		slog.InfoContext(ctx, "Not all children complete, waiting", 
			"completed", completedCount, 
			"total", w.numCreated)
		
		return &models.NextState{
			Name:                ParentWaitForChildren,
			ActionLog:           fmt.Sprintf("Waiting for children: %d/%d complete", completedCount, w.numCreated),
			NextExecutionOffset: "30 seconds",
		}, nil
	}
	
	return &models.NextState{
		Name:      ParentProcessChildrenResults,
		ActionLog: "All child workflows complete",
	}, nil
}

// ParentProcessChildrenResults processes results from child workflows
func (w *ParentWorkflow) ParentProcessChildrenResults(ctx context.Context) (*models.NextState, error) {
	slog.InfoContext(ctx, "Processing child workflow results")
	
	children, err := w.GetChildWorkflows(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get child workflows: %w", err)
	}
	
	// Process results from children
	results := make(map[string]string)
	for i, child := range children {
		if child.StateVars.Valid {
			childResults, err := gopherflow.ParseChildWorkflowResults(
				map[string]string{"child_result": child.StateVars.String},
				"child_result",
			)
			if err != nil {
				slog.WarnContext(ctx, "Failed to parse child results", "error", err)
				continue
			}
			
			if childResults != nil {
				results[fmt.Sprintf("child_%d_result", i+1)] = childResults["result"]
			}
		}
	}
	
	// Store aggregated results
	for k, v := range results {
		w.StateVariables[k] = v
	}
	
	return &models.NextState{
		Name:      ParentFinish,
		ActionLog: "Processed all child workflow results",
	}, nil
}

// Setup initializes the workflow with domain data
func (w *ChildWorkflow) Setup(wf *domain.Workflow) {
	w.BaseWorkflow.Setup(wf)
}

func (w *ChildWorkflow) GetWorkflowData() *domain.Workflow {
	return w.WorkflowState
}

func (w *ChildWorkflow) GetStateVariables() map[string]string {
	return w.StateVariables
}

func (w *ChildWorkflow) InitialState() string {
	return ChildInit
}

func (w *ChildWorkflow) Description() string {
	return "Child workflow that performs a task and wakes its parent"
}

func (w *ChildWorkflow) StateTransitions() map[string][]string {
	return map[string][]string{
		ChildInit:        {ChildProcessing},
		ChildProcessing:  {ChildWakeParent},
		ChildWakeParent:  {ChildFinish},
	}
}

func (w *ChildWorkflow) GetAllStates() []models.WorkflowState {
	return []models.WorkflowState{
		{Name: ChildInit, StateType: models.StateStart},
		{Name: ChildProcessing, StateType: models.StateNormal},
		{Name: ChildWakeParent, StateType: models.StateNormal},
		{Name: ChildFinish, StateType: models.StateEnd},
	}
}

func (w *ChildWorkflow) GetRetryConfig() models.RetryConfig {
	return models.RetryConfig{
		MaxRetryCount:    3,
		RetryIntervalMin: time.Second * 1,
		RetryIntervalMax: time.Second * 5,
	}
}

// WakeParent implements the ParentChildCapable interface
func (w *ChildWorkflow) WakeParent(ctx context.Context) error {
	slog.InfoContext(ctx, "Waking parent workflow")
	
	if w.repo != nil {
		w.repo.ParentWoken = true
	}
	
	// In a real implementation, this would call the repository to wake the parent
	return nil
}

// ChildInit initializes the child workflow
func (w *ChildWorkflow) ChildInit(ctx context.Context) (*models.NextState, error) {
	slog.InfoContext(ctx, "Initializing child workflow")
	
	input := w.StateVariables["input"]
	slog.InfoContext(ctx, "Child workflow received input", "input", input)
	
	return &models.NextState{
		Name:      ChildProcessing,
		ActionLog: "Child workflow initialized",
	}, nil
}

// ChildProcessing performs the child workflow's task
func (w *ChildWorkflow) ChildProcessing(ctx context.Context) (*models.NextState, error) {
	slog.InfoContext(ctx, "Child workflow processing")
	
	// Store the result
	w.StateVariables["result"] = "success"
	
	return &models.NextState{
		Name:      ChildWakeParent,
		ActionLog: "Child workflow processed data",
	}, nil
}

// ChildWakeParent wakes the parent workflow
func (w *ChildWorkflow) ChildWakeParent(ctx context.Context) (*models.NextState, error) {
	slog.InfoContext(ctx, "Child workflow waking parent")
	
	// Wake the parent
	err := w.WakeParent(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to wake parent: %w", err)
	}
	
	return &models.NextState{
		Name:      ChildFinish,
		ActionLog: "Parent workflow awakened",
	}, nil
}
