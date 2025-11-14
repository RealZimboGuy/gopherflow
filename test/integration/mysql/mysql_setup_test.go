package mysql

import (
	"context"
	"log/slog"
	"os"

	_ "github.com/go-sql-driver/mysql"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

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

	//host, _ := container.Host(ctx)
	port, _ := container.MappedPort(ctx, "3306")

	// MySQL DSN: mysql://workflow:password@tcp(localhost:3306)/workflow?parseTime=true
	dsn := "mysql://test:test@tcp(localhost:" + port.Port() + ")/testdb?parseTime=true"
	os.Setenv("GFLOW_DATABASE_TYPE", "MYSQL")
	os.Setenv("GFLOW_DATABASE_URL", dsn)
	return container, dsn
}
