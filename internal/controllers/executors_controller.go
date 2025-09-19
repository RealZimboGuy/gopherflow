package controllers

import (
	"encoding/json"
	"github.com/RealZimboGuy/gopherflow/internal/repository"
	"log/slog"
	"net/http"
)

type ExecutorsController struct {
	AuthController
	ExecutorsRepo *repository.ExecutorRepository
}

func NewExecutorsController(
	workflowExecutorsRepo *repository.ExecutorRepository, userRepo *repository.UserRepository) *ExecutorsController {
	return &ExecutorsController{
		ExecutorsRepo: workflowExecutorsRepo,
		AuthController: AuthController{
			UserRepo: userRepo,
		},
	}
}

func (c *ExecutorsController) handleGetExecutors(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	//if the external id is a duplicate, we return the existing workflow
	results, err := c.ExecutorsRepo.GetExecutorsByLastActive(20)
	if err != nil {
		slog.Error("Failed to search executors", "error", err)
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
