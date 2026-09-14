package domain

import "time"
import "database/sql"

// Workflow is one persisted workflow instance: its current state, scheduling
// information and serialised state variables. It is the row the engine loads,
// executes and writes back on every state transition.
type Workflow struct {
	ID               int64
	Status           string
	ExecutionCount   int
	RetryCount       int
	Created          time.Time
	Modified         time.Time
	NextActivation   sql.NullTime
	Started          sql.NullTime
	ExecutorID       sql.NullString
	ExecutorGroup    string
	WorkflowType     string
	ExternalID       string
	BusinessKey      string
	State            string
	StateVars        sql.NullString
	ParentWorkflowID sql.NullInt64
}
