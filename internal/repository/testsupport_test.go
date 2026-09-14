package repository

import (
	"database/sql"
	"io/fs"
	"path/filepath"
	"testing"
	"time"

	"github.com/RealZimboGuy/gopherflow/internal/config"
	"github.com/RealZimboGuy/gopherflow/internal/migrations"
	"github.com/RealZimboGuy/gopherflow/pkg/gopherflow/core"
	"github.com/RealZimboGuy/gopherflow/pkg/gopherflow/domain"
	migrate "github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/sqlite"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	_ "modernc.org/sqlite"
)

// testClock returns a time that advances only when the test asks it to, so
// rows written in a single test are ordered predictably.
type testClock struct{ t time.Time }

func newTestClock() *testClock {
	return &testClock{t: time.Date(2026, 3, 4, 5, 6, 7, 0, time.UTC)}
}

func (c *testClock) Now() time.Time                         { return c.t }
func (c *testClock) After(d time.Duration) <-chan time.Time { return time.After(d) }
func (c *testClock) Sleep(d time.Duration)                  { time.Sleep(d) }
func (c *testClock) Advance(d time.Duration)                { c.t = c.t.Add(d) }

// newTestDB opens a fresh SQLite database with the real embedded migrations
// applied, so the tests run against the same schema the engine ships.
func newTestDB(t *testing.T) (*sql.DB, *testClock) {
	t.Helper()

	withDBType(t, config.DATABASE_TYPE_SQLLITE)

	file := filepath.Join(t.TempDir(), "test.db")
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
	sourceErr, dbErr := m.Close()
	if sourceErr != nil || dbErr != nil {
		t.Fatalf("migrate close: %v / %v", sourceErr, dbErr)
	}

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.Ping(); err != nil {
		t.Fatalf("ping db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	return db, newTestClock()
}

// newUserRepo returns a UserRepository backed by a fresh migrated database.
func newUserRepo(t *testing.T) (*UserRepository, *testClock) {
	t.Helper()
	db, clock := newTestDB(t)
	return NewUserRepository(db, clock), clock
}

// newUser builds a valid user row with a unique api key per username.
func newUser(username string) *domain.User {
	return &domain.User{
		Username: username,
		Password: "hashed-" + username,
		ApiKey:   sql.NullString{String: "key-" + username, Valid: true},
		Enabled:  sql.NullBool{Bool: true, Valid: true},
	}
}

// newWorkflowRepo returns a WorkflowRepository backed by a fresh migrated database.
func newWorkflowRepo(t *testing.T) (*WorkflowRepository, *testClock) {
	t.Helper()
	db, clock := newTestDB(t)
	return NewWorkflowRepository(db, clock), clock
}

// newWorkflow builds a runnable workflow row due for immediate execution.
func newWorkflow(externalID string) *domain.Workflow {
	return &domain.Workflow{
		Status:        "NEW",
		ExecutorGroup: "default",
		WorkflowType:  "DemoWorkflow",
		ExternalID:    externalID,
		BusinessKey:   "bk-" + externalID,
		State:         "Init",
	}
}

var _ core.Clock = (*testClock)(nil)
