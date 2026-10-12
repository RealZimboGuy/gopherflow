package engine

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"strings"
	"testing"

	"github.com/RealZimboGuy/gopherflow/pkg/gopherflow/core"
	"github.com/RealZimboGuy/gopherflow/pkg/gopherflow/domain"
)

// captureWarnings swaps the default logger for one that records WARN and
// above, and restores the original when the test ends.
func captureWarnings(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelWarn})))
	t.Cleanup(func() { slog.SetDefault(prev) })
	return &buf
}

func registryWithMockWorkflow() *map[string]func() core.Workflow {
	return &map[string]func() core.Workflow{
		"MockWorkflow": func() core.Workflow { return &MockWorkflow{} },
	}
}

// On a fresh database no definition exists yet, so FindByName returns
// sql.ErrNoRows for every workflow. That is the expected path, not a fault,
// and must not be logged as a warning: it is the first thing a new user sees.
func TestRegisterWorkflowDefinitionsFirstStartIsQuiet(t *testing.T) {
	logs := captureWarnings(t)

	var saved []string
	defRepo := &MockDefinitionRepo{
		FindByNameFunc: func(string) (*domain.WorkflowDefinition, error) {
			return nil, sql.ErrNoRows
		},
		SaveFunc: func(def *domain.WorkflowDefinition) error {
			saved = append(saved, def.Name)
			return nil
		},
	}
	wm := NewWorkflowManager(nil, nil, nil, defRepo, registryWithMockWorkflow(), core.NewRealClock())

	registerWorkflowDefinitions(context.Background(), wm)

	if len(saved) != 1 || saved[0] != "MockWorkflow" {
		t.Fatalf("expected MockWorkflow to be created, saved = %v", saved)
	}
	if logs.Len() != 0 {
		t.Fatalf("first start logged a warning:\n%s", logs.String())
	}
}

// A lookup that fails for any other reason is still worth reporting.
func TestRegisterWorkflowDefinitionsWarnsOnRealLookupFailure(t *testing.T) {
	logs := captureWarnings(t)

	defRepo := &MockDefinitionRepo{
		FindByNameFunc: func(string) (*domain.WorkflowDefinition, error) {
			return nil, errors.New("connection reset by peer")
		},
	}
	wm := NewWorkflowManager(nil, nil, nil, defRepo, registryWithMockWorkflow(), core.NewRealClock())

	registerWorkflowDefinitions(context.Background(), wm)

	out := logs.String()
	if !strings.Contains(out, "lookup failed") || !strings.Contains(out, "connection reset by peer") {
		t.Fatalf("expected a warning carrying the underlying error, got:\n%s", out)
	}
}
