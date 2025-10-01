package main

import (
	"log/slog"

	"github.com/RealZimboGuy/gopherflow/internal/workflows"
	"github.com/RealZimboGuy/gopherflow/pkg/gopherflow"
	"github.com/RealZimboGuy/gopherflow/pkg/gopherflow/core"
)

func main() {

	//you may do your own logger setup here or use this default one with slog
	gopherflow.SetupLogger()

	gopherflow.WorkflowRegistry = map[string]func() core.Workflow{
		"DemoWorkflow": func() core.Workflow {
			return &workflows.DemoWorkflow{}
		},
		"GetIpWorkflow": func() core.Workflow {
			// You can now inject dependencies here
			return &workflows.GetIpWorkflow{
				// HTTPClient: httpClient,
				// Logger: logger,
				// MyService: myService,
			}
		},
	}

	app := gopherflow.Setup()

	if err := app.Run(); err != nil {
		slog.Error("Engine exited with error", "error", err)
	}

}
