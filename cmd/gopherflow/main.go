package main

import (
	"context"
	"log/slog"

	"github.com/RealZimboGuy/gopherflow/internal/workflows"
	"github.com/RealZimboGuy/gopherflow/pkg/gopherflow"
	"github.com/RealZimboGuy/gopherflow/pkg/gopherflow/core"
)

func main() {

	//you may do your own logger setup here or use this default one with slog
	ctx := context.Background()

	gopherflow.SetupLogger(slog.LevelInfo)

	workflowRegistry := map[string]func() core.Workflow{
		"DemoWorkflow": func() core.Workflow {
			return &workflows.DemoWorkflow{
				Clock: core.NewRealClock(),
			}
		},
		"GetIpWorkflow": func() core.Workflow {
			// You can inject dependencies here
			return &workflows.GetIpWorkflow{
				// HTTPClient: httpClient,
				// MyService: myService,
			}
		},
		// Parent-Child Demo Workflows
		"DemoParentWorkflow": func() core.Workflow {
			return &workflows.DemoParentWorkflow{
				Clock: core.NewRealClock(),
			}
		},
		"DemoChildWorkflow": func() core.Workflow {
			return &workflows.DemoChildWorkflow{
				Clock: core.NewRealClock(),
			}
		},
	}

	app := gopherflow.Setup(workflowRegistry)

	if err := app.Run(ctx); err != nil {
		slog.Error("Engine exited with error", "error", err)
	}

}
