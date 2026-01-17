package mysql

import (
	"context"
	"io/fs"
	"log/slog"
	"os"
	"strconv"
	"sync/atomic"
	"testing"

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
	container, _ := SetupMySQLTestInstance(t.Context())
	defer container.Terminate(t.Context())
	testFunc(t, port)
}

func SetupMySQLTestInstance(ctx context.Context) (testcontainers.Container, string) {
	req := testcontainers.ContainerRequest{
		Image:        "mysql:8.1", // MySQL image
		ExposedPorts: []string{"3306/tcp"},
		Env: map[string]string{
			"MYSQL_ROOT_PASSWORD": "test",
			"MYSQL_USER":          "test",
			"MYSQL_PASSWORD":      "test",
			"MYSQL_DATABASE":      "testdb",
		},
		WaitingFor: wait.ForListeningPort("3306/tcp"),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		slog.Error("error starting MySQL container", "error", err)
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
	
	return container, dsn
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
