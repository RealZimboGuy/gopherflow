package sqllite

import (
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/RealZimboGuy/gopherflow/internal/repository"
	"github.com/RealZimboGuy/gopherflow/pkg/gopherflow/domain"
	"github.com/RealZimboGuy/gopherflow/test/integration"
)

func TestParentChildWorkflowRepository(t *testing.T) {
	runTestWithSetup(t, func(t *testing.T, port int) {
		// Initialize fake clock
		clock := integration.NewFakeClock(time.Now())
		
		// Open database connection directly - we'll use the environment variables
		// that have been set by runTestWithSetup to ensure the file path is correct
		dbName := os.Getenv("GFLOW_DATABASE_SQLLITE_FILE_NAME")
		db, err := sql.Open("sqlite3", dbName)
		if err != nil {
			t.Fatalf("Failed to open database: %v", err)
		}
		defer db.Close()
		
		// Execute migrations directly since we're not using the full app setup
		// This ensures the tables exist before running the tests
		_, err = db.Exec(`
			-- Initial schema for workflow table
			CREATE TABLE IF NOT EXISTS workflow (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				status TEXT,
				execution_count INT,
				retry_count INT,
				created TIMESTAMP,
				modified TIMESTAMP,
				next_activation TIMESTAMP,
				started TIMESTAMP,
				executor_id INT,
				executor_group TEXT,
				workflow_type TEXT,
				external_id TEXT,
				business_key TEXT,
				state TEXT,
				state_vars TEXT,
				parent_workflow_id INTEGER NULL REFERENCES workflow(id)
			);
		`)
		if err != nil {
			t.Fatalf("Failed to create workflow table: %v", err)
		}
		
		// Create workflow repository
		wfRepo := repository.NewWorkflowRepository(db, clock)
		
		// Test CreateChildWorkflow
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
				State:          "ParentInit",
				StateVars:      sql.NullString{String: "{}", Valid: true},
			}
			
			// Save the parent workflow
			parentID, err := wfRepo.Save(parentWf)
			if err != nil {
				t.Fatalf("Failed to save parent workflow: %v", err)
			}
			
			// Create a child workflow
			childWf, err := wfRepo.CreateChildWorkflow(
				parentID,
				"ChildWorkflow",
				"ChildInit",
				"child-1",
				`{"input":"test-value"}`,
				"default",
				"",
			)
			
			if err != nil {
				t.Fatalf("Failed to create child workflow: %v", err)
			}
			
			// Verify child workflow was created properly
			if childWf == nil {
				t.Fatal("Child workflow is nil")
			}
			
			if childWf.ParentWorkflowID.Valid == false || childWf.ParentWorkflowID.Int64 != parentID {
				t.Errorf("Expected ParentWorkflowID to be %d, got %v", parentID, childWf.ParentWorkflowID)
			}
			
			if childWf.WorkflowType != "ChildWorkflow" {
				t.Errorf("Expected WorkflowType to be ChildWorkflow, got %s", childWf.WorkflowType)
			}
			
			if childWf.State != "ChildInit" {
				t.Errorf("Expected State to be ChildInit, got %s", childWf.State)
			}
			
			if childWf.BusinessKey != "child-1" {
				t.Errorf("Expected BusinessKey to be child-1, got %s", childWf.BusinessKey)
			}
		})
		
		// Test GetChildrenByParentID
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
				State:          "ParentInit",
				StateVars:      sql.NullString{String: "{}", Valid: true},
			}
			
			// Save the parent workflow
			parentID, err := wfRepo.Save(parentWf)
			if err != nil {
				t.Fatalf("Failed to save parent workflow: %v", err)
			}
			
			// Create child workflows
			for i := 1; i <= 3; i++ {
				_, err := wfRepo.CreateChildWorkflow(
					parentID,
					"ChildWorkflow",
					"ChildInit",
					fmt.Sprintf("child-%d", i),
					fmt.Sprintf(`{"input":"value%d"}`, i),
					"default",
					"",
				)
				
				if err != nil {
					t.Fatalf("Failed to create child workflow %d: %v", i, err)
				}
			}
			
			// Get all children
			children, err := wfRepo.GetChildrenByParentID(parentID, false)
			if err != nil {
				t.Fatalf("Failed to get child workflows: %v", err)
			}
			
			// Verify we got all children
			if len(*children) != 3 {
				t.Errorf("Expected 3 child workflows, got %d", len(*children))
			}
			
			// Get only active children
			activeChildren, err := wfRepo.GetChildrenByParentID(parentID, true)
			if err != nil {
				t.Fatalf("Failed to get active child workflows: %v", err)
			}
			
			// Verify we got all active children (all should be active)
			if len(*activeChildren) != 3 {
				t.Errorf("Expected 3 active child workflows, got %d", len(*activeChildren))
			}
			
			// Set one child to FINISHED status
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
			
			// Verify we now have one less active child
			if len(*activeChildren) != 2 {
				t.Errorf("Expected 2 active child workflows, got %d", len(*activeChildren))
			}
		})
		
		// Test WakeParentWorkflow
		t.Run("WakeParentWorkflow", func(t *testing.T) {
			// Create a parent workflow with next_activation set to far future
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
				State:          "ParentWaitForChildren",
				StateVars:      sql.NullString{String: "{}", Valid: true},
			}
			
			// Save the parent workflow
			parentID, err := wfRepo.Save(parentWf)
			if err != nil {
				t.Fatalf("Failed to save parent workflow: %v", err)
			}
			
			// Create a child workflow but don't store the result since we don't need it
			_, err = wfRepo.CreateChildWorkflow(
				parentID,
				"ChildWorkflow",
				"ChildInit",
				"child-wake-test",
				`{"input":"wake-parent"}`,
				"default",
				"",
			)
		
			if err != nil {
				t.Fatalf("Failed to create child workflow: %v", err)
			}
		
			// Get the parent workflow to check its current next_activation
			parentBefore, err := wfRepo.FindByID(parentID)
			if err != nil {
				t.Fatalf("Failed to get parent workflow: %v", err)
			}
		
			// Verify next_activation is set to future
			if !parentBefore.NextActivation.Valid || !parentBefore.NextActivation.Time.After(clock.Now()) {
				t.Errorf("Expected parent workflow next_activation to be in the future")
			}
		
			// Wake the parent
			err = wfRepo.WakeParentWorkflow(parentID)
			if err != nil {
				t.Fatalf("Failed to wake parent workflow: %v", err)
			}
		
			// Get the parent workflow again
			parentAfter, err := wfRepo.FindByID(parentID)
			if err != nil {
				t.Fatalf("Failed to get parent workflow after wake: %v", err)
			}
		
			// Verify next_activation is updated to now
			if !parentAfter.NextActivation.Valid {
				t.Errorf("Expected parent workflow next_activation to be valid")
			}
		
			// Should be very close to now (within a second)
			timeDiff := parentAfter.NextActivation.Time.Sub(clock.Now())
			if timeDiff < -1*time.Second || timeDiff > 1*time.Second {
				t.Errorf("Expected parent workflow next_activation to be close to now, got diff: %v", timeDiff)
			}
		})
	})
}

// Note: This test has been commented out as it requires additional infrastructure setup
// func TestParentChildWorkflowExecution(t *testing.T) {
//     // This test requires a full application integration test with API access.
//     // For now, we're testing the repository methods only in TestParentChildWorkflowRepository.
// }
