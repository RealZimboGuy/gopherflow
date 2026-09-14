package workflows

import (
	"database/sql"
	"testing"

	"github.com/RealZimboGuy/gopherflow/pkg/gopherflow/core"
	"github.com/RealZimboGuy/gopherflow/pkg/gopherflow/domain"
	"github.com/RealZimboGuy/gopherflow/pkg/gopherflow/models"
	"github.com/RealZimboGuy/gopherflow/pkg/gopherflow/workflow_helpers"
)

// assertWorkflowShape checks the invariants every workflow definition must
// hold: the initial state exists, every transition target is a declared state,
// and there is at least one end state for the engine to finish on.
func assertWorkflowShape(t *testing.T, wf core.Workflow) {
	t.Helper()

	states := wf.GetAllStates()
	if len(states) == 0 {
		t.Fatal("GetAllStates returned no states")
	}

	known := make(map[string]bool, len(states))
	var hasEnd bool
	for _, s := range states {
		known[s.Name] = true
		if s.StateType == models.StateEnd {
			hasEnd = true
		}
	}
	if !hasEnd {
		t.Error("no state is declared as models.StateEnd, so the workflow can never finish")
	}

	if initial := wf.InitialState(); !known[initial] {
		t.Errorf("InitialState() = %q, which is not in GetAllStates()", initial)
	}

	for from, tos := range wf.StateTransitions() {
		if !known[from] {
			t.Errorf("transition source %q is not a declared state", from)
		}
		for _, to := range tos {
			if !known[to] {
				t.Errorf("transition %q -> %q targets an undeclared state", from, to)
			}
		}
	}

	if wf.Description() == "" {
		t.Error("Description() is empty")
	}

	rc := wf.GetRetryConfig()
	if rc.MaxRetryCount <= 0 {
		t.Errorf("MaxRetryCount = %d, want a positive value", rc.MaxRetryCount)
	}
	if rc.RetryIntervalMin <= 0 || rc.RetryIntervalMax < rc.RetryIntervalMin {
		t.Errorf("retry interval range is invalid: min %v, max %v", rc.RetryIntervalMin, rc.RetryIntervalMax)
	}
}

func TestDemoWorkflowShape(t *testing.T)       { assertWorkflowShape(t, &DemoWorkflow{}) }
func TestGetIpWorkflowShape(t *testing.T)      { assertWorkflowShape(t, &GetIpWorkflow{}) }
func TestDemoParentWorkflowShape(t *testing.T) { assertWorkflowShape(t, &DemoParentWorkflow{}) }
func TestDemoChildWorkflowShape(t *testing.T)  { assertWorkflowShape(t, &DemoChildWorkflow{}) }

func TestDemoWorkflowSetupAccessors(t *testing.T) {
	var wf DemoWorkflow
	row := &domain.Workflow{
		ID:        7,
		StateVars: sql.NullString{String: `{"name":"existing"}`, Valid: true},
	}

	wf.Setup(row)

	if wf.GetWorkflowData() != row {
		t.Error("GetWorkflowData did not return the workflow passed to Setup")
	}
	vars := wf.GetStateVariables()
	if vars == nil {
		t.Fatal("GetStateVariables returned nil")
	}
	if vars["name"] != "existing" {
		t.Errorf("state vars were not loaded: %v", vars)
	}
}

func TestDemoWorkflowInitSetsStateVariables(t *testing.T) {
	var wf DemoWorkflow
	wf.Setup(&domain.Workflow{ID: 1})

	next, err := wf.Init(t.Context())
	if err != nil {
		t.Fatalf("Init: %v", err)
	}
	if next == nil || next.Name != StateReview {
		t.Fatalf("next = %+v, want a transition to %s", next, StateReview)
	}

	vars := wf.GetStateVariables()
	if vars[VAR_NAME] != "Julian" {
		t.Errorf("%s = %q, want Julian", VAR_NAME, vars[VAR_NAME])
	}
	if vars[VAR_AGE] != "33" {
		t.Errorf("%s = %q, want 33", VAR_AGE, vars[VAR_AGE])
	}

	enrollment, err := workflow_helpers.LoadStructFromStateVars[EnrollmentStruct](vars, "enrollment")
	if err != nil {
		t.Fatalf("the enrollment struct was not saved: %v", err)
	}
	if enrollment.Name != "Julian" || enrollment.Age != 33 {
		t.Errorf("enrollment = %+v, want {Julian 33}", *enrollment)
	}
}

func TestDemoWorkflowReviewChangesName(t *testing.T) {
	var wf DemoWorkflow
	wf.Setup(&domain.Workflow{ID: 1})
	if _, err := wf.Init(t.Context()); err != nil {
		t.Fatalf("Init: %v", err)
	}

	next, err := wf.Review(t.Context())
	if err != nil {
		t.Fatalf("Review: %v", err)
	}
	if next == nil || next.Name != StateApprove {
		t.Fatalf("next = %+v, want a transition to %s", next, StateApprove)
	}
	if got := wf.GetStateVariables()[VAR_NAME]; got != "Julian2" {
		t.Errorf("%s = %q, want Julian2", VAR_NAME, got)
	}
	if next.NextExecution.IsZero() {
		t.Error("Review did not schedule a next execution time")
	}
}

func TestDemoWorkflowApproveFinishes(t *testing.T) {
	var wf DemoWorkflow
	wf.Setup(&domain.Workflow{ID: 1})

	next, err := wf.Approve(t.Context())
	if err != nil {
		t.Fatalf("Approve: %v", err)
	}
	if next == nil || next.Name != StateFinish {
		t.Fatalf("next = %+v, want a transition to %s", next, StateFinish)
	}
}
