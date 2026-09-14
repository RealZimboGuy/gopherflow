package repository

import (
	"database/sql"
	"testing"
	"time"

	"github.com/RealZimboGuy/gopherflow/pkg/gopherflow/models"
)

func TestWorkflowSaveAndFindByID(t *testing.T) {
	repo, _ := newWorkflowRepo(t)

	wf := newWorkflow("ext-1")
	id, err := repo.Save(wf)
	if err != nil {
		t.Fatalf("Save: %v", err)
	}
	if id == 0 {
		t.Fatal("Save returned id 0")
	}

	got, err := repo.FindByID(id)
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if got == nil {
		t.Fatal("FindByID returned nil for a saved workflow")
	}
	if got.ExternalID != "ext-1" || got.WorkflowType != "DemoWorkflow" || got.State != "Init" {
		t.Errorf("got %+v, want the saved values", got)
	}
}

func TestWorkflowFindByExternalId(t *testing.T) {
	repo, _ := newWorkflowRepo(t)

	if _, err := repo.Save(newWorkflow("ext-lookup")); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := repo.FindByExternalId("ext-lookup")
	if err != nil {
		t.Fatalf("FindByExternalId: %v", err)
	}
	if got == nil || got.ExternalID != "ext-lookup" {
		t.Fatalf("got %+v, want the workflow with external id ext-lookup", got)
	}
}

func TestWorkflowUpdateStateAndStatus(t *testing.T) {
	repo, _ := newWorkflowRepo(t)

	id, err := repo.Save(newWorkflow("ext-state"))
	if err != nil {
		t.Fatalf("Save: %v", err)
	}

	if err := repo.UpdateState(id, "Review"); err != nil {
		t.Fatalf("UpdateState: %v", err)
	}
	if err := repo.UpdateWorkflowStatus(id, "IN_PROGRESS"); err != nil {
		t.Fatalf("UpdateWorkflowStatus: %v", err)
	}

	got, err := repo.FindByID(id)
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if got.State != "Review" {
		t.Errorf("state = %q, want %q", got.State, "Review")
	}
	if got.Status != "IN_PROGRESS" {
		t.Errorf("status = %q, want %q", got.Status, "IN_PROGRESS")
	}
}

func TestWorkflowUpdateStartingTime(t *testing.T) {
	repo, _ := newWorkflowRepo(t)

	id, _ := repo.Save(newWorkflow("ext-start"))
	if err := repo.UpdateWorkflowStartingTime(id); err != nil {
		t.Fatalf("UpdateWorkflowStartingTime: %v", err)
	}

	got, _ := repo.FindByID(id)
	if !got.Started.Valid {
		t.Error("started timestamp was not set")
	}
}

func TestWorkflowSaveVariables(t *testing.T) {
	repo, _ := newWorkflowRepo(t)

	id, _ := repo.Save(newWorkflow("ext-vars"))

	if err := repo.SaveWorkflowVariables(id, `{"a":"1"}`); err != nil {
		t.Fatalf("SaveWorkflowVariables: %v", err)
	}
	got, _ := repo.FindByID(id)
	if got.StateVars.String != `{"a":"1"}` {
		t.Errorf("state vars = %q, want %q", got.StateVars.String, `{"a":"1"}`)
	}

	if err := repo.SaveWorkflowVariablesAndTouch(id, `{"b":"2"}`); err != nil {
		t.Fatalf("SaveWorkflowVariablesAndTouch: %v", err)
	}
	got, _ = repo.FindByID(id)
	if got.StateVars.String != `{"b":"2"}` {
		t.Errorf("state vars = %q, want %q", got.StateVars.String, `{"b":"2"}`)
	}
}

func TestWorkflowNextActivation(t *testing.T) {
	repo, clock := newWorkflowRepo(t)

	id, _ := repo.Save(newWorkflow("ext-next"))

	target := clock.Now().Add(2 * time.Hour).UTC()
	if err := repo.UpdateNextActivationSpecific(id, target); err != nil {
		t.Fatalf("UpdateNextActivationSpecific: %v", err)
	}
	got, _ := repo.FindByID(id)
	if !got.NextActivation.Valid {
		t.Fatal("next activation was not set")
	}

	if err := repo.UpdateNextActivationOffset(id, "30 minutes"); err != nil {
		t.Fatalf("UpdateNextActivationOffset: %v", err)
	}
	got, _ = repo.FindByID(id)
	if !got.NextActivation.Valid {
		t.Error("next activation was cleared by the offset update")
	}
}

func TestWorkflowRetryCounter(t *testing.T) {
	repo, clock := newWorkflowRepo(t)

	id, _ := repo.Save(newWorkflow("ext-retry"))

	before, _ := repo.FindByID(id)
	if err := repo.IncrementRetryCounterAndSetNextActivation(id, clock.Now().Add(time.Minute)); err != nil {
		t.Fatalf("IncrementRetryCounterAndSetNextActivation: %v", err)
	}

	after, _ := repo.FindByID(id)
	if after.RetryCount != before.RetryCount+1 {
		t.Errorf("retry count = %d, want %d", after.RetryCount, before.RetryCount+1)
	}
}

func TestWorkflowClearExecutorId(t *testing.T) {
	repo, _ := newWorkflowRepo(t)

	id, _ := repo.Save(newWorkflow("ext-clear"))
	if ok := repo.MarkWorkflowAsScheduledForExecution(id, 42, time.Time{}); ok {
		// Locking may legitimately fail on a modified mismatch; either way the
		// clear below must succeed.
		t.Log("workflow was locked")
	}

	if err := repo.ClearExecutorId(id); err != nil {
		t.Fatalf("ClearExecutorId: %v", err)
	}
	got, _ := repo.FindByID(id)
	if got.ExecutorID.Valid {
		t.Errorf("executor id is still set: %v", got.ExecutorID)
	}
}

func TestWorkflowFindPendingWorkflows(t *testing.T) {
	repo, clock := newWorkflowRepo(t)

	wf := newWorkflow("ext-pending")
	wf.NextActivation = sql.NullTime{Time: clock.Now().Add(-time.Hour), Valid: true}
	if _, err := repo.Save(wf); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := repo.FindPendingWorkflows(10, "default")
	if err != nil {
		t.Fatalf("FindPendingWorkflows: %v", err)
	}
	if got == nil {
		t.Fatal("FindPendingWorkflows returned nil")
	}
	if len(*got) == 0 {
		t.Error("expected the due workflow to be returned")
	}
}

func TestWorkflowFindPendingIgnoresOtherExecutorGroup(t *testing.T) {
	repo, clock := newWorkflowRepo(t)

	wf := newWorkflow("ext-other-group")
	wf.ExecutorGroup = "other"
	wf.NextActivation = sql.NullTime{Time: clock.Now().Add(-time.Hour), Valid: true}
	if _, err := repo.Save(wf); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := repo.FindPendingWorkflows(10, "default")
	if err != nil {
		t.Fatalf("FindPendingWorkflows: %v", err)
	}
	if got != nil {
		for _, w := range *got {
			if w.ExternalID == "ext-other-group" {
				t.Error("a workflow from another executor group was returned")
			}
		}
	}
}

func TestWorkflowSearchWorkflows(t *testing.T) {
	repo, _ := newWorkflowRepo(t)

	if _, err := repo.Save(newWorkflow("ext-search-1")); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if _, err := repo.Save(newWorkflow("ext-search-2")); err != nil {
		t.Fatalf("Save: %v", err)
	}

	t.Run("by external id", func(t *testing.T) {
		got, err := repo.SearchWorkflows(models.SearchWorkflowRequest{ExternalID: "ext-search-1", Limit: 10})
		if err != nil {
			t.Fatalf("SearchWorkflows: %v", err)
		}
		if got == nil || len(*got) != 1 {
			t.Fatalf("got %v, want exactly one match", got)
		}
		if (*got)[0].ExternalID != "ext-search-1" {
			t.Errorf("got %q, want ext-search-1", (*got)[0].ExternalID)
		}
	})

	t.Run("by workflow type returns both", func(t *testing.T) {
		got, err := repo.SearchWorkflows(models.SearchWorkflowRequest{WorkflowType: "DemoWorkflow", Limit: 10})
		if err != nil {
			t.Fatalf("SearchWorkflows: %v", err)
		}
		if got == nil || len(*got) < 2 {
			t.Errorf("got %v, want at least two matches", got)
		}
	})

	t.Run("no filters returns everything", func(t *testing.T) {
		got, err := repo.SearchWorkflows(models.SearchWorkflowRequest{Limit: 10})
		if err != nil {
			t.Fatalf("SearchWorkflows: %v", err)
		}
		if got == nil || len(*got) < 2 {
			t.Errorf("got %v, want all workflows", got)
		}
	})
}

func TestWorkflowOverviewQueries(t *testing.T) {
	repo, _ := newWorkflowRepo(t)

	if _, err := repo.Save(newWorkflow("ext-overview")); err != nil {
		t.Fatalf("Save: %v", err)
	}

	rows, err := repo.GetWorkflowOverview()
	if err != nil {
		t.Fatalf("GetWorkflowOverview: %v", err)
	}
	if len(rows) == 0 {
		t.Error("expected at least one overview row")
	}

	stateRows, err := repo.GetDefinitionStateOverview("DemoWorkflow")
	if err != nil {
		t.Fatalf("GetDefinitionStateOverview: %v", err)
	}
	if len(stateRows) == 0 {
		t.Error("expected at least one definition state row")
	}
}

func TestWorkflowTopExecutingAndNextToExecute(t *testing.T) {
	repo, clock := newWorkflowRepo(t)

	wf := newWorkflow("ext-top")
	wf.NextActivation = sql.NullTime{Time: clock.Now().Add(time.Hour), Valid: true}
	if _, err := repo.Save(wf); err != nil {
		t.Fatalf("Save: %v", err)
	}

	if _, err := repo.GetTopExecuting(5); err != nil {
		t.Fatalf("GetTopExecuting: %v", err)
	}

	next, err := repo.GetNextToExecute(5)
	if err != nil {
		t.Fatalf("GetNextToExecute: %v", err)
	}
	if next == nil {
		t.Error("GetNextToExecute returned nil")
	}
}

func TestWorkflowFindStuckWorkflows(t *testing.T) {
	repo, _ := newWorkflowRepo(t)

	wf := newWorkflow("ext-stuck")
	wf.Status = "EXECUTING"
	if _, err := repo.Save(wf); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Nothing should be stuck when the repair window is far in the past.
	got, err := repo.FindStuckWorkflows("600", "default", 10)
	if err != nil {
		t.Fatalf("FindStuckWorkflows: %v", err)
	}
	if got == nil {
		t.Fatal("FindStuckWorkflows returned nil")
	}
}

func TestWorkflowParentChild(t *testing.T) {
	repo, _ := newWorkflowRepo(t)

	parentID, err := repo.Save(newWorkflow("ext-parent"))
	if err != nil {
		t.Fatalf("Save parent: %v", err)
	}

	child := newWorkflow("ext-child")
	child.ParentWorkflowID = sql.NullInt64{Int64: parentID, Valid: true}
	if _, err := repo.Save(child); err != nil {
		t.Fatalf("Save child: %v", err)
	}

	children, err := repo.GetChildrenByParentID(parentID, false)
	if err != nil {
		t.Fatalf("GetChildrenByParentID: %v", err)
	}
	if children == nil || len(*children) != 1 {
		t.Fatalf("got %v, want one child", children)
	}
	if (*children)[0].ExternalID != "ext-child" {
		t.Errorf("got %q, want ext-child", (*children)[0].ExternalID)
	}

	if err := repo.WakeParentWorkflow(parentID); err != nil {
		t.Fatalf("WakeParentWorkflow: %v", err)
	}
}

func TestWorkflowLockWorkflowByModified(t *testing.T) {
	repo, _ := newWorkflowRepo(t)

	id, _ := repo.Save(newWorkflow("ext-lock"))
	saved, _ := repo.FindByID(id)

	// A stale modified timestamp must not win the lock.
	if repo.LockWorkflowByModified(id, saved.Modified.Add(-time.Hour)) {
		t.Error("lock was granted with a stale modified timestamp")
	}
}
