package gopherflow

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/RealZimboGuy/gopherflow/pkg/gopherflow/core"
)

// stubClock returns a fixed time so the injected timestamp is assertable.
type stubClock struct{ t time.Time }

func (c stubClock) Now() time.Time                         { return c.t }
func (c stubClock) After(d time.Duration) <-chan time.Time { return time.After(d) }
func (c stubClock) Sleep(d time.Duration)                  { time.Sleep(d) }

// newCapturingHandler wraps a JSON handler writing into buf, so the attributes
// logHandler adds can be inspected.
func newCapturingHandler(buf *bytes.Buffer, clock core.Clock) *logHandler {
	return &logHandler{
		Handler: slog.NewJSONHandler(buf, &slog.HandlerOptions{Level: slog.LevelDebug}),
		Clock:   clock,
	}
}

func TestLogHandlerAddsSeverity(t *testing.T) {
	tests := []struct {
		level slog.Level
		want  string
	}{
		{slog.LevelError, "ERROR"},
		{slog.LevelWarn, "WARNING"},
		{slog.LevelInfo, "INFO"},
		{slog.LevelDebug, "DEBUG"},
		{slog.LevelDebug - 4, "DEFAULT"},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			var buf bytes.Buffer
			h := newCapturingHandler(&buf, stubClock{t: time.Now()})

			rec := slog.NewRecord(time.Now(), tt.level, "message", 0)
			if err := h.Handle(context.Background(), rec); err != nil {
				t.Fatalf("Handle: %v", err)
			}

			if !strings.Contains(buf.String(), `"severity":"`+tt.want+`"`) {
				t.Errorf("log line %s does not carry severity %q", buf.String(), tt.want)
			}
		})
	}
}

func TestLogHandlerUsesTheInjectedClock(t *testing.T) {
	fixed := time.Date(2026, 3, 4, 5, 6, 7, 0, time.UTC)
	var buf bytes.Buffer
	h := newCapturingHandler(&buf, stubClock{t: fixed})

	// The record carries a different time; the handler must override it.
	rec := slog.NewRecord(time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC), slog.LevelInfo, "message", 0)
	if err := h.Handle(context.Background(), rec); err != nil {
		t.Fatalf("Handle: %v", err)
	}

	if !strings.Contains(buf.String(), "2026-03-04") {
		t.Errorf("log line %s does not use the injected clock time", buf.String())
	}
}

func TestLogHandlerAddsContextFields(t *testing.T) {
	var buf bytes.Buffer
	h := newCapturingHandler(&buf, stubClock{t: time.Now()})

	ctx := context.WithValue(context.Background(), core.CtxKeyExecutorId, "exec-7")
	ctx = context.WithValue(ctx, core.CtxKeyUsername, "alice")

	rec := slog.NewRecord(time.Now(), slog.LevelInfo, "message", 0)
	if err := h.Handle(ctx, rec); err != nil {
		t.Fatalf("Handle: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "exec-7") {
		t.Errorf("log line %s is missing the executor id from the context", out)
	}
	if !strings.Contains(out, "alice") {
		t.Errorf("log line %s is missing the username from the context", out)
	}
}

func TestLogHandlerIgnoresAbsentAndWrongTypedContextValues(t *testing.T) {
	var buf bytes.Buffer
	h := newCapturingHandler(&buf, stubClock{t: time.Now()})

	// Wrong type and empty string must both be skipped rather than panicking.
	ctx := context.WithValue(context.Background(), core.CtxKeyExecutorId, 42)
	ctx = context.WithValue(ctx, core.CtxKeyUsername, "")

	rec := slog.NewRecord(time.Now(), slog.LevelInfo, "message", 0)
	if err := h.Handle(ctx, rec); err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if !strings.Contains(buf.String(), `"severity":"INFO"`) {
		t.Errorf("log line %s was not written", buf.String())
	}
}

func TestLogHandlerDoesNotMutateTheOriginalRecord(t *testing.T) {
	var buf bytes.Buffer
	h := newCapturingHandler(&buf, stubClock{t: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)})

	original := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	rec := slog.NewRecord(original, slog.LevelInfo, "message", 0)
	before := rec.NumAttrs()

	if err := h.Handle(context.Background(), rec); err != nil {
		t.Fatalf("Handle: %v", err)
	}

	if !rec.Time.Equal(original) {
		t.Errorf("the original record's time was mutated to %v", rec.Time)
	}
	if rec.NumAttrs() != before {
		t.Errorf("attributes were added to the original record: %d -> %d", before, rec.NumAttrs())
	}
}

func TestSetupLoggerInstallsADefault(t *testing.T) {
	prev := slog.Default()
	t.Cleanup(func() { slog.SetDefault(prev) })

	SetupLogger(slog.LevelInfo)
	if slog.Default() == nil {
		t.Fatal("SetupLogger left a nil default logger")
	}

	SetupLoggerWithClock(slog.LevelDebug, stubClock{t: time.Now()})
	if slog.Default() == nil {
		t.Fatal("SetupLoggerWithClock left a nil default logger")
	}
}
