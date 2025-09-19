package controllers

import "net/http"

// RegisterRoutes wires the HTTP routes for this controller.
func (c *WorkflowsController) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/workflows", c.RequireAuth(c.handleCreateWorkflow))
	mux.HandleFunc("/api/workflows/search", c.RequireAuth(c.handleSearchWorkflows))
	mux.HandleFunc("POST /api/workflows/{id}/state", c.RequireAuth(c.handleUpdateWorkflowState))
	mux.HandleFunc("POST /api/workflows/{id}/statevars", c.RequireAuth(c.handleUpdateStateVar))
}
func (c *ActionsController) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/actions/byWorkflowId/{id}", c.RequireAuth(c.handleGetActionsForWorkflow))
}
func (c *ExecutorsController) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/executors", c.RequireAuth(c.handleGetExecutors))
}
