package controllers

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthController_Healthz_OK(t *testing.T) {
	rc := newRealController(t)

	c := NewHealthController(rc.DB)

	req := httptest.NewRequest("GET", "/healthz", nil)
	w := httptest.NewRecorder()

	c.handleHealthz(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", resp.StatusCode)
	}

	var body map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	if body["status"] != "ok" {
		t.Errorf("Expected status ok, got %q", body["status"])
	}
	if body["database"] != "ok" {
		t.Errorf("Expected database ok, got %q", body["database"])
	}
}

func TestHealthController_Healthz_DatabaseFailure(t *testing.T) {
	tests := []struct {
		name string
		db   *sql.DB
	}{
		{
			name: "nil database",
			db:   nil,
		},
		{
			name: "unreachable database",
			db:   sql.OpenDB(failingConnector{}),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewHealthController(tt.db)

			req := httptest.NewRequest("GET", "/healthz", nil)
			w := httptest.NewRecorder()

			c.handleHealthz(w, req)

			resp := w.Result()
			if resp.StatusCode != http.StatusServiceUnavailable {
				t.Fatalf("Expected status 503, got %d", resp.StatusCode)
			}

			var body map[string]string
			if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
				t.Fatalf("Failed to decode response: %v", err)
			}
			if body["status"] != "unavailable" {
				t.Errorf("Expected status unavailable, got %q", body["status"])
			}
		})
	}
}

var errFailingConnect = errors.New("database unavailable")

type failingConnector struct{}

func (failingConnector) Connect(context.Context) (driver.Conn, error) {
	return nil, errFailingConnect
}

func (failingConnector) Driver() driver.Driver {
	return failingDriver{}
}

type failingDriver struct{}

func (failingDriver) Open(string) (driver.Conn, error) {
	return nil, errFailingConnect
}
