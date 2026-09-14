package core

import (
	"database/sql"
	"testing"
	"time"

	"github.com/RealZimboGuy/gopherflow/pkg/gopherflow/domain"
)

func TestBaseWorkflowSetupInitialisesStateVariables(t *testing.T) {
	var b BaseWorkflow
	wf := &domain.Workflow{ID: 1}

	b.Setup(wf)

	if b.StateVariables == nil {
		t.Fatal("StateVariables is nil after Setup; it must be usable immediately")
	}
	if b.WorkflowState != wf {
		t.Error("WorkflowState was not set to the given workflow")
	}
}

func TestBaseWorkflowSetupParsesStateVars(t *testing.T) {
	var b BaseWorkflow
	wf := &domain.Workflow{
		ID:        1,
		StateVars: sql.NullString{String: `{"name":"Ada","age":"36"}`, Valid: true},
	}

	b.Setup(wf)

	if got := b.StateVariables["name"]; got != "Ada" {
		t.Errorf("StateVariables[name] = %q, want %q", got, "Ada")
	}
	if got := b.StateVariables["age"]; got != "36" {
		t.Errorf("StateVariables[age] = %q, want %q", got, "36")
	}
}

func TestBaseWorkflowSetupIgnoresEmptyAndNullStateVars(t *testing.T) {
	for _, sv := range []sql.NullString{
		{Valid: false},
		{String: "", Valid: true},
		{String: "null", Valid: true},
	} {
		var b BaseWorkflow
		b.Setup(&domain.Workflow{StateVars: sv})

		if b.StateVariables == nil {
			t.Fatalf("StateVariables is nil for %+v", sv)
		}
		if len(b.StateVariables) != 0 {
			t.Errorf("StateVariables = %v for %+v, want empty", b.StateVariables, sv)
		}
	}
}

func TestBaseWorkflowSetupResetsStaleValuesOnReuse(t *testing.T) {
	// The engine reuses workflow instances, so a previous run's variables must
	// not leak into the next one.
	b := BaseWorkflow{StateVariables: map[string]string{"stale": "value"}}

	b.Setup(&domain.Workflow{
		StateVars: sql.NullString{String: `{"fresh":"value"}`, Valid: true},
	})

	if _, found := b.StateVariables["stale"]; found {
		t.Error("a value from a previous run survived Setup")
	}
	if got := b.StateVariables["fresh"]; got != "value" {
		t.Errorf("StateVariables[fresh] = %q, want %q", got, "value")
	}
}

func TestBaseWorkflowSetupWithInvalidJSON(t *testing.T) {
	var b BaseWorkflow

	// Must not panic; the workflow should still be usable.
	b.Setup(&domain.Workflow{StateVars: sql.NullString{String: "{not json", Valid: true}})

	if b.StateVariables == nil {
		t.Fatal("StateVariables is nil after a parse failure")
	}
}

func TestBaseWorkflowSetChildWorkflows(t *testing.T) {
	var b BaseWorkflow
	children := []domain.Workflow{{ID: 1}, {ID: 2}}

	b.SetChildWorkflows(children)

	if len(b.ChildWorkflows) != 2 {
		t.Fatalf("ChildWorkflows has %d entries, want 2", len(b.ChildWorkflows))
	}
	if b.ChildWorkflows[0].ID != 1 || b.ChildWorkflows[1].ID != 2 {
		t.Errorf("ChildWorkflows = %+v, want ids 1 and 2", b.ChildWorkflows)
	}
}

func TestRealClock(t *testing.T) {
	c := NewRealClock()

	before := time.Now()
	got := c.Now()
	if got.Before(before.Add(-time.Second)) || got.After(time.Now().Add(time.Second)) {
		t.Errorf("Now() = %v, which is not close to the wall clock", got)
	}

	start := time.Now()
	c.Sleep(5 * time.Millisecond)
	if elapsed := time.Since(start); elapsed < 5*time.Millisecond {
		t.Errorf("Sleep returned after %v, want at least 5ms", elapsed)
	}

	select {
	case <-c.After(5 * time.Millisecond):
	case <-time.After(time.Second):
		t.Error("After did not fire within a second")
	}
}
