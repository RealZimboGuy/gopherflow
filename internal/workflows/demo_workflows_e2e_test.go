package workflows_test

import (
	"context"
	"database/sql"
	"strings"
	"testing"
	"time"

	"github.com/RealZimboGuy/gopherflow/internal/engine"
	"github.com/RealZimboGuy/gopherflow/internal/repository"
	"github.com/RealZimboGuy/gopherflow/internal/workflows"
	"github.com/RealZimboGuy/gopherflow/pkg/gopherflow/core"
	"github.com/RealZimboGuy/gopherflow/pkg/gopherflow/domain"
	"github.com/RealZimboGuy/gopherflow/pkg/gopherflow/models"
)

// stubRepo is a hand-rolled engine.WorkflowRepo whose state can be inspected
// and mutated between executor invocations - the way a real DB would behave
// across polling cycles.
type stubRepo struct {
	state    string
	status   string
	vars     string
	children []domain.Workflow

	wakeCalls int
	saved     []*domain.Workflow
}

func (r *stubRepo) UpdateWorkflowStatus(_ int64, status string) error { r.status = status; return nil }
func (r *stubRepo) UpdateWorkflowStartingTime(_ int64) error          { return nil }
func (r *stubRepo) UpdateState(_ int64, state string) error           { r.state = state; return nil }
func (r *stubRepo) SaveWorkflowVariables(_ int64, vars string) error  { r.vars = vars; return nil }
func (r *stubRepo) SaveWorkflowVariablesAndTouch(_ int64, vars string) error {
	r.vars = vars
	return nil
}
func (r *stubRepo) WakeParentWorkflow(_ int64) error { r.wakeCalls++; return nil }
func (r *stubRepo) Save(wf *domain.Workflow) (int64, error) {
	r.saved = append(r.saved, wf)
	return int64(1000 + len(r.saved)), nil
}
func (r *stubRepo) FindByID(id int64) (*domain.Workflow, error) {
	return &domain.Workflow{ID: id, Status: "NEW"}, nil
}
func (r *stubRepo) UpdateNextActivationSpecific(_ int64, _ time.Time) error { return nil }
func (r *stubRepo) UpdateNextActivationOffset(_ int64, _ string) error      { return nil }
func (r *stubRepo) ClearExecutorId(_ int64) error                           { return nil }
func (r *stubRepo) IncrementRetryCounterAndSetNextActivation(_ int64, _ time.Time) error {
	return nil
}
func (r *stubRepo) GetChildrenByParentID(_ int64, _ bool) (*[]domain.Workflow, error) {
	return &r.children, nil
}
func (r *stubRepo) FindPendingWorkflows(_ int, _ string) (*[]domain.Workflow, error) {
	return nil, nil
}
func (r *stubRepo) MarkWorkflowAsScheduledForExecution(_ int64, _ int64, _ time.Time) bool {
	return true
}
func (r *stubRepo) FindStuckWorkflows(_ string, _ string, _ int) (*[]domain.Workflow, error) {
	return nil, nil
}
func (r *stubRepo) LockWorkflowByModified(_ int64, _ time.Time) bool { return true }
func (r *stubRepo) SearchWorkflows(_ models.SearchWorkflowRequest) (*[]domain.Workflow, error) {
	return nil, nil
}
func (r *stubRepo) GetTopExecuting(_ int) (*[]domain.Workflow, error)  { return nil, nil }
func (r *stubRepo) GetNextToExecute(_ int) (*[]domain.Workflow, error) { return nil, nil }
func (r *stubRepo) GetWorkflowOverview() ([]repository.WorkflowOverviewRow, error) {
	return nil, nil
}
func (r *stubRepo) GetDefinitionStateOverview(_ string) ([]repository.DefinitionStateRow, error) {
	return nil, nil
}
func (r *stubRepo) FindByExternalId(_ string) (*domain.Workflow, error) { return nil, nil }

// stubActions records action types/names so we can assert the executor's
// transition log matches expectations.
type stubActions struct{ actions []string }

func (s *stubActions) Save(a *domain.WorkflowAction) (int64, error) {
	s.actions = append(s.actions, a.Type+":"+a.Name)
	return 1, nil
}
func (s *stubActions) FindAllByWorkflowID(_ int64) (*[]domain.WorkflowAction, error) {
	return nil, nil
}

// runChildStep mirrors what WorkflowManager does for one execution pass.
func runChildStep(t *testing.T, repo *stubRepo, clock core.Clock) []string {
	t.Helper()
	wf := &workflows.DemoChildWorkflow{Clock: clock}
	wf.Setup(&domain.Workflow{
		ID:               1,
		State:            repo.state,
		StateVars:        sql.NullString{String: repo.vars, Valid: repo.vars != ""},
		ParentWorkflowID: sql.NullInt64{Int64: 99, Valid: true},
	})
	actions := &stubActions{}
	engine.RunWorkflow(context.Background(), wf, repo, actions, 1, "test-worker")
	return actions.actions
}

func runParentStep(t *testing.T, repo *stubRepo, clock core.Clock) []string {
	t.Helper()
	wf := &workflows.DemoParentWorkflow{Clock: clock}
	wf.Setup(&domain.Workflow{
		ID:        42,
		State:     repo.state,
		StateVars: sql.NullString{String: repo.vars, Valid: repo.vars != ""},
	})
	actions := &stubActions{}
	engine.RunWorkflow(context.Background(), wf, repo, actions, 1, "test-worker")
	return actions.actions
}

func TestDemoChildWorkflow_EndToEnd(t *testing.T) {
	clock := core.NewRealClock()
	repo := &stubRepo{}

	// Pass 1: ChildInit -> ChildProcessing -> pauses on ChildWakeParent (NextExecution set)
	acts1 := runChildStep(t, repo, clock)
	if repo.state != workflows.ChildWakeParent {
		t.Fatalf("after pass 1 expected state=%q, got %q", workflows.ChildWakeParent, repo.state)
	}
	if repo.status == "FINISHED" {
		t.Fatalf("after pass 1 child should not be FINISHED yet")
	}
	if !containsAction(acts1, "TRANSITION:ChildInit") || !containsAction(acts1, "TRANSITION:ChildProcessing") {
		t.Errorf("pass 1 missing expected transitions, got: %v", acts1)
	}

	// Pass 2: ChildWakeParent -> ChildFinish (StateEnd) -> FINISHED. Wakes parent.
	acts2 := runChildStep(t, repo, clock)
	if repo.state != workflows.ChildFinish {
		t.Fatalf("after pass 2 expected state=%q, got %q", workflows.ChildFinish, repo.state)
	}
	if repo.status != "FINISHED" {
		t.Fatalf("after pass 2 expected status=FINISHED, got %q", repo.status)
	}
	if repo.wakeCalls != 1 {
		t.Fatalf("expected one WakeParentWorkflow call, got %d", repo.wakeCalls)
	}
	if !containsAction(acts2, "TRANSITION:ChildWakeParent") || !containsAction(acts2, "END:ChildFinish") {
		t.Errorf("pass 2 missing expected transitions/end, got: %v", acts2)
	}

	// State vars should have carried the processing result through.
	if !strings.Contains(repo.vars, "Processed after") {
		t.Errorf("expected state vars to retain processing result, got %q", repo.vars)
	}
}

func TestDemoParentWorkflow_EndToEnd(t *testing.T) {
	clock := core.NewRealClock()
	repo := &stubRepo{}

	// Pass 1: ParentInit -> ParentSpawnChildren -> ParentWaitForChildren (paused, NextExecutionOffset).
	// Two child workflows should be created via Save.
	acts1 := runParentStep(t, repo, clock)
	if repo.state != workflows.ParentWaitForChildren {
		t.Fatalf("after pass 1 expected state=%q, got %q", workflows.ParentWaitForChildren, repo.state)
	}
	if len(repo.saved) != 2 {
		t.Fatalf("expected 2 child workflows created, got %d", len(repo.saved))
	}
	for i, c := range repo.saved {
		if !c.ParentWorkflowID.Valid || c.ParentWorkflowID.Int64 != 42 {
			t.Errorf("saved child %d has wrong parent_id: %+v", i, c.ParentWorkflowID)
		}
		if c.WorkflowType != "DemoChildWorkflow" {
			t.Errorf("saved child %d has wrong type: %q", i, c.WorkflowType)
		}
	}
	if !containsAction(acts1, "CHILD_CREATED:ParentWaitForChildren") {
		t.Errorf("pass 1 missing CHILD_CREATED action, got: %v", acts1)
	}

	// Pass 2: ParentWaitForChildren with no children done -> stay in ParentWaitForChildren.
	repo.children = []domain.Workflow{
		{ID: 1001, Status: "EXECUTING", BusinessKey: "child-1-of-42"},
		{ID: 1002, Status: "EXECUTING", BusinessKey: "child-2-of-42"},
	}
	runParentStep(t, repo, clock)
	if repo.state != workflows.ParentWaitForChildren {
		t.Fatalf("after pass 2 expected state still %q, got %q", workflows.ParentWaitForChildren, repo.state)
	}
	if repo.status == "FINISHED" {
		t.Fatalf("after pass 2 parent should not be FINISHED yet")
	}

	// Pass 3: all children finished -> ParentFinish (StateEnd) -> FINISHED.
	repo.children = []domain.Workflow{
		{ID: 1001, Status: "FINISHED", BusinessKey: "child-1-of-42"},
		{ID: 1002, Status: "FINISHED", BusinessKey: "child-2-of-42"},
	}
	acts3 := runParentStep(t, repo, clock)
	if repo.state != workflows.ParentFinish {
		t.Fatalf("after pass 3 expected state=%q, got %q", workflows.ParentFinish, repo.state)
	}
	if repo.status != "FINISHED" {
		t.Fatalf("after pass 3 expected status=FINISHED, got %q", repo.status)
	}
	if !containsAction(acts3, "TRANSITION:ParentWaitForChildren") || !containsAction(acts3, "END:ParentFinish") {
		t.Errorf("pass 3 missing expected transitions/end, got: %v", acts3)
	}
}

func containsAction(actions []string, want string) bool {
	for _, a := range actions {
		if a == want {
			return true
		}
	}
	return false
}
