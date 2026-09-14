package gopherflow

import (
	"net/http"
	"path/filepath"
	"testing"

	"github.com/RealZimboGuy/gopherflow/internal/config"
	"github.com/RealZimboGuy/gopherflow/pkg/gopherflow/core"
)

func TestSetupWiresTheAppAgainstSQLite(t *testing.T) {
	t.Setenv(config.DATABASE_TYPE, config.DATABASE_TYPE_SQLLITE)
	t.Setenv(config.DATABASE_SQLLITE_FILE_NAME, filepath.Join(t.TempDir(), "setup.db"))

	// Setup registers routes on the default mux; Shutdown replaces it, so
	// restore whatever was there for any test that runs afterwards.
	prevMux := http.DefaultServeMux
	t.Cleanup(func() { http.DefaultServeMux = prevMux })

	registry := map[string]func() core.Workflow{}
	app := Setup(registry)

	if app == nil {
		t.Fatal("Setup returned nil")
	}
	if app.DB == nil {
		t.Fatal("Setup did not open a database")
	}
	if err := app.DB.Ping(); err != nil {
		t.Fatalf("the database is not usable: %v", err)
	}
	if app.Manager == nil {
		t.Error("Setup did not build a workflow manager")
	}
	if app.Repos.Workflows == nil || app.Repos.Actions == nil ||
		app.Repos.Executors == nil || app.Repos.Definitions == nil || app.Repos.Users == nil {
		t.Error("Setup did not wire all repositories")
	}

	// Migrations must have run: the seeded admin user is present.
	admin, err := app.Repos.Users.FindByUsername("admin")
	if err != nil {
		t.Fatalf("FindByUsername: %v", err)
	}
	if admin == nil {
		t.Error("migrations did not run; the seeded admin user is missing")
	}

	app.Shutdown()
}

func TestSetupRejectsAnUnknownDatabaseType(t *testing.T) {
	t.Setenv(config.DATABASE_TYPE, "ORACLE")

	defer func() {
		if recover() == nil {
			t.Error("Setup accepted an unsupported database type; want a panic")
		}
	}()

	_ = Setup(map[string]func() core.Workflow{})
}

func TestSetupRejectsAMissingDatabaseType(t *testing.T) {
	t.Setenv(config.DATABASE_TYPE, "")

	defer func() {
		if recover() == nil {
			t.Error("Setup accepted an empty database type; want a panic")
		}
	}()

	_ = Setup(map[string]func() core.Workflow{})
}
