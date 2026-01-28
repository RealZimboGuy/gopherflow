package engine

import (
	"context"
	"log/slog"
	"strconv"

	"github.com/RealZimboGuy/gopherflow/pkg/gopherflow/core"
)

// Worker function that processes workflows from the queue
func Worker(ctx context.Context, id int, executorID int64, workflowRepository WorkflowRepo, workflowActionRepository WorkflowActionRepo, workflowQueue <-chan core.Workflow) {
	for {
		for {
			select {
			case <-ctx.Done():
				slog.InfoContext(ctx, "Worker shutting down", "worker_id", id)
				return

			case wf, ok := <-workflowQueue:
				if !ok {
					slog.InfoContext(ctx, "Workflow queue closed", "worker_id", id)
					return
				}

				slog.InfoContext(ctx, "Worker starting workflow", "worker_id", id)
				RunWorkflow(ctx, wf, workflowRepository, workflowActionRepository, executorID, strconv.Itoa(id))
				slog.InfoContext(ctx, "Worker finished workflow", "worker_id", id)
			}
		}
	}
}
