package controllers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/RealZimboGuy/gopherflow/pkg/gopherflow/domain"
	"github.com/RealZimboGuy/gopherflow/pkg/gopherflow/models"
)

func postJSON(t *testing.T, target string, body any) *http.Request {
	t.Helper()
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal body: %v", err)
	}
	return httptest.NewRequest(http.MethodPost, target, bytes.NewReader(raw))
}

func TestHandleGetWorkflowByExternalId(t *testing.T) {
	rc := newRealController(t)
	rc.seedWorkflow(t, "ext-find")

	t.Run("found", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/workflowByExternalId/ext-find", nil)
		req.SetPathValue("externalId", "ext-find")
		w := httptest.NewRecorder()

		rc.Ctrl.handleGetWorkflowByExternalId(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200; body %s", w.Code, w.Body.String())
		}
		var got models.WorkflowApiResponse
		if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if got.ExternalID != "ext-find" {
			t.Errorf("externalId = %q, want ext-find", got.ExternalID)
		}
	})

	t.Run("not found", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/workflowByExternalId/nope", nil)
		req.SetPathValue("externalId", "nope")
		w := httptest.NewRecorder()

		rc.Ctrl.handleGetWorkflowByExternalId(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("status = %d, want 404", w.Code)
		}
	})

	t.Run("missing path value", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/workflowByExternalId/", nil)
		w := httptest.NewRecorder()

		rc.Ctrl.handleGetWorkflowByExternalId(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want 400", w.Code)
		}
	})

	t.Run("wrong method", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/workflowByExternalId/ext-find", nil)
		w := httptest.NewRecorder()

		rc.Ctrl.handleGetWorkflowByExternalId(w, req)

		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("status = %d, want 405", w.Code)
		}
	})
}

func TestHandleSearchWorkflows(t *testing.T) {
	rc := newRealController(t)
	rc.seedWorkflow(t, "ext-s1")
	rc.seedWorkflow(t, "ext-s2")

	t.Run("by external id", func(t *testing.T) {
		req := postJSON(t, "/api/workflows/search", models.SearchWorkflowRequest{
			ExternalID: "ext-s1", Limit: 10,
		})
		w := httptest.NewRecorder()

		rc.Ctrl.handleSearchWorkflows(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200; body %s", w.Code, w.Body.String())
		}
		var got models.SearchWorkflowResponse
		if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
			t.Fatalf("decode response: %v", err)
		}
	})

	t.Run("empty filter returns results", func(t *testing.T) {
		req := postJSON(t, "/api/workflows/search", models.SearchWorkflowRequest{Limit: 10})
		w := httptest.NewRecorder()

		rc.Ctrl.handleSearchWorkflows(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("status = %d, want 200; body %s", w.Code, w.Body.String())
		}
	})

	t.Run("invalid json", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/workflows/search", bytes.NewReader([]byte("{not json")))
		w := httptest.NewRecorder()

		rc.Ctrl.handleSearchWorkflows(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want 400", w.Code)
		}
	})

	t.Run("wrong method", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/workflows/search", nil)
		w := httptest.NewRecorder()

		rc.Ctrl.handleSearchWorkflows(w, req)

		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("status = %d, want 405", w.Code)
		}
	})
}

func TestHandleUpdateWorkflowState(t *testing.T) {
	rc := newRealController(t)
	id := rc.seedWorkflow(t, "ext-update")

	t.Run("by numeric id", func(t *testing.T) {
		req := postJSON(t, "/api/workflows/1/state", models.UpdateWorkflowStateRequest{State: "Review"})
		req.SetPathValue("id", itoa(id))
		w := httptest.NewRecorder()

		rc.Ctrl.handleUpdateWorkflowState(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200; body %s", w.Code, w.Body.String())
		}
		got, err := rc.Workflow.FindByID(id)
		if err != nil {
			t.Fatalf("FindByID: %v", err)
		}
		if got.State != "Review" {
			t.Errorf("state = %q, want Review", got.State)
		}
	})

	t.Run("by external id", func(t *testing.T) {
		req := postJSON(t, "/api/workflows/ext-update/state", models.UpdateWorkflowStateRequest{State: "Approve"})
		req.SetPathValue("id", "ext-update")
		w := httptest.NewRecorder()

		rc.Ctrl.handleUpdateWorkflowState(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200; body %s", w.Code, w.Body.String())
		}
	})

	t.Run("unknown workflow", func(t *testing.T) {
		req := postJSON(t, "/api/workflows/nope/state", models.UpdateWorkflowStateRequest{State: "Review"})
		req.SetPathValue("id", "nope")
		w := httptest.NewRecorder()

		rc.Ctrl.handleUpdateWorkflowState(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("status = %d, want 404", w.Code)
		}
	})

	t.Run("missing id", func(t *testing.T) {
		req := postJSON(t, "/api/workflows//state", models.UpdateWorkflowStateRequest{State: "Review"})
		w := httptest.NewRecorder()

		rc.Ctrl.handleUpdateWorkflowState(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want 400", w.Code)
		}
	})

	t.Run("wrong method", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/workflows/1/state", nil)
		w := httptest.NewRecorder()

		rc.Ctrl.handleUpdateWorkflowState(w, req)

		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("status = %d, want 405", w.Code)
		}
	})
}

func TestHandleUpdateStateVar(t *testing.T) {
	rc := newRealController(t)
	id := rc.seedWorkflow(t, "ext-statevar")

	t.Run("sets a variable", func(t *testing.T) {
		req := postJSON(t, "/api/workflows/1/stateVar", models.UpdateStateVarRequest{
			Key: "colour", Value: "blue",
		})
		req.SetPathValue("id", itoa(id))
		w := httptest.NewRecorder()

		rc.Ctrl.handleUpdateStateVar(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200; body %s", w.Code, w.Body.String())
		}
		got, err := rc.Workflow.FindByID(id)
		if err != nil {
			t.Fatalf("FindByID: %v", err)
		}
		if got.StateVars.String == "" {
			t.Error("state vars were not written")
		}
	})

	t.Run("unknown workflow", func(t *testing.T) {
		req := postJSON(t, "/api/workflows/nope/stateVar", models.UpdateStateVarRequest{Key: "k", Value: "v"})
		req.SetPathValue("id", "nope")
		w := httptest.NewRecorder()

		rc.Ctrl.handleUpdateStateVar(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("status = %d, want 404", w.Code)
		}
	})

	t.Run("invalid json", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/workflows/1/stateVar", bytes.NewReader([]byte("{bad")))
		req.SetPathValue("id", itoa(id))
		w := httptest.NewRecorder()

		rc.Ctrl.handleUpdateStateVar(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want 400", w.Code)
		}
	})

	t.Run("wrong method", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/workflows/1/stateVar", nil)
		w := httptest.NewRecorder()

		rc.Ctrl.handleUpdateStateVar(w, req)

		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("status = %d, want 405", w.Code)
		}
	})
}

func TestHandleGetWorkflowDefinitionByName(t *testing.T) {
	rc := newRealController(t)

	if err := rc.Defs.Save(&domain.WorkflowDefinition{
		Name: "DemoWorkflow", Description: "demo",
		Created: time.Now(), Updated: time.Now(), FlowChart: "flowchart TD",
	}); err != nil {
		t.Fatalf("seed definition: %v", err)
	}

	t.Run("found", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/definitions/DemoWorkflow", nil)
		req.SetPathValue("name", "DemoWorkflow")
		w := httptest.NewRecorder()

		rc.Ctrl.handleGetWorkflowDefinitionByName(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200; body %s", w.Code, w.Body.String())
		}
	})

	t.Run("missing name", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/definitions/", nil)
		w := httptest.NewRecorder()

		rc.Ctrl.handleGetWorkflowDefinitionByName(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want 400", w.Code)
		}
	})

	t.Run("wrong method", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/definitions/DemoWorkflow", nil)
		w := httptest.NewRecorder()

		rc.Ctrl.handleGetWorkflowDefinitionByName(w, req)

		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("status = %d, want 405", w.Code)
		}
	})
}

func itoa(i int64) string {
	return strconv.FormatInt(i, 10)
}
