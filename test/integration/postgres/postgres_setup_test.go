package postgres

import (
	"context"
	"database/sql"
	"log/slog"
	"os"

	_ "github.com/lib/pq"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

func SetupPostgresTestInstance(ctx context.Context) (testcontainers.Container, string) {

	req := testcontainers.ContainerRequest{
		Image:        "postgres:16-alpine",
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_PASSWORD": "test",
			"POSTGRES_USER":     "test",
			"POSTGRES_DB":       "testdb",
		},
		WaitingFor: wait.ForListeningPort("5432/tcp"),
	}
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		slog.Error("error starting postgres container", "error", err)
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
	return container, dsn
}
