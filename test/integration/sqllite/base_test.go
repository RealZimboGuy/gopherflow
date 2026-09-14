package sqllite

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"sync/atomic"
	"testing"

	_ "modernc.org/sqlite"
)

var portBase int32 = 9018 // starting port number (can be anything safe)

func nextPort() int {
	return int(atomic.AddInt32(&portBase, 1))
}
func RunTestWithSetup(t *testing.T, testFunc func(t *testing.T, port int)) {
	port := nextPort()
	filename := fmt.Sprintf("gopherflow-test-%d.db", port)
	// WAL mode writes -wal and -shm sidecar files next to the database; remove
	// them too or they are left behind in the working tree.
	defer func() {
		for _, suffix := range []string{"", "-wal", "-shm"} {
			os.Remove(filename + suffix)
		}
	}()
	os.Setenv("HTTP_ADDR", ":"+strconv.Itoa(port))
	SetupSqlLiteTestInstance(t.Context(), filename)
	testFunc(t, port)
}

func SetupSqlLiteTestInstance(ctx context.Context, filename string) {

	os.Setenv("GFLOW_DATABASE_TYPE", "SQLLITE")
	os.Setenv("GFLOW_DATABASE_SQLLITE_FILE_NAME", filename)
}
