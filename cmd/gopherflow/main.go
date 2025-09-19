package main

import (
	"gopherflow/internal/engine"
	"gopherflow/internal/workflows"
	"gopherflow/pkg/gopherflow"
	"log/slog"
	"reflect"
)

func main() {

	//you may do your own logger setup here or use this default one with slog
	gopherflow.SetupLogger()

	engine.WorkflowRegistry = map[string]reflect.Type{
		"DemoWorkflow":  reflect.TypeOf(workflows.DemoWorkflow{}),
		"GetIpWorkflow": reflect.TypeOf(workflows.GetIpWorkflow{}),
	}
	if err := gopherflow.Start(); err != nil {
		slog.Error("Engine exited with error", "error", err)
	}
}
