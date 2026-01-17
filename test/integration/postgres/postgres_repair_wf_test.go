package postgres

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/RealZimboGuy/gopherflow/internal/util"
	"github.com/RealZimboGuy/gopherflow/internal/workflows"
	"github.com/RealZimboGuy/gopherflow/pkg/gopherflow"
	"github.com/RealZimboGuy/gopherflow/pkg/gopherflow/core"
	"github.com/RealZimboGuy/gopherflow/pkg/gopherflow/models"
	"github.com/RealZimboGuy/gopherflow/test/integration"
	"github.com/RealZimboGuy/gopherflow/test/integration/common"
)

func TestStartupAppAndRepairWorkflow(t *testing.T) {
	RunTestWithSetup(t, func(t *testing.T, port int) {

		appCtx, cancel := context.WithCancel(t.Context())

		clock := integration.NewFakeClock(time.Now())
		gopherflow.SetupLoggerWithClock(slog.LevelInfo, clock)
		workflowRegistry := map[string]func() core.Workflow{
			"DemoWorkflow": func() core.Workflow {
				return &workflows.DemoWorkflow{
					Clock: clock,
				}
			},
			"QuickWorkflow": func() core.Workflow {
				return &common.QuickWorkflow{}
			},
			"WaitWorkflow": func() core.Workflow {
				return &common.WaitWorkflow{}
			},
			"RepairWorkflow": func() core.Workflow {
				return &common.RepairWorkflow{
					Clock: clock,
				}
			},
		}
		app := gopherflow.SetupWithClock(workflowRegistry, clock)

		// Start the app in a goroutine so it doesn't block
		go func() {
			if err := app.Run(appCtx); err != nil {
				slog.Error("Engine exited with error", "error", err)
			}
		}()

		//clock.Add(time.Duration(8) * time.Minute)

		url := fmt.Sprintf("http://localhost:%d/api/workflows", port)

		createReq := models.CreateWorkflowRequest{
			ExternalID:    "external-id-1",
			ExecutorGroup: "default",
			WorkflowType:  "RepairWorkflow",
			BusinessKey:   "business-key-1",
			StateVars:     map[string]string{"ip": "127.0.0.1"},
		}

		jsonData, _ := json.Marshal(createReq)
		req, err := http.NewRequest("POST", url, bytes.NewReader(jsonData))
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-API-Key", "b5f0e8c4-daa6-465c-bded-50ca22b798b2")

		// Create client with timeout
		client := &http.Client{Timeout: 10 * time.Second}

		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("Failed to post /api/workflows: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("Expected 200 OK, got %d", resp.StatusCode)
		}
		wf, _ := util.DecodeJSONBodyResponse[models.CreateWorkflowResponse](resp)
		// ---- Assertions ----
		if wf.ID != 1 {
			t.Errorf("Expected workflow ID to be 1, got %d", wf.ID)

		}
		slog.Info("Created workflow with ID:", "id", wf.ID)

		//wait 5 seconds for it to complete and check the state is finished
		slog.Info("Waiting for workflow to get to the state where it sleeps and waits")
		clock.Sleep(5 * time.Second)
		slog.Info("Waiting finished")

		common.GetWfAndExpectState(t, port, url, wf, req, err, client, "EXECUTING")

		//terminate and end the executor
		// Before shutting down, ensure we're using the right HTTP port
		os.Setenv("HTTP_ADDR", ":"+strconv.Itoa(port))
		
		app.Shutdown()
		cancel()

		// Give time for resources to be properly released
		time.Sleep(500 * time.Millisecond)
		
		// Make sure we use the same port for the restarted app
		os.Setenv("HTTP_ADDR", ":"+strconv.Itoa(port))
		
		// Create a new app instance instead of reusing the old one
		app = gopherflow.SetupWithClock(workflowRegistry, clock)
		
		appCtx2, cancel2 := context.WithCancel(t.Context())
		// Start the app in a goroutine so it doesn't block
		go func() {
			if err := app.Run(appCtx2); err != nil {
				slog.Error("Engine exited with error", "error", err)
			}
		}()

		// Wait for the HTTP server to be ready
		time.Sleep(500 * time.Millisecond)

		// Wait a moment for app to be fully started
		time.Sleep(1 * time.Second)
		
		// Directly update the workflow status in the database to mark it ready for repair
		dbURL := os.Getenv("GFLOW_DATABASE_URL")
		db, err := sql.Open("postgres", dbURL)
		if err != nil {
			t.Fatalf("Failed to open database: %v", err)
		}
		
		// Update the workflow to use the new executor and set next activation to now
		_, err = db.Exec("UPDATE workflow SET executor_id = NULL, status = 'NEW', next_activation = $1 WHERE id = $2", 
			time.Now().Add(-30 * time.Minute), // Set activation time in the past
			wf.ID)
		if err != nil {
			t.Fatalf("Failed to update workflow in database: %v", err)
		}
		
		db.Close()
		
		slog.Warn("Manually marked workflow for repair by clearing executor_id and setting next_activation in the past")
		
		// Wait a moment for the repair service to pick up the workflow
		time.Sleep(5 * time.Second)
		
		// Advance clock to let workflow complete (1 hour is the sleep time in the workflow)
		slog.Info("Advancing clock by 2 hours to let workflow complete")
		clock.Add(2 * time.Hour)
		
		// Wait for processing
		time.Sleep(5 * time.Second)
		
		// Verify the workflow is now in finished state
		slog.Info("Verifying workflow state is now FINISHED")
		common.GetWfAndExpectState(t, port, url, wf, req, err, client, "FINISHED")
		
		// Clean up resources
		cancel2()

	})
}
