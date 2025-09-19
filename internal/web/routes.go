package web

import (
	"io/fs"
	"net/http"
)

func (c *WebController) RegisterRoutes(mux *http.ServeMux) {
	
	// Static files (images)
	imagesSub, err := fs.Sub(templatesFS, "internal/web/images")
	if err != nil {
		panic(err)
	}
	mux.Handle("/images/", http.StripPrefix("/images/", http.FileServer(http.FS(imagesSub))))

	// Public routes
	mux.HandleFunc("GET /login", c.loginPageHandler)
	mux.HandleFunc("POST /login", c.loginSubmitHandler)

	// Protected routes
	mux.HandleFunc("/", c.RequireAuth(c.handler))
	mux.HandleFunc("POST /logout", c.RequireAuth(c.logoutHandler))
	// Home dashboard fragments
	mux.HandleFunc("GET /home/overview", c.RequireAuth(c.overviewHandler))
	mux.HandleFunc("GET /home/inprogresscount", c.RequireAuth(c.inProgressTopHandler))
	mux.HandleFunc("GET /home/nextexecutioncount", c.RequireAuth(c.nextExecutionTopHandler))
	// Settings page
	mux.HandleFunc("GET /settings", c.RequireAuth(c.settingsHandler))
	// Search page and results
	mux.HandleFunc("GET /search", c.RequireAuth(c.searchPageHandler))
	mux.HandleFunc("GET /search/results", c.RequireAuth(c.searchResultsHandler))
	mux.HandleFunc("GET /details/{id}", c.RequireAuth(c.workflowDetailsHandler))
	// Executors page
	mux.HandleFunc("GET /executors", c.RequireAuth(c.executorsHandler))
	// Full page list of definitions
	mux.HandleFunc("GET /definitions", c.RequireAuth(c.definitionsHandler))
	// Detail fragment; support both /definitions/{name} and /definitions/{group}/{name}
	mux.HandleFunc("GET /definitions/{name}", c.RequireAuth(c.definitionByNameHandler))
	mux.HandleFunc("GET /definitions/{group}/{name}", c.RequireAuth(c.definitionByNameHandler))
	// Create workflow page for a given definition
	mux.HandleFunc("GET /definitions/{name}/create", c.RequireAuth(c.createWorkflowPageHandler))
}
