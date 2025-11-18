package postgres

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
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

func TestStartupAppAndWaitWorkflow(t *testing.T) {
	runTestWithSetup(t, func(t *testing.T, port int) {

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
		}
		app := gopherflow.SetupWithClock(workflowRegistry, clock)

		// Start the app in a goroutine so it doesn't block
		go func() {
			if err := app.Run(t.Context()); err != nil {
				slog.Error("Engine exited with error", "error", err)
			}
		}()

		//clock.Add(time.Duration(8) * time.Minute)

		url := fmt.Sprintf("http://localhost:%d/api/workflows", port)

		createReq := models.CreateWorkflowRequest{
			ExternalID:    "external-id-1",
			ExecutorGroup: "default",
			WorkflowType:  "WaitWorkflow",
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
		slog.Info("Waiting for workflow to complete")
		clock.Sleep(5 * time.Second)
		slog.Info("Waiting finished")

		common.GetWfAndExpectState(t, port, url, wf, req, err, client, "IN_PROGRESS")

		slog.Warn("advance clock by 15 minutes")
		clock.Add(time.Duration(15) * time.Minute)

		//the workflow should execute again now
		slog.Info("Waiting for workflow to complete")
		clock.Sleep(5 * time.Second)

		common.GetWfAndExpectState(t, port, url, wf, req, err, client, "FINISHED")

	})
}
