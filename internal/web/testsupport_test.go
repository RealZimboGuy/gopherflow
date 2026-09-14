package web

import (
	"database/sql"
	"io/fs"
	"path/filepath"
	"testing"
	"time"

	"github.com/RealZimboGuy/gopherflow/internal/config"
	"github.com/RealZimboGuy/gopherflow/internal/engine"
	"github.com/RealZimboGuy/gopherflow/internal/migrations"
	"github.com/RealZimboGuy/gopherflow/internal/repository"
	"github.com/RealZimboGuy/gopherflow/pkg/gopherflow/core"
	"github.com/RealZimboGuy/gopherflow/pkg/gopherflow/domain"
	"github.com/RealZimboGuy/gopherflow/pkg/gopherflow/models"
	migrate "github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/sqlite"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	_ "modernc.org/sqlite"
)

// testWeb wires a WebController to real repositories backed by a migrated
// SQLite database, so the handlers render against real data.
type testWeb struct {
	Ctrl     *WebController
	Users    *repository.UserRepository
	Workflow *repository.WorkflowRepository
	Actions  *repository.WorkflowActionRepository
	Defs     *repository.WorkflowDefinitionRepository
	Execs    *repository.ExecutorRepository
	DB       *sql.DB
}

func newTestWeb(t *testing.T) *testWeb {
	t.Helper()
	t.Setenv(config.DATABASE_TYPE, config.DATABASE_TYPE_SQLLITE)

	file := filepath.Join(t.TempDir(), "web.db")
	dsn := file + "?_busy_timeout=10000&_pragma=journal_mode(WAL)"

	sub, err := fs.Sub(migrations.FS, "sqllite3")
	if err != nil {
		t.Fatalf("migrations sub fs: %v", err)
	}
	source, err := iofs.New(sub, ".")
	if err != nil {
		t.Fatalf("iofs source: %v", err)
	}
	m, err := migrate.NewWithSourceInstance("iofs", source, "sqlite://"+dsn)
	if err != nil {
		t.Fatalf("migrate init: %v", err)
	}
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		t.Fatalf("migrate up: %v", err)
	}
	if se, de := m.Close(); se != nil || de != nil {
		t.Fatalf("migrate close: %v / %v", se, de)
	}

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	clock := core.NewRealClock()
	wfRepo := repository.NewWorkflowRepository(db, clock)
	actionRepo := repository.NewWorkflowActionRepository(db, clock)
	execRepo := repository.NewExecutorRepository(db, clock)
	defRepo := repository.NewWorkflowDefinitionRepository(db, clock)
	userRepo := repository.NewUserRepository(db, clock)

	registry := map[string]func() core.Workflow{
		"DemoWorkflow": func() core.Workflow { return &webTestWorkflow{} },
	}
	manager := engine.NewWorkflowManager(wfRepo, actionRepo, execRepo, defRepo, &registry, clock)

	return &testWeb{
		Ctrl:     NewWebController(manager, userRepo),
		Users:    userRepo,
		Workflow: wfRepo,
		Actions:  actionRepo,
		Defs:     defRepo,
		Execs:    execRepo,
		DB:       db,
	}
}

func (tw *testWeb) seedWorkflow(t *testing.T, externalID string) int64 {
	t.Helper()
	id, err := tw.Workflow.Save(&domain.Workflow{
		Status:         "NEW",
		ExecutorGroup:  "default",
		WorkflowType:   "DemoWorkflow",
		ExternalID:     externalID,
		BusinessKey:    "bk-" + externalID,
		State:          "Init",
		NextActivation: sql.NullTime{Time: time.Now().Add(-time.Hour).UTC(), Valid: true},
	})
	if err != nil {
		t.Fatalf("seed workflow %s: %v", externalID, err)
	}
	return id
}

func (tw *testWeb) seedDefinition(t *testing.T, name string) {
	t.Helper()
	err := tw.Defs.Save(&domain.WorkflowDefinition{
		Name: name, Description: name + " description",
		Created: time.Now(), Updated: time.Now(),
		FlowChart: "flowchart TD\n    Init --> Review",
	})
	if err != nil {
		t.Fatalf("seed definition %s: %v", name, err)
	}
}

// webTestWorkflow is a minimal core.Workflow for the manager's registry.
type webTestWorkflow struct {
	core.BaseWorkflow
}

func (w *webTestWorkflow) InitialState() string                 { return "Init" }
func (w *webTestWorkflow) Description() string                  { return "web test workflow" }
func (w *webTestWorkflow) GetWorkflowData() *domain.Workflow    { return w.WorkflowState }
func (w *webTestWorkflow) GetStateVariables() map[string]string { return w.StateVariables }

func (w *webTestWorkflow) GetRetryConfig() models.RetryConfig {
	return models.RetryConfig{
		MaxRetryCount:    3,
		RetryIntervalMin: time.Second,
		RetryIntervalMax: time.Minute,
	}
}

func (w *webTestWorkflow) StateTransitions() map[string][]string {
	return map[string][]string{
		"Init":   {"Review"},
		"Review": {"Finish"},
	}
}

func (w *webTestWorkflow) GetAllStates() []models.WorkflowState {
	return []models.WorkflowState{
		{Name: "Init", StateType: models.StateStart},
		{Name: "Review", StateType: models.StateNormal},
		{Name: "Finish", StateType: models.StateEnd},
	}
}
