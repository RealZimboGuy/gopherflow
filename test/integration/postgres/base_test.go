package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"github.com/RealZimboGuy/gopherflow/internal/migrations"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	_ "github.com/lib/pq"
)

var portBase int32 = 9098 // starting port number (can be anything safe)

func nextPort() int {
	return int(atomic.AddInt32(&portBase, 1))
}

func RunTestWithSetup(t *testing.T, testFunc func(t *testing.T, port int)) {
	port := nextPort()
	os.Setenv("HTTP_ADDR", ":"+strconv.Itoa(port))
	container, dsn, err := SetupPostgresTestInstance(t.Context())
	if err != nil {
		t.Fatalf("could not start the Postgres test container: %v", err)
	}
	defer container.Terminate(t.Context())
	waitForDB(t, dsn)
	testFunc(t, port)
}

func SetupPostgresTestInstance(ctx context.Context) (testcontainers.Container, string, error) {
	req := testcontainers.ContainerRequest{
		Image:        "postgres:16-alpine",
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_PASSWORD": "test",
			"POSTGRES_USER":     "test",
			"POSTGRES_DB":       "testdb",
		},
		// The postgres image also boots a temporary server during init, so wait
		// for the second readiness line rather than just the port.
		WaitingFor: wait.ForAll(
			wait.ForLog("database system is ready to accept connections").WithOccurrence(2).WithStartupTimeout(3*time.Minute),
			wait.ForListeningPort("5432/tcp"),
		).WithDeadline(4 * time.Minute),
	}
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return nil, "", fmt.Errorf("start Postgres container: %w", err)
	}

	host, _ := container.Host(ctx)
	port, _ := container.MappedPort(ctx, "5432")

	dsn := "postgres://test:test@" + host + ":" + port.Port() + "/testdb?sslmode=disable"
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		slog.Error("error connecting to postgres", "error", err)
	}

	if err := db.Ping(); err != nil {
		slog.Error("error pinging postgres", "error", err)
	}

	os.Setenv("GFLOW_DATABASE_TYPE", "POSTGRES")
	os.Setenv("GFLOW_DATABASE_URL", dsn)

	// Run migrations directly
	if err := runMigrationsFromEmbed("postgres", dsn); err != nil {
		slog.Error("DB migration failed", "error", err)
	}

	return container, dsn, nil
}

// runMigrationsFromEmbed runs database migrations from the embedded migrations FS
func runMigrationsFromEmbed(migrationsPath string, dbURL string) error {
	sub, err := fs.Sub(migrations.FS, migrationsPath)
	if err != nil {
		return err
	}
	source, err := iofs.New(sub, ".")
	if err != nil {
		return err
	}
	m, err := migrate.NewWithSourceInstance("iofs", source, dbURL)
	if err != nil {
		return err
	}
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return err
	}
	return nil
}

// waitForDB polls until the database accepts a real query. The container wait
// strategy watches log output, which can fire while the server is still
// restarting after initialisation, so verify with an actual connection.
func waitForDB(t *testing.T, dsn string) {
	t.Helper()

	// lib/pq takes the full URL including the scheme; stripping it makes the
	// driver fall back to parsing keyword/value pairs and dial the default port.
	connStr := dsn

	deadline := time.Now().Add(90 * time.Second)
	var lastErr error
	for time.Now().Before(deadline) {
		db, err := sql.Open("postgres", connStr)
		if err != nil {
			lastErr = err
			time.Sleep(500 * time.Millisecond)
			continue
		}
		if err := db.Ping(); err != nil {
			lastErr = err
			_ = db.Close()
			time.Sleep(500 * time.Millisecond)
			continue
		}
		var one int
		if err := db.QueryRow("SELECT 1").Scan(&one); err != nil {
			lastErr = err
			_ = db.Close()
			time.Sleep(500 * time.Millisecond)
			continue
		}
		_ = db.Close()
		return
	}
	t.Fatalf("database never became ready: %v", lastErr)
}
