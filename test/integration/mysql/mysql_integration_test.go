package mysql

import (
	"fmt"
	"log/slog"
	"net/http"
	"testing"
	"time"

	"github.com/RealZimboGuy/gopherflow/internal/util"
	"github.com/RealZimboGuy/gopherflow/internal/workflows"
	"github.com/RealZimboGuy/gopherflow/pkg/gopherflow"
	"github.com/RealZimboGuy/gopherflow/pkg/gopherflow/core"
	"github.com/RealZimboGuy/gopherflow/pkg/gopherflow/domain"
	"github.com/RealZimboGuy/gopherflow/test/integration"
)

func TestStartupAppAndGetExecutor(t *testing.T) {
	runTestWithSetup(t, func(t *testing.T, port int) {

		clock := integration.NewFakeClock(time.Now())
		gopherflow.SetupLoggerWithClock(clock)
		gopherflow.WorkflowRegistry = map[string]func() core.Workflow{
			"DemoWorkflow": func() core.Workflow {
				return &workflows.DemoWorkflow{
					Clock: clock,
				}
			},
			"GetIpWorkflow": func() core.Workflow {
				return &workflows.GetIpWorkflow{}
			},
		}
		app := gopherflow.SetupWithClock(clock)

		// Start the app in a goroutine so it doesn't block
		go func() {
			if err := app.Run(t.Context()); err != nil {
				slog.Error("Engine exited with error", "error", err)
			}
		}()

		clock.Add(time.Duration(8) * time.Minute)

		url := fmt.Sprintf("http://localhost:%d/api/executors", port)

		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-API-Key", "b5f0e8c4-daa6-465c-bded-50ca22b798b2")

		// Create client with timeout
		client := &http.Client{Timeout: 10 * time.Second}

		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("Failed to GET /api/executors: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("Expected 200 OK, got %d", resp.StatusCode)
		}
		executors, _ := util.DecodeJSONBodyResponse[[]domain.Executor](resp)
		// ---- Assertions ----
		if len(executors) != 1 {
			t.Errorf("Expected at least one executor, got none")
		} else {
			t.Logf("Got %d executors: %+v", len(executors), executors)
			//validate that the id, name, started, last active are set
			for _, e := range executors {
				if e.ID == 0 {
					t.Errorf("Executor ID is 0")
				}
				if e.Name == "" {
					t.Errorf("Executor name is empty")
				}
				if e.Started.IsZero() {
					t.Errorf("Executor started time is zero")
				}
			}
		}

	})

}
