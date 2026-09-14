package domain

import "time"

// Executor is a running engine instance. Executors register themselves and
// heartbeat through LastActive so that stuck workflows left behind by a dead
// executor can be picked up again.
type Executor struct {
	ID         int64     // BIGSERIAL
	Name       string    // TEXT
	Started    time.Time // TIMESTAMP
	LastActive time.Time // TIMESTAMP
}
