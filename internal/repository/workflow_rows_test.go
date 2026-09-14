package repository

import (
	"database/sql"
	"testing"
	"time"
)

func TestWorkflowGetTopExecutingReturnsExecutingRows(t *testing.T) {
	repo, _ := newWorkflowRepo(t)

	wf := newWorkflow("ext-executing")
	if _, err := repo.Save(wf); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if err := repo.UpdateWorkflowStatus(wf.ID, "EXECUTING"); err != nil {
		t.Fatalf("UpdateWorkflowStatus: %v", err)
	}

	got, err := repo.GetTopExecuting(10)
	if err != nil {
		t.Fatalf("GetTopExecuting: %v", err)
	}
	if got == nil || len(*got) != 1 {
		t.Fatalf("got %v, want exactly one executing workflow", got)
	}
	if (*got)[0].ExternalID != "ext-executing" {
		t.Errorf("got %q, want ext-executing", (*got)[0].ExternalID)
	}
}

func TestWorkflowMarkAsScheduledSucceedsWithTheCurrentModified(t *testing.T) {
	repo, _ := newWorkflowRepo(t)

	id, err := repo.Save(newWorkflow("ext-mark"))
	if err != nil {
		t.Fatalf("Save: %v", err)
	}
	saved, err := repo.FindByID(id)
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}

	if !repo.MarkWorkflowAsScheduledForExecution(id, 7, saved.Modified) {
		t.Fatal("failed to mark a fresh NEW workflow as scheduled")
	}

	after, err := repo.FindByID(id)
	if err != nil {
		t.Fatalf("FindByID after mark: %v", err)
	}
	if after.Status != "SCHEDULED" {
		t.Errorf("status = %q, want SCHEDULED", after.Status)
	}
	if !after.ExecutorID.Valid {
		t.Error("executor id was not set")
	}

	// A second attempt with the now-stale modified value must not win the row.
	if repo.MarkWorkflowAsScheduledForExecution(id, 8, saved.Modified) {
		t.Error("a second executor acquired the same workflow")
	}
}

func TestWorkflowLockByModifiedSucceedsWithTheCurrentModified(t *testing.T) {
	repo, _ := newWorkflowRepo(t)

	id, err := repo.Save(newWorkflow("ext-lock-ok"))
	if err != nil {
		t.Fatalf("Save: %v", err)
	}
	saved, err := repo.FindByID(id)
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}

	if !repo.LockWorkflowByModified(id, saved.Modified) {
		t.Error("failed to lock with the current modified timestamp")
	}
}

func TestWorkflowGetChildrenOnlyActive(t *testing.T) {
	repo, _ := newWorkflowRepo(t)

	parentID, err := repo.Save(newWorkflow("ext-parent-active"))
	if err != nil {
		t.Fatalf("Save parent: %v", err)
	}

	active := newWorkflow("ext-child-active")
	active.ParentWorkflowID = sql.NullInt64{Int64: parentID, Valid: true}
	if _, err := repo.Save(active); err != nil {
		t.Fatalf("Save active child: %v", err)
	}

	done := newWorkflow("ext-child-done")
	done.ParentWorkflowID = sql.NullInt64{Int64: parentID, Valid: true}
	if _, err := repo.Save(done); err != nil {
		t.Fatalf("Save finished child: %v", err)
	}
	if err := repo.UpdateWorkflowStatus(done.ID, "FINISHED"); err != nil {
		t.Fatalf("UpdateWorkflowStatus: %v", err)
	}

	all, err := repo.GetChildrenByParentID(parentID, false)
	if err != nil {
		t.Fatalf("GetChildrenByParentID(all): %v", err)
	}
	if all == nil || len(*all) != 2 {
		t.Fatalf("got %v, want both children", all)
	}

	onlyActive, err := repo.GetChildrenByParentID(parentID, true)
	if err != nil {
		t.Fatalf("GetChildrenByParentID(active): %v", err)
	}
	if onlyActive == nil {
		t.Fatal("GetChildrenByParentID(active) returned nil")
	}
	for _, c := range *onlyActive {
		if c.Status == "FINISHED" {
			t.Errorf("a finished child was returned as active: %+v", c)
		}
	}
}

func TestWorkflowFindStuckWorkflowsReturnsAnOldExecutingRow(t *testing.T) {
	repo, clock := newWorkflowRepo(t)

	wf := newWorkflow("ext-really-stuck")
	if _, err := repo.Save(wf); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if err := repo.UpdateWorkflowStatus(wf.ID, "EXECUTING"); err != nil {
		t.Fatalf("UpdateWorkflowStatus: %v", err)
	}

	// Move the clock forward so the row looks old to the repair query.
	clock.Advance(2 * time.Hour)

	got, err := repo.FindStuckWorkflows("1", "default", 10)
	if err != nil {
		t.Fatalf("FindStuckWorkflows: %v", err)
	}
	if got == nil {
		t.Fatal("FindStuckWorkflows returned nil")
	}
}

func TestWorkflowUpdateNextActivationOffsetVariants(t *testing.T) {
	repo, _ := newWorkflowRepo(t)

	id, err := repo.Save(newWorkflow("ext-offsets"))
	if err != nil {
		t.Fatalf("Save: %v", err)
	}

	for _, offset := range []string{"30 minutes", "2 hours", "45 seconds", "not-an-interval", ""} {
		t.Run(offset, func(t *testing.T) {
			if err := repo.UpdateNextActivationOffset(id, offset); err != nil {
				t.Fatalf("UpdateNextActivationOffset(%q): %v", offset, err)
			}
			got, err := repo.FindByID(id)
			if err != nil {
				t.Fatalf("FindByID: %v", err)
			}
			if !got.NextActivation.Valid {
				t.Errorf("next activation is unset after offset %q", offset)
			}
		})
	}
}
