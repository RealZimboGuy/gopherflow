package web

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// newBrokenWeb returns a controller whose database is closed, so every
// repository call fails. That exercises the error branch of each handler.
func newBrokenWeb(t *testing.T) *testWeb {
	t.Helper()
	tw := newTestWeb(t)
	if err := tw.DB.Close(); err != nil {
		t.Fatalf("close db: %v", err)
	}
	return tw
}

func TestHandlersReportAnErrorWhenTheDatabaseIsUnavailable(t *testing.T) {
	handlers := map[string]func(*WebController, http.ResponseWriter, *http.Request){
		"overview": func(c *WebController, w http.ResponseWriter, r *http.Request) {
			c.overviewHandler(w, r)
		},
		"inProgressTop": func(c *WebController, w http.ResponseWriter, r *http.Request) {
			c.inProgressTopHandler(w, r)
		},
		"nextExecutionTop": func(c *WebController, w http.ResponseWriter, r *http.Request) {
			c.nextExecutionTopHandler(w, r)
		},
		"definitions": func(c *WebController, w http.ResponseWriter, r *http.Request) {
			c.definitionsHandler(w, r)
		},
		"executors": func(c *WebController, w http.ResponseWriter, r *http.Request) {
			c.executorsHandler(w, r)
		},
		"users": func(c *WebController, w http.ResponseWriter, r *http.Request) {
			c.usersHandler(w, r)
		},
		"searchResults": func(c *WebController, w http.ResponseWriter, r *http.Request) {
			c.searchResultsHandler(w, r)
		},
	}

	for name, call := range handlers {
		t.Run(name, func(t *testing.T) {
			tw := newBrokenWeb(t)

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			w := httptest.NewRecorder()

			call(tw.Ctrl, w, req)

			if w.Code == http.StatusOK {
				t.Errorf("status = 200 with an unusable database; want an error response")
			}
		})
	}
}

func TestWorkflowDetailsReportsAnErrorWhenTheDatabaseIsUnavailable(t *testing.T) {
	tw := newBrokenWeb(t)

	req := httptest.NewRequest(http.MethodGet, "/details/1", nil)
	req.SetPathValue("id", "1")
	w := httptest.NewRecorder()

	tw.Ctrl.workflowDetailsHandler(w, req)

	if w.Code == http.StatusOK {
		t.Error("status = 200 with an unusable database; want an error response")
	}
}

func TestLoginSubmitReportsAnErrorWhenTheDatabaseIsUnavailable(t *testing.T) {
	tw := newBrokenWeb(t)

	req := postForm(t, "/login", map[string][]string{
		"username": {"admin"},
		"password": {"admin"},
	})
	w := httptest.NewRecorder()

	tw.Ctrl.loginSubmitHandler(w, req)

	if w.Code == http.StatusOK || w.Code == http.StatusSeeOther {
		t.Errorf("status = %d with an unusable database; want an error response", w.Code)
	}
}

func TestEditUserReportsAnErrorWhenTheDatabaseIsUnavailable(t *testing.T) {
	tw := newBrokenWeb(t)

	req := httptest.NewRequest(http.MethodGet, "/users/1/edit", nil)
	req.SetPathValue("id", "1")
	w := httptest.NewRecorder()

	tw.Ctrl.editUserHandler(w, req)

	if w.Code == http.StatusOK {
		t.Error("status = 200 with an unusable database; want an error response")
	}
}

func TestCreateUserSubmitReportsAnErrorWhenTheDatabaseIsUnavailable(t *testing.T) {
	tw := newBrokenWeb(t)

	req := postForm(t, "/users/create", map[string][]string{
		"username": {"someone"},
		"password": {"pw"},
	})
	w := httptest.NewRecorder()

	tw.Ctrl.createUserSubmitHandler(w, req)

	if w.Code == http.StatusOK || w.Code == http.StatusSeeOther {
		t.Errorf("status = %d with an unusable database; want an error response", w.Code)
	}
}
