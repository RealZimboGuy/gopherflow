package common

import (
	"fmt"
	"log/slog"
	"net/http"
	"testing"

	"github.com/RealZimboGuy/gopherflow/internal/util"
	"github.com/RealZimboGuy/gopherflow/pkg/gopherflow/models"
)

func GetWfAndExpectState(t *testing.T, port int, url string, wf models.CreateWorkflowResponse, req *http.Request, err error, client *http.Client, state string) {
	url = fmt.Sprintf("http://localhost:%d/api/workflows/%d", port, wf.ID)

	req, err = http.NewRequest("GET", url, nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", "b5f0e8c4-daa6-465c-bded-50ca22b798b2")

	// Create client with timeout

	resp2, err := client.Do(req)
	if err != nil {
		t.Fatalf("Failed to GET /api/workflow by id: %v", err)
	}
	defer resp2.Body.Close()

	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("Expected 200 OK, got %d", resp2.StatusCode)
	}
	wfRsp, _ := util.DecodeJSONBodyResponse[models.WorkflowApiResponse](resp2)
	slog.Info("Workflow response:", "wfRsp", wfRsp)
	// ---- Assertions ----
	if wfRsp.Status != state {
		t.Errorf("Expected workflow status to be %s, got %s", state, wfRsp.Status)
	}
}
