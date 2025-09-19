package engine

import (
	"log/slog"
	"strconv"

	"github.com/RealZimboGuy/gopherflow/internal/repository"
)

// Worker function that processes workflows from the queue
func Worker(id int, executorID int64, workflowRepository repository.WorkflowRepository, workflowActionRepository repository.WorkflowActionRepository, workflowQueue <-chan Workflow) {
	for {
		wf := <-workflowQueue // blocks until a job arrives
		slog.Info("Worker starting workflow", "worker_id", id)
		RunWorkflow(wf, workflowRepository, workflowActionRepository, executorID, strconv.Itoa(id))
		slog.Info("Worker finished workflow", "worker_id", id)
	}
}
