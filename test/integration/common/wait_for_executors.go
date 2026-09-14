package common

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/RealZimboGuy/gopherflow/internal/util"
	"github.com/RealZimboGuy/gopherflow/pkg/gopherflow/domain"
)

// WaitForExecutors polls /api/executors until at least one executor has
// registered, and returns them.
//
// The engine registers its executor asynchronously after the HTTP server starts
// accepting connections, so a single immediate request races with startup. That
// race is usually won on a fast developer machine and lost on a loaded CI
// runner, which makes it look like a flaky test rather than a missing wait.
func WaitForExecutors(t *testing.T, port int, timeout time.Duration) []domain.Executor {
	t.Helper()

	url := fmt.Sprintf("http://localhost:%d/api/executors", port)
	client := &http.Client{Timeout: 10 * time.Second}
	deadline := time.Now().Add(timeout)

	var lastErr error
	for time.Now().Before(deadline) {
		executors, err := fetchExecutors(client, url)
		if err != nil {
			lastErr = err
		} else if len(executors) > 0 {
			return executors
		}
		time.Sleep(100 * time.Millisecond)
	}

	t.Fatalf("no executor registered within %s (last error: %v)", timeout, lastErr)
	return nil
}

func fetchExecutors(client *http.Client, url string) ([]domain.Executor, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", "b5f0e8c4-daa6-465c-bded-50ca22b798b2")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("expected 200 OK, got %d", resp.StatusCode)
	}

	return util.DecodeJSONBodyResponse[[]domain.Executor](resp)
}
