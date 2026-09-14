package common

import (
	"fmt"
	"net"
	"testing"
	"time"
)

// WaitForHTTP blocks until the app's HTTP server accepts connections on port.
//
// The integration tests start the engine with app.Run in a goroutine and then
// issue requests immediately. The listener is not up yet at that point, which a
// fast developer machine usually gets away with and a loaded CI runner does
// not, producing "connection refused" that looks like a flaky test.
func WaitForHTTP(t *testing.T, port int) {
	t.Helper()

	addr := fmt.Sprintf("localhost:%d", port)
	deadline := time.Now().Add(30 * time.Second)

	var lastErr error
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", addr, time.Second)
		if err == nil {
			_ = conn.Close()
			return
		}
		lastErr = err
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("the HTTP server never started listening on %s: %v", addr, lastErr)
}
