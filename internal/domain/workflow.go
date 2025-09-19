package domain

import "time"
import "database/sql"

type Workflow struct {
	ID             int64
	Status         string
	ExecutionCount int
	RetryCount     int
	Created        time.Time
	Modified       time.Time
	NextActivation sql.NullTime
	Started        sql.NullTime
	ExecutorID     sql.NullString
	ExecutorGroup  string
	WorkflowType   string
	ExternalID     string
	BusinessKey    string
	State          string
	StateVars      sql.NullString
}
