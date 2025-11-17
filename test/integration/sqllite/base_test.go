package sqllite

import (
	"context"
	"os"
	"strconv"
	"sync/atomic"
	"testing"
)

var portBase int32 = 9018 // starting port number (can be anything safe)

func nextPort() int {
	return int(atomic.AddInt32(&portBase, 1))
}
func runTestWithSetup(t *testing.T, testFunc func(t *testing.T, port int)) {
	port := nextPort()
	os.Setenv("HTTP_ADDR", ":"+strconv.Itoa(port))
	SetupSqlLiteTestInstance(t.Context())
	testFunc(t, port)
}

func SetupSqlLiteTestInstance(ctx context.Context) {

	os.Setenv("GFLOW_DATABASE_TYPE", "SQLLITE")
	os.Setenv("GFLOW_DATABASE_SQLLITE_FILE_NAME", "memory")
}
