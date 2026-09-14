package repository

import (
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/RealZimboGuy/gopherflow/pkg/gopherflow/domain"
)

func TestWorkflowActionSaveAndFind(t *testing.T) {
	db, clock := newTestDB(t)
	wfRepo := NewWorkflowRepository(db, clock)
	repo := NewWorkflowActionRepository(db, clock)

	wfID, err := wfRepo.Save(newWorkflow("ext-action"))
	if err != nil {
		t.Fatalf("Save workflow: %v", err)
	}

	action := &domain.WorkflowAction{
		WorkflowID:     wfID,
		ExecutorID:     1,
		ExecutionCount: 1,
		RetryCount:     0,
		Type:           "STATE_CHANGE",
		Name:           "Init",
		Text:           "moved to Review",
		DateTime:       clock.Now(),
	}

	id, err := repo.Save(action)
	if err != nil {
		t.Fatalf("Save action: %v", err)
	}
	if id == 0 {
		t.Fatal("Save returned id 0")
	}

	got, err := repo.FindByID(id)
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if got == nil {
		t.Fatal("FindByID returned nil for a saved action")
	}
	if got.Type != "STATE_CHANGE" || got.Name != "Init" || got.WorkflowID != wfID {
		t.Errorf("got %+v, want the saved action", got)
	}
}

func TestWorkflowActionFindAllByWorkflowID(t *testing.T) {
	db, clock := newTestDB(t)
	wfRepo := NewWorkflowRepository(db, clock)
	repo := NewWorkflowActionRepository(db, clock)

	wfID, _ := wfRepo.Save(newWorkflow("ext-actions"))
	otherID, _ := wfRepo.Save(newWorkflow("ext-actions-other"))

	for i := 0; i < 3; i++ {
		_, err := repo.Save(&domain.WorkflowAction{
			WorkflowID: wfID, ExecutorID: 1, Type: "LOG",
			Name: "Init", Text: "entry", DateTime: clock.Now(),
		})
		if err != nil {
			t.Fatalf("Save action %d: %v", i, err)
		}
	}
	if _, err := repo.Save(&domain.WorkflowAction{
		WorkflowID: otherID, ExecutorID: 1, Type: "LOG",
		Name: "Init", Text: "other", DateTime: clock.Now(),
	}); err != nil {
		t.Fatalf("Save other action: %v", err)
	}

	got, err := repo.FindAllByWorkflowID(wfID)
	if err != nil {
		t.Fatalf("FindAllByWorkflowID: %v", err)
	}
	if got == nil {
		t.Fatal("FindAllByWorkflowID returned nil")
	}
	if len(*got) != 3 {
		t.Errorf("got %d actions, want 3 (must not include the other workflow's)", len(*got))
	}
}

func TestWorkflowActionFindByIDMissing(t *testing.T) {
	db, clock := newTestDB(t)
	repo := NewWorkflowActionRepository(db, clock)

	// Note the inconsistency with UserRepository.FindByUsername, which maps a
	// missing row to (nil, nil). This one surfaces sql.ErrNoRows to the caller.
	got, err := repo.FindByID(9999)
	if !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("err = %v, want sql.ErrNoRows", err)
	}
	if got != nil {
		t.Errorf("expected a nil action alongside the error, got %+v", got)
	}
}

func TestWorkflowDefinitionSaveIsUpsert(t *testing.T) {
	db, clock := newTestDB(t)
	repo := NewWorkflowDefinitionRepository(db, clock)

	def := &domain.WorkflowDefinition{
		Name:        "DemoWorkflow",
		Description: "first description",
		Created:     clock.Now(),
		Updated:     clock.Now(),
		FlowChart:   "flowchart TD",
	}
	if err := repo.Save(def); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Saving the same name again must update rather than insert a duplicate.
	def.Description = "second description"
	def.FlowChart = "flowchart TD\n  A --> B"
	if err := repo.Save(def); err != nil {
		t.Fatalf("Save (update): %v", err)
	}

	got, err := repo.FindByName("DemoWorkflow")
	if err != nil {
		t.Fatalf("FindByName: %v", err)
	}
	if got == nil {
		t.Fatal("FindByName returned nil")
	}
	if got.Description != "second description" {
		t.Errorf("description = %q, want the updated value", got.Description)
	}

	all, err := repo.FindAll()
	if err != nil {
		t.Fatalf("FindAll: %v", err)
	}
	if all == nil || len(*all) != 1 {
		t.Errorf("got %v definitions, want exactly 1 after the upsert", all)
	}
}

func TestWorkflowDefinitionFindByNameMissing(t *testing.T) {
	db, clock := newTestDB(t)
	repo := NewWorkflowDefinitionRepository(db, clock)

	got, err := repo.FindByName("NoSuchWorkflow")
	if err == nil && got == nil {
		return // nil, nil is an acceptable "not found"
	}
	if err != nil && got != nil {
		t.Errorf("got both a result %+v and an error %v", got, err)
	}
}

func TestWorkflowDefinitionFindAll(t *testing.T) {
	db, clock := newTestDB(t)
	repo := NewWorkflowDefinitionRepository(db, clock)

	for _, name := range []string{"A", "B"} {
		err := repo.Save(&domain.WorkflowDefinition{
			Name: name, Description: name, Created: clock.Now(),
			Updated: clock.Now(), FlowChart: "flowchart TD",
		})
		if err != nil {
			t.Fatalf("Save(%s): %v", name, err)
		}
	}

	all, err := repo.FindAll()
	if err != nil {
		t.Fatalf("FindAll: %v", err)
	}
	if all == nil || len(*all) != 2 {
		t.Errorf("got %v, want 2 definitions", all)
	}
}

func TestExecutorSaveAndList(t *testing.T) {
	db, clock := newTestDB(t)
	repo := NewExecutorRepository(db, clock)

	id, err := repo.Save(&domain.Executor{Name: "worker-1"})
	if err != nil {
		t.Fatalf("Save: %v", err)
	}
	if id == 0 {
		t.Fatal("Save returned id 0")
	}

	list, err := repo.GetExecutorsByLastActive(10)
	if err != nil {
		t.Fatalf("GetExecutorsByLastActive: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("got %d executors, want 1", len(list))
	}
	if list[0].Name != "worker-1" {
		t.Errorf("name = %q, want worker-1", list[0].Name)
	}
	if list[0].Started.IsZero() {
		t.Error("started timestamp was not defaulted on save")
	}
}

func TestExecutorSaveKeepsExplicitTimestamps(t *testing.T) {
	db, clock := newTestDB(t)
	repo := NewExecutorRepository(db, clock)

	started := clock.Now().Add(-2 * time.Hour).UTC()
	if _, err := repo.Save(&domain.Executor{Name: "worker-2", Started: started, LastActive: started}); err != nil {
		t.Fatalf("Save: %v", err)
	}

	list, err := repo.GetExecutorsByLastActive(10)
	if err != nil {
		t.Fatalf("GetExecutorsByLastActive: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("got %d executors, want 1", len(list))
	}
	if diff := list[0].Started.Sub(started); diff > time.Second || diff < -time.Second {
		t.Errorf("started = %v, want approximately %v", list[0].Started, started)
	}
}

func TestExecutorUpdateLastActive(t *testing.T) {
	db, clock := newTestDB(t)
	repo := NewExecutorRepository(db, clock)

	id, err := repo.Save(&domain.Executor{Name: "worker-3"})
	if err != nil {
		t.Fatalf("Save: %v", err)
	}

	later := clock.Now().Add(time.Hour).UTC()
	if err := repo.UpdateLastActive(id, later); err != nil {
		t.Fatalf("UpdateLastActive: %v", err)
	}

	list, err := repo.GetExecutorsByLastActive(10)
	if err != nil {
		t.Fatalf("GetExecutorsByLastActive: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("got %d executors, want 1", len(list))
	}
	if list[0].LastActive.Before(clock.Now()) {
		t.Errorf("last active = %v, want it advanced to about %v", list[0].LastActive, later)
	}
}

func TestExecutorListRespectsLimit(t *testing.T) {
	db, clock := newTestDB(t)
	repo := NewExecutorRepository(db, clock)

	for i := 0; i < 5; i++ {
		if _, err := repo.Save(&domain.Executor{Name: "worker"}); err != nil {
			t.Fatalf("Save %d: %v", i, err)
		}
	}

	list, err := repo.GetExecutorsByLastActive(3)
	if err != nil {
		t.Fatalf("GetExecutorsByLastActive: %v", err)
	}
	if len(list) != 3 {
		t.Errorf("got %d executors, want the limit of 3", len(list))
	}
}
