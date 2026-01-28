package controllers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/RealZimboGuy/gopherflow/pkg/gopherflow/domain"
)

func TestExecutorsController_GetExecutors(t *testing.T) {
	mockExecutorRepo := &MockExecutorRepo{
		GetExecutorsByLastActiveFunc: func(limit int) ([]*domain.Executor, error) {
			return []*domain.Executor{
				{ID: 1, Name: "executor1"},
			}, nil
		},
	}

	c := NewExecutorsController(mockExecutorRepo, &MockUserRepo{})

	req := httptest.NewRequest("GET", "/api/executors", nil)
	w := httptest.NewRecorder()

	c.handleGetExecutors(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var executors []domain.Executor
	if err := json.NewDecoder(resp.Body).Decode(&executors); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	if len(executors) != 1 {
		t.Errorf("Expected 1 executor, got %d", len(executors))
	}
}
