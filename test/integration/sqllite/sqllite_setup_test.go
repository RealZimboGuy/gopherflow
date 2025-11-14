package postgres

import (
	"context"
	"os"

	_ "github.com/lib/pq"
)

func SetupSqlLiteTestInstance(ctx context.Context) {

	os.Setenv("GFLOW_DATABASE_TYPE", "SQLLITE")
	os.Setenv("GFLOW_DATABASE_SQLLITE_FILE_NAME", "memory")
}
