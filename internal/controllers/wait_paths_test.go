package controllers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/RealZimboGuy/gopherflow/internal/repository"
	"github.com/RealZimboGuy/gopherflow/pkg/gopherflow/core"
	"github.com/RealZimboGuy/gopherflow/pkg/gopherflow/domain"
	"github.com/RealZimboGuy/gopherflow/pkg/gopherflow/models"
)

func TestHandleCreateAndWaitReturnsWhenTheStateAlreadyMatches(t *testing.T) {
	rc := newRealController(t)

	// The workflow is created in its initial state, so waiting for that state
	// succeeds on the first tick rather than timing out.
	body := models.CreateAndWaitRequest{
		CreateWorkflowRequest: validCreateRequest("ext-caw-hit"),
		WaitSeconds:           10,
		CheckSeconds:          1,
		WaitForStates:         []string{"Init"},
	}
	req := postJSON(t, "/api/createAndWait", body)
	w := httptest.NewRecorder()

	rc.Ctrl.handleCreateAndWaitWorkflow(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body %s", w.Code, w.Body.String())
	}
	var got models.WorkflowApiResponse
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got.State != "Init" {
		t.Errorf("state = %q, want Init", got.State)
	}
	if got.ExternalID != "ext-caw-hit" {
		t.Errorf("externalId = %q, want ext-caw-hit", got.ExternalID)
	}
}

func TestHandleCreateAndWaitWithNoWaitStatesReturnsImmediately(t *testing.T) {
	rc := newRealController(t)

	body := models.CreateAndWaitRequest{
		CreateWorkflowRequest: validCreateRequest("ext-caw-any"),
		WaitSeconds:           10,
		CheckSeconds:          1,
		// No WaitForStates means any state satisfies the wait.
	}
	req := postJSON(t, "/api/createAndWait", body)
	w := httptest.NewRecorder()

	rc.Ctrl.handleCreateAndWaitWorkflow(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body %s", w.Code, w.Body.String())
	}
}

func TestHandleCreateAndWaitRejectsAnIncompleteRequest(t *testing.T) {
	rc := newRealController(t)

	bad := validCreateRequest("ext-caw-bad")
	bad.BusinessKey = ""
	req := postJSON(t, "/api/createAndWait", models.CreateAndWaitRequest{
		CreateWorkflowRequest: bad,
		WaitSeconds:           1,
		CheckSeconds:          1,
	})
	w := httptest.NewRecorder()

	rc.Ctrl.handleCreateAndWaitWorkflow(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestHandleUpdateWorkflowStateAndWaitSuccess(t *testing.T) {
	rc := newRealController(t)
	rc.seedWorkflow(t, "ext-usaw-hit")

	// Move Init -> Review and wait for Review, which the update itself produces.
	body := models.UpdateWorkflowStateAndWaitRequest{
		UpdateWorkflowStateRequest: models.UpdateWorkflowStateRequest{State: "Review"},
		UpdateStateVarRequest:      models.UpdateStateVarRequest{Key: "colour", Value: "blue"},
		WaitSeconds:                10,
		CheckSeconds:               1,
		FromStates:                 []string{"Init"},
		WaitForStates:              []string{"Review"},
	}
	req := postJSON(t, "/api/workflows/ext-usaw-hit/stateAndWait", body)
	req.SetPathValue("id", "ext-usaw-hit")
	w := httptest.NewRecorder()

	rc.Ctrl.handleUpdateWorkflowStateAndWait(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body %s", w.Code, w.Body.String())
	}

	stored, err := rc.Workflow.FindByExternalId("ext-usaw-hit")
	if err != nil {
		t.Fatalf("FindByExternalId: %v", err)
	}
	if stored.State != "Review" {
		t.Errorf("state = %q, want Review", stored.State)
	}
}

func TestHandleUpdateWorkflowStateAndWaitRejectsAWrongFromState(t *testing.T) {
	rc := newRealController(t)
	rc.seedWorkflow(t, "ext-usaw-from")

	body := models.UpdateWorkflowStateAndWaitRequest{
		UpdateWorkflowStateRequest: models.UpdateWorkflowStateRequest{State: "Finish"},
		WaitSeconds:                1,
		CheckSeconds:               1,
		FromStates:                 []string{"Approve"}, // the workflow is in Init
		WaitForStates:              []string{"Finish"},
	}
	req := postJSON(t, "/api/workflows/ext-usaw-from/stateAndWait", body)
	req.SetPathValue("id", "ext-usaw-from")
	w := httptest.NewRecorder()

	rc.Ctrl.handleUpdateWorkflowStateAndWait(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400 for a from-state mismatch; body %s", w.Code, w.Body.String())
	}
}

func TestHandleUpdateWorkflowStateAndWaitRequiresAState(t *testing.T) {
	rc := newRealController(t)
	rc.seedWorkflow(t, "ext-usaw-nostate")

	body := models.UpdateWorkflowStateAndWaitRequest{
		WaitSeconds:  1,
		CheckSeconds: 1,
	}
	req := postJSON(t, "/api/workflows/ext-usaw-nostate/stateAndWait", body)
	req.SetPathValue("id", "ext-usaw-nostate")
	w := httptest.NewRecorder()

	rc.Ctrl.handleUpdateWorkflowStateAndWait(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400 when state is missing", w.Code)
	}
}

func TestHandleUpdateWorkflowStateAndWaitUnknownWorkflow(t *testing.T) {
	rc := newRealController(t)

	req := postJSON(t, "/api/workflows/nope/stateAndWait", models.UpdateWorkflowStateAndWaitRequest{
		UpdateWorkflowStateRequest: models.UpdateWorkflowStateRequest{State: "Review"},
		WaitSeconds:                1,
		CheckSeconds:               1,
	})
	req.SetPathValue("id", "nope")
	w := httptest.NewRecorder()

	rc.Ctrl.handleUpdateWorkflowStateAndWait(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", w.Code)
	}
}

func TestHandleListWorkflowDefinitions(t *testing.T) {
	rc := newRealController(t)

	if err := rc.Defs.Save(&domain.WorkflowDefinition{
		Name: "DemoWorkflow", Description: "demo",
	}); err != nil {
		t.Fatalf("seed definition: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/definitions", nil)
	w := httptest.NewRecorder()

	rc.Ctrl.handleListWorkflowDefinitions(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body %s", w.Code, w.Body.String())
	}
	var defs []domain.WorkflowDefinition
	if err := json.Unmarshal(w.Body.Bytes(), &defs); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(defs) == 0 {
		t.Error("expected at least the seeded definition")
	}
}

func TestHandleGetActionsForWorkflow(t *testing.T) {
	rc := newRealController(t)
	wfID := rc.seedWorkflow(t, "ext-actions-api")

	if _, err := rc.Actions.Save(&domain.WorkflowAction{
		WorkflowID: wfID, ExecutorID: 1, Type: "LOG", Name: "Init", Text: "entry",
	}); err != nil {
		t.Fatalf("seed action: %v", err)
	}

	ac := NewActionsController(rc.Workflow, rc.Actions, rc.Users)

	t.Run("returns the actions", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/actions/1", nil)
		req.SetPathValue("id", itoa(wfID))
		w := httptest.NewRecorder()

		ac.handleGetActionsForWorkflow(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200; body %s", w.Code, w.Body.String())
		}
	})

	t.Run("non-numeric id", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/actions/abc", nil)
		req.SetPathValue("id", "abc")
		w := httptest.NewRecorder()

		ac.handleGetActionsForWorkflow(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want 400", w.Code)
		}
	})

	t.Run("wrong method", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/actions/1", nil)
		w := httptest.NewRecorder()

		ac.handleGetActionsForWorkflow(w, req)

		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("status = %d, want 405", w.Code)
		}
	})
}

func TestHandleGetExecutors(t *testing.T) {
	rc := newRealController(t)

	execRepo := repository.NewExecutorRepository(rc.DB, engineClock())
	if _, err := execRepo.Save(&domain.Executor{Name: "worker-api"}); err != nil {
		t.Fatalf("seed executor: %v", err)
	}

	ec := NewExecutorsController(execRepo, rc.Users)

	t.Run("returns the executors", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/executors", nil)
		w := httptest.NewRecorder()

		ec.handleGetExecutors(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200; body %s", w.Code, w.Body.String())
		}
		var execs []domain.Executor
		if err := json.Unmarshal(w.Body.Bytes(), &execs); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if len(execs) == 0 {
			t.Error("expected the seeded executor")
		}
	})

	t.Run("wrong method", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/executors", nil)
		w := httptest.NewRecorder()

		ec.handleGetExecutors(w, req)

		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("status = %d, want 405", w.Code)
		}
	})
}

func engineClock() core.Clock { return core.NewRealClock() }
