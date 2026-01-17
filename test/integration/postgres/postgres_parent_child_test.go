package postgres

import (
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/RealZimboGuy/gopherflow/internal/repository"
	"github.com/RealZimboGuy/gopherflow/pkg/gopherflow/domain"
	"github.com/RealZimboGuy/gopherflow/test/integration"
	_ "github.com/lib/pq"
)

// TestParentChildRepositoryMethods tests the repository methods for parent-child workflow functionality
func TestParentChildRepositoryMethods(t *testing.T) {
	RunTestWithSetup(t, func(t *testing.T, port int) {
		// Create a fake clock for testing
		clock := integration.NewFakeClock(time.Now())
		
		// Get the database connection string
		dsn := os.Getenv("GFLOW_DATABASE_URL")
		
		// Connect to the database
		db, err := sql.Open("postgres", dsn)
		if err != nil {
			t.Fatalf("Failed to connect to database: %v", err)
		}
		defer db.Close()
		
		// Create the workflow repository
		wfRepo := repository.NewWorkflowRepository(db, clock)
		
		// Test creating a child workflow
		t.Run("CreateChildWorkflow", func(t *testing.T) {
			// Create a parent workflow first
			parentWf := &domain.Workflow{
				Status:         "NEW",
				ExecutionCount: 0,
				RetryCount:     0,
				Created:        clock.Now(),
				Modified:       clock.Now(),
				NextActivation: sql.NullTime{Time: clock.Now(), Valid: true},
				ExecutorGroup:  "DEFAULT",
				WorkflowType:   "ParentWorkflow",
				BusinessKey:    "parent-1",
				State:          "Init",
				StateVars:      sql.NullString{String: "{}", Valid: true},
			}
			
			// Save the parent workflow
			parentID, err := wfRepo.Save(parentWf)
			if err != nil {
				t.Fatalf("Failed to save parent workflow: %v", err)
			}
			
			// Create a child workflow directly using Save
			childWf := &domain.Workflow{
				Status:           "NEW",
				ExecutionCount:   0,
				RetryCount:       0,
				Created:          clock.Now(),
				Modified:         clock.Now(),
				NextActivation:   sql.NullTime{Time: clock.Now(), Valid: true},
				ExecutorGroup:    "default",
				WorkflowType:     "ChildWorkflow",
				BusinessKey:      "child-1",
				State:            "Init",
				ExternalID:       `{"input":"test-value"}`,
				StateVars:        sql.NullString{String: "", Valid: false},
				ParentWorkflowID: sql.NullInt64{Int64: parentID, Valid: true},
			}
			
			childID, err := wfRepo.Save(childWf)
			if err != nil {
				t.Fatalf("Failed to save child workflow: %v", err)
			}
			
			// Get the saved child workflow
			childWf, err = wfRepo.FindByID(childID)
			
			if err != nil {
				t.Fatalf("Failed to create child workflow: %v", err)
			}
			
			// Verify the child workflow
			if childWf == nil {
				t.Fatal("Child workflow is nil")
			}
			
			if childWf.ParentWorkflowID.Valid == false {
				t.Error("Expected ParentWorkflowID to be valid")
			}
			
			if childWf.ParentWorkflowID.Int64 != parentID {
				t.Errorf("Expected ParentWorkflowID to be %d, got %d", parentID, childWf.ParentWorkflowID.Int64)
			}
			
			if childWf.WorkflowType != "ChildWorkflow" {
				t.Errorf("Expected WorkflowType to be ChildWorkflow, got %s", childWf.WorkflowType)
			}
			
			if childWf.State != "Init" {
				t.Errorf("Expected State to be Init, got %s", childWf.State)
			}
			
			if childWf.BusinessKey != "child-1" {
				t.Errorf("Expected BusinessKey to be child-1, got %s", childWf.BusinessKey)
			}
		})
		
		// Test getting children by parent ID
		t.Run("GetChildrenByParentID", func(t *testing.T) {
			// Create a parent workflow
			parentWf := &domain.Workflow{
				Status:         "NEW",
				ExecutionCount: 0,
				RetryCount:     0,
				Created:        clock.Now(),
				Modified:       clock.Now(),
				NextActivation: sql.NullTime{Time: clock.Now(), Valid: true},
				ExecutorGroup:  "DEFAULT",
				WorkflowType:   "ParentWorkflow",
				BusinessKey:    "parent-2",
				State:          "Init",
				StateVars:      sql.NullString{String: "{}", Valid: true},
			}
			
			// Save the parent workflow
			parentID, err := wfRepo.Save(parentWf)
			if err != nil {
				t.Fatalf("Failed to save parent workflow: %v", err)
			}
			
			// Create multiple child workflows
			childCount := 3
			for i := 1; i <= childCount; i++ {
				childWf := &domain.Workflow{
					Status:           "NEW",
					ExecutionCount:   0,
					RetryCount:       0,
					Created:          clock.Now(),
					Modified:         clock.Now(),
					NextActivation:   sql.NullTime{Time: clock.Now(), Valid: true},
					ExecutorGroup:    "default",
					WorkflowType:     "ChildWorkflow",
					BusinessKey:      fmt.Sprintf("child-%d", i),
					State:            "Init",
					ExternalID:       fmt.Sprintf(`{"index":%d}`, i),
					StateVars:        sql.NullString{String: "", Valid: false},
					ParentWorkflowID: sql.NullInt64{Int64: parentID, Valid: true},
				}
				
				_, err := wfRepo.Save(childWf)
				
				if err != nil {
					t.Fatalf("Failed to create child workflow %d: %v", i, err)
				}
			}
			
			// Get all child workflows
			children, err := wfRepo.GetChildrenByParentID(parentID, false)
			if err != nil {
				t.Fatalf("Failed to get child workflows: %v", err)
			}
			
			// Verify we got the expected number of children
			if len(*children) != childCount {
				t.Errorf("Expected %d child workflows, got %d", childCount, len(*children))
			}
			
			// Get only active children (all should be active)
			activeChildren, err := wfRepo.GetChildrenByParentID(parentID, true)
			if err != nil {
				t.Fatalf("Failed to get active child workflows: %v", err)
			}
			
			// Verify all children are active
			if len(*activeChildren) != childCount {
				t.Errorf("Expected %d active child workflows, got %d", childCount, len(*activeChildren))
			}
			
			// Set one child to FINISHED
			firstChild := (*children)[0]
			err = wfRepo.UpdateWorkflowStatus(firstChild.ID, "FINISHED")
			if err != nil {
				t.Fatalf("Failed to update child workflow status: %v", err)
			}
			
			// Get active children again
			activeChildren, err = wfRepo.GetChildrenByParentID(parentID, true)
			if err != nil {
				t.Fatalf("Failed to get active child workflows: %v", err)
			}
			
			// Verify we have one fewer active child
			if len(*activeChildren) != childCount-1 {
				t.Errorf("Expected %d active child workflows, got %d", childCount-1, len(*activeChildren))
			}
		})
		
		// Test waking parent workflow
		t.Run("WakeParentWorkflow", func(t *testing.T) {
			// Create a parent workflow with next_activation in the future
			futureTime := clock.Now().Add(24 * time.Hour)
			parentWf := &domain.Workflow{
				Status:         "IN_PROGRESS",
				ExecutionCount: 1,
				RetryCount:     0,
				Created:        clock.Now(),
				Modified:       clock.Now(),
				NextActivation: sql.NullTime{Time: futureTime, Valid: true},
				ExecutorGroup:  "DEFAULT",
				WorkflowType:   "ParentWorkflow",
				BusinessKey:    "parent-3",
				State:          "WaitForChildren",
				StateVars:      sql.NullString{String: "{}", Valid: true},
			}
			
			// Save the parent workflow
			parentID, err := wfRepo.Save(parentWf)
			if err != nil {
				t.Fatalf("Failed to save parent workflow: %v", err)
			}
			
			// Create a child workflow
			childWf := &domain.Workflow{
				Status:           "NEW",
				ExecutionCount:   0,
				RetryCount:       0,
				Created:          clock.Now(),
				Modified:         clock.Now(),
				NextActivation:   sql.NullTime{Time: clock.Now(), Valid: true},
				ExecutorGroup:    "default",
				WorkflowType:     "ChildWorkflow",
				BusinessKey:      "child-wake-test",
				State:            "Init",
				ExternalID:       `{"task":"wake-parent"}`,
				StateVars:        sql.NullString{String: "", Valid: false},
				ParentWorkflowID: sql.NullInt64{Int64: parentID, Valid: true},
			}
			
			_, err = wfRepo.Save(childWf)
			
			if err != nil {
				t.Fatalf("Failed to create child workflow: %v", err)
			}
			
			// Verify parent's next_activation is in the future
			parentBefore, err := wfRepo.FindByID(parentID)
			if err != nil {
				t.Fatalf("Failed to get parent workflow: %v", err)
			}
			
			if !parentBefore.NextActivation.Valid || !parentBefore.NextActivation.Time.After(clock.Now()) {
				t.Error("Expected parent workflow next_activation to be in the future")
			}
			
			// Wake the parent
			err = wfRepo.WakeParentWorkflow(parentID)
			if err != nil {
				t.Fatalf("Failed to wake parent workflow: %v", err)
			}
			
			// Get the parent workflow after waking
			parentAfter, err := wfRepo.FindByID(parentID)
			if err != nil {
				t.Fatalf("Failed to get parent workflow after wake: %v", err)
			}
			
			// Verify next_activation is updated to now
			if !parentAfter.NextActivation.Valid {
				t.Error("Expected parent workflow next_activation to be valid")
			}
			
			// Check that next_activation is close to now (within a second)
			timeDiff := parentAfter.NextActivation.Time.Sub(clock.Now())
			if timeDiff < -1*time.Second || timeDiff > 1*time.Second {
				t.Errorf("Expected parent workflow next_activation to be close to now, got diff: %v", timeDiff)
			}
		})
	})
}
