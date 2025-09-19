package domain

import "time"

type WorkflowAction struct {
	ID             int64     // BIGSERIAL
	WorkflowID     int64     // BIGSERIAL (foreign key)
	ExecutorID     int64     // BIGINT (foreign key to executors.id)
	ExecutionCount int       // INT
	RetryCount     int       // INT
	Type           string    // TEXT
	Name           string    // TEXT
	Text           string    // TEXT
	DateTime       time.Time // TIMESTAMP
}
