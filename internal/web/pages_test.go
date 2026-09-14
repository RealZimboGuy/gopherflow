package web

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/RealZimboGuy/gopherflow/pkg/gopherflow/domain"
)

// assertRendered fails when the handler did not return a successful HTML body.
func assertRendered(t *testing.T, w *httptest.ResponseRecorder, mustContain ...string) {
	t.Helper()
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", w.Code, w.Body.String())
	}
	if w.Body.Len() == 0 {
		t.Fatal("handler rendered an empty body")
	}
	body := w.Body.String()
	for _, want := range mustContain {
		if !strings.Contains(body, want) {
			t.Errorf("rendered page does not contain %q", want)
		}
	}
}

func TestDashboardHandler(t *testing.T) {
	tw := newTestWeb(t)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	tw.Ctrl.handler(w, req)

	assertRendered(t, w, "GopherFlow")
}

func TestOverviewHandler(t *testing.T) {
	tw := newTestWeb(t)
	tw.seedWorkflow(t, "ext-overview-1")
	tw.seedWorkflow(t, "ext-overview-2")

	req := httptest.NewRequest(http.MethodGet, "/home/overview", nil)
	w := httptest.NewRecorder()

	tw.Ctrl.overviewHandler(w, req)

	assertRendered(t, w, "DemoWorkflow")
}

func TestOverviewHandlerWithNoWorkflows(t *testing.T) {
	tw := newTestWeb(t)

	req := httptest.NewRequest(http.MethodGet, "/home/overview", nil)
	w := httptest.NewRecorder()

	tw.Ctrl.overviewHandler(w, req)

	assertRendered(t, w)
}

func TestInProgressTopHandler(t *testing.T) {
	tw := newTestWeb(t)
	tw.seedWorkflow(t, "ext-inprogress")

	req := httptest.NewRequest(http.MethodGet, "/home/inprogresscount", nil)
	w := httptest.NewRecorder()

	tw.Ctrl.inProgressTopHandler(w, req)

	assertRendered(t, w)
}

func TestNextExecutionTopHandler(t *testing.T) {
	tw := newTestWeb(t)
	tw.seedWorkflow(t, "ext-next")

	req := httptest.NewRequest(http.MethodGet, "/home/nextexecutioncount", nil)
	w := httptest.NewRecorder()

	tw.Ctrl.nextExecutionTopHandler(w, req)

	assertRendered(t, w)
}

func TestSettingsHandler(t *testing.T) {
	tw := newTestWeb(t)

	req := httptest.NewRequest(http.MethodGet, "/settings", nil)
	w := httptest.NewRecorder()

	tw.Ctrl.settingsHandler(w, req)

	assertRendered(t, w)
}

func TestExecutorsHandler(t *testing.T) {
	tw := newTestWeb(t)

	if _, err := tw.Execs.Save(&domain.Executor{Name: "worker-web"}); err != nil {
		t.Fatalf("seed executor: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/executors", nil)
	w := httptest.NewRecorder()

	tw.Ctrl.executorsHandler(w, req)

	assertRendered(t, w, "worker-web")
}

func TestDefinitionsHandler(t *testing.T) {
	tw := newTestWeb(t)
	tw.seedDefinition(t, "DemoWorkflow")

	req := httptest.NewRequest(http.MethodGet, "/definitions", nil)
	w := httptest.NewRecorder()

	tw.Ctrl.definitionsHandler(w, req)

	assertRendered(t, w, "DemoWorkflow")
}

func TestDefinitionByNameHandler(t *testing.T) {
	tw := newTestWeb(t)
	tw.seedDefinition(t, "DemoWorkflow")
	tw.seedWorkflow(t, "ext-def")

	t.Run("known definition", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/definitions/DemoWorkflow", nil)
		req.SetPathValue("name", "DemoWorkflow")
		w := httptest.NewRecorder()

		tw.Ctrl.definitionByNameHandler(w, req)

		assertRendered(t, w, "DemoWorkflow")
	})

	t.Run("unknown definition", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/definitions/Nope", nil)
		req.SetPathValue("name", "Nope")
		w := httptest.NewRecorder()

		tw.Ctrl.definitionByNameHandler(w, req)

		if w.Code == http.StatusOK && w.Body.Len() == 0 {
			t.Error("handler returned 200 with an empty body for an unknown definition")
		}
	})
}

func TestCreateWorkflowPageHandler(t *testing.T) {
	tw := newTestWeb(t)
	tw.seedDefinition(t, "DemoWorkflow")

	req := httptest.NewRequest(http.MethodGet, "/definitions/DemoWorkflow/create", nil)
	req.SetPathValue("name", "DemoWorkflow")
	w := httptest.NewRecorder()

	tw.Ctrl.createWorkflowPageHandler(w, req)

	assertRendered(t, w, "DemoWorkflow")
}

func TestWorkflowDetailsHandler(t *testing.T) {
	tw := newTestWeb(t)
	tw.seedDefinition(t, "DemoWorkflow")
	id := tw.seedWorkflow(t, "ext-details")

	t.Run("known workflow", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/details/1", nil)
		req.SetPathValue("id", strconv.FormatInt(id, 10))
		w := httptest.NewRecorder()

		tw.Ctrl.workflowDetailsHandler(w, req)

		assertRendered(t, w, "ext-details")
	})

	t.Run("unknown workflow", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/details/9999", nil)
		req.SetPathValue("id", "9999")
		w := httptest.NewRecorder()

		tw.Ctrl.workflowDetailsHandler(w, req)

		if w.Code == http.StatusOK && w.Body.Len() == 0 {
			t.Error("handler returned 200 with an empty body for an unknown workflow")
		}
	})

	t.Run("non-numeric id", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/details/abc", nil)
		req.SetPathValue("id", "abc")
		w := httptest.NewRecorder()

		tw.Ctrl.workflowDetailsHandler(w, req)

		if w.Code == http.StatusOK && w.Body.Len() == 0 {
			t.Error("handler returned 200 with an empty body for a non-numeric id")
		}
	})
}

func TestSearchPageHandler(t *testing.T) {
	tw := newTestWeb(t)
	tw.seedWorkflow(t, "ext-searchpage")

	req := httptest.NewRequest(http.MethodGet, "/search", nil)
	w := httptest.NewRecorder()

	tw.Ctrl.searchPageHandler(w, req)

	assertRendered(t, w)
}

func TestSearchResultsHandler(t *testing.T) {
	tw := newTestWeb(t)
	tw.seedWorkflow(t, "ext-res-1")
	tw.seedWorkflow(t, "ext-res-2")

	tests := []struct {
		name  string
		query string
	}{
		{"no filters", ""},
		{"by external id", "?q=ext-res-1"},
		{"by status", "?status=NEW"},
		{"by state", "?state=Init"},
		{"by workflow type", "?workflowType=DemoWorkflow"},
		{"by executor group", "?executorGroup=default"},
		{"with paging", "?limit=1&offset=0"},
		{"second page", "?limit=1&offset=1"},
		{"invalid paging falls back", "?limit=abc&offset=xyz"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/search/results"+tt.query, nil)
			w := httptest.NewRecorder()

			tw.Ctrl.searchResultsHandler(w, req)

			assertRendered(t, w)
		})
	}
}

func TestWorkflowDetailsRendersActionsAndStateVars(t *testing.T) {
	tw := newTestWeb(t)
	tw.seedDefinition(t, "DemoWorkflow")
	id := tw.seedWorkflow(t, "ext-details-rich")

	if err := tw.Workflow.SaveWorkflowVariables(id, `{"colour":"blue","size":"large"}`); err != nil {
		t.Fatalf("SaveWorkflowVariables: %v", err)
	}
	for _, name := range []string{"Init", "Review"} {
		if _, err := tw.Actions.Save(&domain.WorkflowAction{
			WorkflowID: id, ExecutorID: 1, Type: "STATE_CHANGE",
			Name: name, Text: "entered " + name,
		}); err != nil {
			t.Fatalf("seed action %s: %v", name, err)
		}
	}

	req := httptest.NewRequest(http.MethodGet, "/details/1", nil)
	req.SetPathValue("id", strconv.FormatInt(id, 10))
	w := httptest.NewRecorder()

	tw.Ctrl.workflowDetailsHandler(w, req)

	assertRendered(t, w, "ext-details-rich", "colour", "blue")
}

func TestDefinitionByNameWithGroupRoute(t *testing.T) {
	tw := newTestWeb(t)
	tw.seedDefinition(t, "DemoWorkflow")

	req := httptest.NewRequest(http.MethodGet, "/definitions/default/DemoWorkflow", nil)
	req.SetPathValue("group", "default")
	req.SetPathValue("name", "DemoWorkflow")
	w := httptest.NewRecorder()

	tw.Ctrl.definitionByNameHandler(w, req)

	assertRendered(t, w, "DemoWorkflow")
}

func TestDefinitionsHandlerWithExecutingWorkflows(t *testing.T) {
	tw := newTestWeb(t)
	tw.seedDefinition(t, "DemoWorkflow")

	id := tw.seedWorkflow(t, "ext-defs-exec")
	if err := tw.Workflow.UpdateWorkflowStatus(id, "EXECUTING"); err != nil {
		t.Fatalf("UpdateWorkflowStatus: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/definitions", nil)
	w := httptest.NewRecorder()

	tw.Ctrl.definitionsHandler(w, req)

	assertRendered(t, w, "DemoWorkflow")
}

func TestSearchResultsWithFinishedWorkflow(t *testing.T) {
	tw := newTestWeb(t)
	id := tw.seedWorkflow(t, "ext-finished")
	if err := tw.Workflow.UpdateWorkflowStatus(id, "FINISHED"); err != nil {
		t.Fatalf("UpdateWorkflowStatus: %v", err)
	}

	// A finished workflow renders "-" for next activation rather than a duration.
	req := httptest.NewRequest(http.MethodGet, "/search/results?status=FINISHED", nil)
	w := httptest.NewRecorder()

	tw.Ctrl.searchResultsHandler(w, req)

	assertRendered(t, w, "ext-finished")
}
