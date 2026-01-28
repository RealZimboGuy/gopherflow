package controllers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/RealZimboGuy/gopherflow/pkg/gopherflow/domain"
)

func TestActionsController_GetActionsForWorkflow_Success(t *testing.T) {
	mockActionRepo := &MockWorkflowActionRepo{
		FindAllByWorkflowIDFunc: func(workflowID int64) (*[]domain.WorkflowAction, error) {
			return &[]domain.WorkflowAction{
				{ID: 1, WorkflowID: workflowID, Name: "Action1", Type: "INFO"},
			}, nil
		},
	}

	c := NewActionsController(&MockWorkflowRepo{}, mockActionRepo, &MockUserRepo{})

	req := httptest.NewRequest("GET", "/api/actions/byWorkflowId/10", nil)
	req.SetPathValue("id", "10") // Go 1.22 routing
	w := httptest.NewRecorder()

	c.handleGetActionsForWorkflow(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var actions []domain.WorkflowAction
	if err := json.NewDecoder(resp.Body).Decode(&actions); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	if len(actions) != 1 {
		t.Errorf("Expected 1 action, got %d", len(actions))
	}
	if actions[0].WorkflowID != 10 {
		t.Errorf("Expected WorkflowID 10, got %d", actions[0].WorkflowID)
	}
}

func TestActionsController_GetActionsForWorkflow_InvalidID(t *testing.T) {
	c := NewActionsController(&MockWorkflowRepo{}, &MockWorkflowActionRepo{}, &MockUserRepo{})

	req := httptest.NewRequest("GET", "/api/actions/byWorkflowId/abc", nil)
	req.SetPathValue("id", "abc")
	w := httptest.NewRecorder()

	c.handleGetActionsForWorkflow(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", resp.StatusCode)
	}
}
