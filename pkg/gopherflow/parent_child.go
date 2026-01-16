package gopherflow

import (
	"encoding/json"

	"github.com/RealZimboGuy/gopherflow/pkg/gopherflow/models"
)

// CreateChildWorkflowRequest creates a new child workflow request that can be added to
// a NextState's ChildWorkflows list
func CreateChildWorkflowRequest(
	workflowType string,
	businessKey string,
	initialState string,
	stateVars map[string]string,
) models.ChildWorkflowRequest {
	return models.ChildWorkflowRequest{
		WorkflowType:   workflowType,
		BusinessKey:    businessKey,
		InitialState:   initialState,
		StateVariables: stateVars,
	}
}

// ParseChildWorkflowResults parses the child workflow results from a parent workflow's state variables
// This can be used to extract state variables from completed child workflows
func ParseChildWorkflowResults(stateVars map[string]string, key string) (map[string]string, error) {
	data, ok := stateVars[key]
	if !ok {
		return nil, nil // No results found, return empty map
	}
	
	var results map[string]string
	if err := json.Unmarshal([]byte(data), &results); err != nil {
		return nil, err
	}
	
	return results, nil
}
