package models

import "github.com/RealZimboGuy/gopherflow/internal/domain"

type SearchWorkflowRequest struct {
	ID            int64  `json:"id"`
	ExternalID    string `json:"externalId"`
	ExecutorGroup string `json:"executorGroup"`
	WorkflowType  string `json:"workflowType"`
	BusinessKey   string `json:"businessKey"`
	State         string `json:"state"`
	Status        string `json:"status"`
	Limit         int64  `json:"limit"`
	Offset        int64  `json:"offset"`
}
type SearchWorkflowResponse struct {
	Results   int               `json:"results"`
	Workflows []domain.Workflow `json:"workflows"`
	Offset    int64             `json:"offset"`
}
