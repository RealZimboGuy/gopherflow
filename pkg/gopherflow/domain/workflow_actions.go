package domain

import "time"

// WorkflowAction is a single entry in a workflow's history: a state change, a
// retry, or a log line. The console renders these as the workflow's audit trail.
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
