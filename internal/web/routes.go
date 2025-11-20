package web

import (
	"io/fs"
	"net/http"
)

func (c *WebController) RegisterRoutes() {

	// Static files (images)
	imagesSub, err := fs.Sub(templatesFS, "images")
	if err != nil {
		panic(err)
	}
	http.Handle("/images/", http.StripPrefix("/images/", http.FileServer(http.FS(imagesSub))))

	// Public routes
	http.HandleFunc("GET /login", c.loginPageHandler)
	http.HandleFunc("POST /login", c.loginSubmitHandler)

	// Protected routes
	http.HandleFunc("/", c.RequireAuth(c.handler))
	http.HandleFunc("POST /logout", c.RequireAuth(c.logoutHandler))
	// Home dashboard fragments
	http.HandleFunc("GET /home/overview", c.RequireAuth(c.overviewHandler))
	http.HandleFunc("GET /home/inprogresscount", c.RequireAuth(c.inProgressTopHandler))
	http.HandleFunc("GET /home/nextexecutioncount", c.RequireAuth(c.nextExecutionTopHandler))
	// Settings page
	http.HandleFunc("GET /settings", c.RequireAuth(c.settingsHandler))
	// Search page and results
	http.HandleFunc("GET /search", c.RequireAuth(c.searchPageHandler))
	http.HandleFunc("GET /search/results", c.RequireAuth(c.searchResultsHandler))
	http.HandleFunc("GET /details/{id}", c.RequireAuth(c.workflowDetailsHandler))
	// Executors page
	http.HandleFunc("GET /executors", c.RequireAuth(c.executorsHandler))
	// Full page list of definitions
	http.HandleFunc("GET /definitions", c.RequireAuth(c.definitionsHandler))
	// Detail fragment; support both /definitions/{name} and /definitions/{group}/{name}
	http.HandleFunc("GET /definitions/{name}", c.RequireAuth(c.definitionByNameHandler))
	http.HandleFunc("GET /definitions/{group}/{name}", c.RequireAuth(c.definitionByNameHandler))
	// Create workflow page for a given definition
	http.HandleFunc("GET /definitions/{name}/create", c.RequireAuth(c.createWorkflowPageHandler))
	// User management pages
	http.HandleFunc("GET /users", c.RequireAuth(c.usersHandler))
	http.HandleFunc("GET /users/create", c.RequireAuth(c.createUserHandler))
	http.HandleFunc("POST /users/create", c.RequireAuth(c.createUserSubmitHandler))
	http.HandleFunc("POST /users/{id}/delete", c.RequireAuth(c.deleteUserHandler))
}
