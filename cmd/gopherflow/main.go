package main

import (
	"log/slog"
	"reflect"

	"github.com/RealZimboGuy/gopherflow/internal/workflows"
	"github.com/RealZimboGuy/gopherflow/pkg/gopherflow"
)

func main() {

	//you may do your own logger setup here or use this default one with slog
	gopherflow.SetupLogger()

	gopherflow.WorkflowRegistry = map[string]reflect.Type{
		"DemoWorkflow":  reflect.TypeOf(workflows.DemoWorkflow{}),
		"GetIpWorkflow": reflect.TypeOf(workflows.GetIpWorkflow{}),
	}
	app := gopherflow.Setup()

	if err := app.Run(); err != nil {
		slog.Error("Engine exited with error", "error", err)
	}

}
