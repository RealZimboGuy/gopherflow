package engine

import (
	"testing"

	"github.com/RealZimboGuy/gopherflow/pkg/gopherflow/core"
)

func TestCreateWorkflowInstanceReturnsAFreshInstance(t *testing.T) {
	registry := map[string]func() core.Workflow{
		"Known": func() core.Workflow { return &MockWorkflow{} },
	}
	wm := NewWorkflowManager(nil, nil, nil, nil, &registry, nil)

	first, err := CreateWorkflowInstance(wm, "Known")
	if err != nil {
		t.Fatalf("CreateWorkflowInstance: %v", err)
	}
	if first == nil {
		t.Fatal("CreateWorkflowInstance returned a nil workflow")
	}

	// The registry holds factories, so each call must yield a separate instance;
	// the engine reuses the manager across concurrent workflows.
	second, err := CreateWorkflowInstance(wm, "Known")
	if err != nil {
		t.Fatalf("CreateWorkflowInstance (second): %v", err)
	}
	if first == second {
		t.Error("the same instance was returned twice; state would leak between workflows")
	}
}

func TestCreateWorkflowInstanceUnknownType(t *testing.T) {
	registry := map[string]func() core.Workflow{}
	wm := NewWorkflowManager(nil, nil, nil, nil, &registry, nil)

	got, err := CreateWorkflowInstance(wm, "NoSuchWorkflow")
	if err == nil {
		t.Fatal("expected an error for an unregistered workflow type, got nil")
	}
	if got != nil {
		t.Errorf("expected a nil workflow alongside the error, got %+v", got)
	}
}
