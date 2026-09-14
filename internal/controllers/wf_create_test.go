package controllers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/RealZimboGuy/gopherflow/pkg/gopherflow/models"
)

func validCreateRequest(externalID string) models.CreateWorkflowRequest {
	return models.CreateWorkflowRequest{
		ExternalID:    externalID,
		ExecutorGroup: "default",
		WorkflowType:  "DemoWorkflow",
		BusinessKey:   "bk-" + externalID,
		StateVars:     map[string]string{"a": "1"},
	}
}

func TestValidateCreateWorkflow(t *testing.T) {
	t.Run("accepts a complete request", func(t *testing.T) {
		if err := validateCreateWorkflow(t.Context(), validCreateRequest("ext-ok")); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	fields := []struct {
		name   string
		mutate func(*models.CreateWorkflowRequest)
	}{
		{"missing externalId", func(r *models.CreateWorkflowRequest) { r.ExternalID = "" }},
		{"missing executorGroup", func(r *models.CreateWorkflowRequest) { r.ExecutorGroup = "" }},
		{"missing workflowType", func(r *models.CreateWorkflowRequest) { r.WorkflowType = "" }},
		{"missing businessKey", func(r *models.CreateWorkflowRequest) { r.BusinessKey = "" }},
	}
	for _, f := range fields {
		t.Run(f.name, func(t *testing.T) {
			req := validCreateRequest("ext-x")
			f.mutate(&req)
			if err := validateCreateWorkflow(t.Context(), req); err == nil {
				t.Error("expected a validation error, got nil")
			}
		})
	}
}

func TestHandleCreateWorkflow(t *testing.T) {
	rc := newRealController(t)

	t.Run("creates and returns the id", func(t *testing.T) {
		req := postJSON(t, "/api/workflows", validCreateRequest("ext-create"))
		w := httptest.NewRecorder()

		rc.Ctrl.handleCreateWorkflow(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200; body %s", w.Code, w.Body.String())
		}
		var got models.CreateWorkflowResponse
		if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if got.ID == 0 {
			t.Error("response id = 0, want the new workflow id")
		}

		stored, err := rc.Workflow.FindByExternalId("ext-create")
		if err != nil || stored == nil {
			t.Fatalf("workflow was not persisted: %v", err)
		}
		if stored.Status != "NEW" {
			t.Errorf("status = %q, want NEW", stored.Status)
		}
	})

	t.Run("returns the existing id when the external id is reused", func(t *testing.T) {
		first := postJSON(t, "/api/workflows", validCreateRequest("ext-dup"))
		w1 := httptest.NewRecorder()
		rc.Ctrl.handleCreateWorkflow(w1, first)
		if w1.Code != http.StatusOK {
			t.Fatalf("first create failed: %s", w1.Body.String())
		}

		second := postJSON(t, "/api/workflows", validCreateRequest("ext-dup"))
		w2 := httptest.NewRecorder()
		rc.Ctrl.handleCreateWorkflow(w2, second)
		if w2.Code != http.StatusOK {
			t.Fatalf("second create failed: %s", w2.Body.String())
		}

		var a, b models.CreateWorkflowResponse
		_ = json.Unmarshal(w1.Body.Bytes(), &a)
		_ = json.Unmarshal(w2.Body.Bytes(), &b)
		if a.ID != b.ID {
			t.Errorf("ids differ for the same external id: %d vs %d", a.ID, b.ID)
		}
	})

	t.Run("rejects an incomplete request", func(t *testing.T) {
		bad := validCreateRequest("ext-bad")
		bad.WorkflowType = ""
		req := postJSON(t, "/api/workflows", bad)
		w := httptest.NewRecorder()

		rc.Ctrl.handleCreateWorkflow(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want 400", w.Code)
		}
	})

	t.Run("rejects invalid json", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/workflows", bytes.NewReader([]byte("{nope")))
		w := httptest.NewRecorder()

		rc.Ctrl.handleCreateWorkflow(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want 400", w.Code)
		}
	})

	t.Run("rejects unknown fields", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/workflows",
			bytes.NewReader([]byte(`{"externalId":"x","surprise":true}`)))
		w := httptest.NewRecorder()

		rc.Ctrl.handleCreateWorkflow(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want 400 for an unknown field", w.Code)
		}
	})

	t.Run("wrong method", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/workflows", nil)
		w := httptest.NewRecorder()

		rc.Ctrl.handleCreateWorkflow(w, req)

		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("status = %d, want 405", w.Code)
		}
	})
}

func TestHandleGetWorkflowById(t *testing.T) {
	rc := newRealController(t)
	id := rc.seedWorkflow(t, "ext-byid")

	t.Run("found", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/workflows/1", nil)
		req.SetPathValue("id", itoa(id))
		w := httptest.NewRecorder()

		rc.Ctrl.handleGetWorkflowById(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200; body %s", w.Code, w.Body.String())
		}
	})

	t.Run("unknown id", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/workflows/9999", nil)
		req.SetPathValue("id", "9999")
		w := httptest.NewRecorder()

		rc.Ctrl.handleGetWorkflowById(w, req)

		if w.Code == http.StatusOK {
			t.Error("status = 200 for an unknown workflow, want an error")
		}
	})

	t.Run("non-numeric id", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/workflows/abc", nil)
		req.SetPathValue("id", "abc")
		w := httptest.NewRecorder()

		rc.Ctrl.handleGetWorkflowById(w, req)

		if w.Code == http.StatusOK {
			t.Error("status = 200 for a non-numeric id, want an error")
		}
	})

	t.Run("wrong method", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/workflows/1", nil)
		w := httptest.NewRecorder()

		rc.Ctrl.handleGetWorkflowById(w, req)

		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("status = %d, want 405", w.Code)
		}
	})
}

func TestHandleCreateAndWaitWorkflow(t *testing.T) {
	rc := newRealController(t)

	t.Run("times out waiting for a state the engine never reaches", func(t *testing.T) {
		// No engine is running, so the workflow stays in NEW. The handler must
		// give up after waitSeconds rather than block forever.
		body := models.CreateAndWaitRequest{
			CreateWorkflowRequest: validCreateRequest("ext-caw"),
			WaitSeconds:           1,
			CheckSeconds:          1,
			WaitForStates:         []string{"Finish"},
		}
		req := postJSON(t, "/api/createAndWait", body)
		w := httptest.NewRecorder()

		rc.Ctrl.handleCreateAndWaitWorkflow(w, req)

		if w.Code == 0 {
			t.Fatal("handler did not write a response")
		}
		if stored, _ := rc.Workflow.FindByExternalId("ext-caw"); stored == nil {
			t.Error("the workflow was not created before waiting")
		}
	})

	t.Run("rejects invalid json", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/createAndWait", bytes.NewReader([]byte("{bad")))
		w := httptest.NewRecorder()

		rc.Ctrl.handleCreateAndWaitWorkflow(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want 400", w.Code)
		}
	})

	t.Run("wrong method", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/createAndWait", nil)
		w := httptest.NewRecorder()

		rc.Ctrl.handleCreateAndWaitWorkflow(w, req)

		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("status = %d, want 405", w.Code)
		}
	})
}

func TestHandleUpdateWorkflowStateAndWait(t *testing.T) {
	rc := newRealController(t)
	rc.seedWorkflow(t, "ext-usaw")

	t.Run("times out when the target state is never reached", func(t *testing.T) {
		body := models.UpdateWorkflowStateAndWaitRequest{
			UpdateWorkflowStateRequest: models.UpdateWorkflowStateRequest{State: "Review"},
			UpdateStateVarRequest:      models.UpdateStateVarRequest{Key: "k", Value: "v"},
			WaitSeconds:                1,
			CheckSeconds:               1,
			FromStates:                 []string{"Init"},
			WaitForStates:              []string{"Finish"},
		}
		req := postJSON(t, "/api/workflows/ext-usaw/stateAndWait", body)
		req.SetPathValue("externalId", "ext-usaw")
		w := httptest.NewRecorder()

		rc.Ctrl.handleUpdateWorkflowStateAndWait(w, req)

		if w.Code == 0 {
			t.Fatal("handler did not write a response")
		}
	})

	t.Run("rejects invalid json", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/workflows/ext-usaw/stateAndWait", bytes.NewReader([]byte("{bad")))
		req.SetPathValue("externalId", "ext-usaw")
		w := httptest.NewRecorder()

		rc.Ctrl.handleUpdateWorkflowStateAndWait(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want 400", w.Code)
		}
	})

	t.Run("wrong method", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/workflows/ext-usaw/stateAndWait", nil)
		w := httptest.NewRecorder()

		rc.Ctrl.handleUpdateWorkflowStateAndWait(w, req)

		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("status = %d, want 405", w.Code)
		}
	})
}
