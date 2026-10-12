// Package benchmark measures end-to-end engine throughput against a real
// database, so the numbers quoted in the README can be reproduced.
//
// It is skipped by default because it is a load test, not a correctness test.
// Run it explicitly:
//
//	GFLOW_BENCH=1 go test ./test/benchmark/ -run TestThroughput -v -timeout 20m
//
// Tuning knobs (all optional):
//
//	GFLOW_BENCH_WORKFLOWS=2000   how many workflows to push through
//	GFLOW_ENGINE_EXECUTOR_SIZE=32
//	GFLOW_ENGINE_BATCH_SIZE=200
package benchmark

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/RealZimboGuy/gopherflow/pkg/gopherflow"
	"github.com/RealZimboGuy/gopherflow/pkg/gopherflow/core"
	"github.com/RealZimboGuy/gopherflow/pkg/gopherflow/domain"
	"github.com/RealZimboGuy/gopherflow/test/integration/common"

	_ "modernc.org/sqlite"
)

func envInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return def
}

// TestThroughput seeds a batch of workflows, runs the engine until every one
// of them reaches a terminal state, and reports the sustained rate.
//
// The unit that matters is state transitions per second: every transition is a
// state function plus a workflow row update plus an action-history insert, so
// it is the engine's real unit of work. Workflows per second is reported too
// because it is the number people actually want.
func TestThroughput(t *testing.T) {
	if os.Getenv("GFLOW_BENCH") == "" {
		t.Skip("load test; set GFLOW_BENCH=1 to run")
	}

	total := envInt("GFLOW_BENCH_WORKFLOWS", 2000)
	dbFile := fmt.Sprintf("bench-%d.db", time.Now().UnixNano())
	defer func() {
		for _, suffix := range []string{"", "-wal", "-shm"} {
			os.Remove(dbFile + suffix)
		}
	}()

	os.Setenv("GFLOW_DATABASE_TYPE", "SQLLITE")
	os.Setenv("GFLOW_DATABASE_SQLLITE_FILE_NAME", dbFile)
	os.Setenv("HTTP_ADDR", ":0")
	// Poll tightly and take big bites: the defaults are tuned for a quiet
	// system, not for saturation.
	if os.Getenv("GFLOW_ENGINE_EXECUTOR_SIZE") == "" {
		os.Setenv("GFLOW_ENGINE_EXECUTOR_SIZE", "32")
	}
	if os.Getenv("GFLOW_ENGINE_BATCH_SIZE") == "" {
		os.Setenv("GFLOW_ENGINE_BATCH_SIZE", "200")
	}
	os.Setenv("GFLOW_ENGINE_CHECK_DB_INTERVAL", "50ms")

	gopherflow.SetupLogger(slog.LevelError)

	registry := map[string]func() core.Workflow{
		"QuickWorkflow": func() core.Workflow { return &common.QuickWorkflow{} },
	}
	app := gopherflow.Setup(registry)

	// Seed every workflow before starting the engine, so the measured window
	// contains execution only and not our own insert loop.
	seedStart := time.Now()
	for i := 0; i < total; i++ {
		wf := &domain.Workflow{
			Status:         "NEW",
			ExecutorGroup:  "default",
			WorkflowType:   "QuickWorkflow",
			ExternalID:     fmt.Sprintf("bench-%d", i),
			BusinessKey:    fmt.Sprintf("bench-%d", i),
			State:          "Init",
			NextActivation: sql.NullTime{Time: time.Now(), Valid: true},
		}
		if _, err := app.Repos.Workflows.Save(wf); err != nil {
			t.Fatalf("seed workflow %d: %v", i, err)
		}
	}
	seedElapsed := time.Since(seedStart)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	start := time.Now()
	go func() {
		if err := app.Run(ctx); err != nil {
			slog.Error("engine exited", "error", err)
		}
	}()

	deadline := time.Now().Add(15 * time.Minute)
	var done int
	for time.Now().Before(deadline) {
		if err := app.DB.QueryRow(
			`SELECT count(*) FROM workflow WHERE status IN ('FINISHED','MANUAL')`,
		).Scan(&done); err != nil {
			t.Fatalf("count finished: %v", err)
		}
		if done >= total {
			break
		}
		time.Sleep(25 * time.Millisecond)
	}
	elapsed := time.Since(start)

	if done < total {
		t.Fatalf("only %d/%d workflows finished in %s", done, total, elapsed)
	}

	var transitions int
	if err := app.DB.QueryRow(`SELECT count(*) FROM workflow_actions`).Scan(&transitions); err != nil {
		t.Fatalf("count actions: %v", err)
	}

	secs := elapsed.Seconds()
	t.Logf("\n"+
		"  workflows            %d\n"+
		"  state transitions    %d\n"+
		"  wall time            %s\n"+
		"  workflows/sec        %.0f\n"+
		"  workflows/min        %.0f\n"+
		"  transitions/sec      %.0f\n"+
		"  executors            %s\n"+
		"  batch size           %s\n"+
		"  seed time (excluded) %s\n",
		total, transitions, elapsed.Round(time.Millisecond),
		float64(total)/secs, float64(total)/secs*60,
		float64(transitions)/secs,
		os.Getenv("GFLOW_ENGINE_EXECUTOR_SIZE"),
		os.Getenv("GFLOW_ENGINE_BATCH_SIZE"),
		seedElapsed.Round(time.Millisecond),
	)
}
