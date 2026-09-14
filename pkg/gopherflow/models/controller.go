package models

import domain "github.com/RealZimboGuy/gopherflow/pkg/gopherflow/domain"

// SearchWorkflowRequest filters a workflow search. The identity fields (ID,
// ExternalID and BusinessKey) are combined with OR; the remaining fields are
// combined with AND. A zero Limit means no limit.
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

// SearchWorkflowResponse is the result of a workflow search, carrying the
// matching workflows along with the offset used so callers can page.
type SearchWorkflowResponse struct {
	Results   int               `json:"results"`
	Workflows []domain.Workflow `json:"workflows"`
	Offset    int64             `json:"offset"`
}
