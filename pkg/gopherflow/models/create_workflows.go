package models

import (
	"time"
)

// CreateWorkflowRequest is the payload for creating a workflow.
type CreateWorkflowRequest struct {
	ExternalID    string            `json:"externalId"`
	ExecutorGroup string            `json:"executorGroup"`
	WorkflowType  string            `json:"workflowType"`
	BusinessKey   string            `json:"businessKey"`
	StateVars     map[string]string `json:"stateVars"`
	// Optional scheduling inputs
	NextActivation       *time.Time `json:"nextActivation,omitempty"`
	NextActivationOffset string     `json:"nextActivationOffset,omitempty"`
}

// createWorkflowResponse is returned on successful creation.
type CreateWorkflowResponse struct {
	ID int64 `json:"id"`
}

// CreateAndWaitRequest is the payload for creating a workflow then waiting the number of seconds for the workflow to reach the given states. otherwise it times out
type CreateAndWaitRequest struct {
	CreateWorkflowRequest CreateWorkflowRequest `json:"createWorkflowRequest"`
	WaitSeconds           int                   `json:"waitSeconds"`
	CheckSeconds          int                   `json:"checkSeconds"`
	WaitForStates         []string              `json:"waitForStates"`
}

// WorkflowApiResponse represents the API response for a workflow.
type WorkflowApiResponse struct {
	ID             int64             `json:"id"`
	Status         string            `json:"status"`
	ExecutionCount int               `json:"executionCount"`
	RetryCount     int               `json:"retryCount"`
	Created        time.Time         `json:"created"`
	Modified       time.Time         `json:"modified"`
	NextActivation time.Time         `json:"nextActivation,omitempty"`
	Started        time.Time         `json:"started,omitempty"`
	ExecutorID     string            `json:"executorId,omitempty"`
	ExecutorGroup  string            `json:"executorGroup"`
	WorkflowType   string            `json:"workflowType"`
	ExternalID     string            `json:"externalId"`
	BusinessKey    string            `json:"businessKey"`
	State          string            `json:"state"`
	StateVars      map[string]string `json:"stateVars,omitempty"`
}
