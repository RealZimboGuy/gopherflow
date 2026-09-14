package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/RealZimboGuy/gopherflow/internal/migrations"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/mysql"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	_ "github.com/go-sql-driver/mysql"
)

var portBase int32 = 9048 // starting port number (can be anything safe)

func nextPort() int {
	return int(atomic.AddInt32(&portBase, 1))
}

func RunTestWithSetup(t *testing.T, testFunc func(t *testing.T, port int)) {
	port := nextPort()
	os.Setenv("HTTP_ADDR", ":"+strconv.Itoa(port))
	container, dsn, err := SetupMySQLTestInstance(t.Context())
	if err != nil {
		t.Fatalf("could not start the MySQL test container: %v", err)
	}
	defer container.Terminate(t.Context())
	waitForDB(t, dsn)
	testFunc(t, port)
}

func SetupMySQLTestInstance(ctx context.Context) (testcontainers.Container, string, error) {
	req := testcontainers.ContainerRequest{
		Image:        "mysql:8.1", // MySQL image
		ExposedPorts: []string{"3306/tcp"},
		Env: map[string]string{
			"MYSQL_ROOT_PASSWORD": "test",
			"MYSQL_USER":          "test",
			"MYSQL_PASSWORD":      "test",
			"MYSQL_DATABASE":      "testdb",
		},
		// The mysql image boots a temporary server for initialisation, shuts it
		// down, then starts the real one. Waiting only for the port can connect
		// to the temporary server and fail with "Server shutdown in progress",
		// so wait for the second "ready for connections" log line too.
		WaitingFor: wait.ForAll(
			wait.ForLog("ready for connections").WithOccurrence(2).WithStartupTimeout(3*time.Minute),
			wait.ForListeningPort("3306/tcp"),
		).WithDeadline(4 * time.Minute),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return nil, "", fmt.Errorf("start MySQL container: %w", err)
	}

	port, _ := container.MappedPort(ctx, "3306")

	// MySQL DSN: mysql://workflow:password@tcp(localhost:3306)/workflow?parseTime=true
	dsn := "mysql://test:test@tcp(localhost:" + port.Port() + ")/testdb?parseTime=true"
	os.Setenv("GFLOW_DATABASE_TYPE", "MYSQL")
	os.Setenv("GFLOW_DATABASE_URL", dsn)

	// Run migrations directly
	if err := runMigrationsFromEmbed("mysql", dsn); err != nil {
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

	connStr := strings.TrimPrefix(dsn, "mysql://")

	deadline := time.Now().Add(90 * time.Second)
	var lastErr error
	for time.Now().Before(deadline) {
		db, err := sql.Open("mysql", connStr)
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
