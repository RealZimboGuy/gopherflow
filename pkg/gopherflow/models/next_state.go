package models

import "time"

// ChildWorkflowRequest represents a request to spawn a child workflow
type ChildWorkflowRequest struct {
	WorkflowType  string            // Type of child workflow to spawn
	BusinessKey   string            // Business key for the child workflow
	InitialState  string            // Initial state for the child workflow
	StateVariables map[string]string // Initial state variables for the child workflow
}

type NextState struct {
	Name                string                // Name of the state
	ActionLog           string                // Additional information about the state
	NextExecution       time.Time             // specific time set by the code
	NextExecutionOffset string                // a human friendly time string sent to the database ie 10 minutes
	ChildWorkflows      []ChildWorkflowRequest // Child workflows to spawn
}
