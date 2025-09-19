package controllers

import (
	"encoding/json"
	"gopherflow/internal/engine"
	"gopherflow/internal/repository"
	"log/slog"
	"net/http"
	"strconv"
)

type ActionsController struct {
	AuthController
	WorkflowRepo       *repository.WorkflowRepository
	WorkflowActionRepo *repository.WorkflowActionRepository
	WorkflowManager    *engine.WorkflowManager
}

func NewActionsController(workflowRepo *repository.WorkflowRepository, workflowManager *engine.WorkflowManager,
	workflowActionsRepo *repository.WorkflowActionRepository, userRepo *repository.UserRepository) *ActionsController {
	return &ActionsController{WorkflowRepo: workflowRepo, WorkflowManager: workflowManager,
		WorkflowActionRepo: workflowActionsRepo, AuthController: AuthController{
			UserRepo: userRepo,
		}}
}

func (c *ActionsController) handleGetActionsForWorkflow(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr) // convert to int
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	//if the external id is a duplicate, we return the existing workflow
	results, err := c.WorkflowActionRepo.FindAllByWorkflowID(int64(id))
	if err != nil {
		slog.Error("Failed to search workflows", "error", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if results != nil {

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(results)
		return
	}

}
