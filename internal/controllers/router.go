package controllers

import "net/http"

// RegisterRoutes wires the HTTP routes for this controller.
func (c *WorkflowsController) RegisterRoutes() {
	http.HandleFunc("/api/workflows", c.RequireAuth(c.handleCreateWorkflow))
	http.HandleFunc("/api/workflows/{id}", c.RequireAuth(c.handleGetWorkflowById))
	http.HandleFunc("/api/workflowByExternalId/{externalId}", c.RequireAuth(c.handleGetWorkflowByExternalId))
	http.HandleFunc("/api/createAndWait", c.RequireAuth(c.handleCreateAndWaitWorkflow))
	http.HandleFunc("/api/workflows/search", c.RequireAuth(c.handleSearchWorkflows))
	http.HandleFunc("/api/definitions", c.RequireAuth(c.handleListWorkflowDefinitions))
	http.HandleFunc("/api/definitions/{name}", c.RequireAuth(c.handleGetWorkflowDefinitionByName))
	http.HandleFunc("POST /api/workflows/{id}/state", c.RequireAuth(c.handleUpdateWorkflowState))
	http.HandleFunc("POST /api/workflows/{id}/stateAndWait", c.RequireAuth(c.handleUpdateWorkflowStateAndWait))
	http.HandleFunc("POST /api/workflows/{id}/statevars", c.RequireAuth(c.handleUpdateStateVar))
}
func (c *ActionsController) RegisterRoutes() {
	http.HandleFunc("/api/actions/byWorkflowId/{id}", c.RequireAuth(c.handleGetActionsForWorkflow))
}
func (c *ExecutorsController) RegisterRoutes() {
	http.HandleFunc("/api/executors", c.RequireAuth(c.handleGetExecutors))
}
