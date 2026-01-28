package controllers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/RealZimboGuy/gopherflow/internal/engine"
)

type ActionsController struct {
	AuthController
	WorkflowRepo       engine.WorkflowRepo
	WorkflowActionRepo engine.WorkflowActionRepo
}

func NewActionsController(workflowRepo engine.WorkflowRepo,
	workflowActionsRepo engine.WorkflowActionRepo, userRepo engine.UserRepo) *ActionsController {
	return &ActionsController{WorkflowRepo: workflowRepo,
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
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(results)
		return
	}

}
