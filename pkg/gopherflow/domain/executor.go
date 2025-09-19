package domain

import "time"

type Executor struct {
	ID         int64     // BIGSERIAL
	Name       string    // TEXT
	Started    time.Time // TIMESTAMP
	LastActive time.Time // TIMESTAMP
}
