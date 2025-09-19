package models

import "time"

type NextState struct {
	Name                string    // Name of the state
	ActionLog           string    // Additional information about the state
	NextExecution       time.Time //specific time set by the code
	NextExecutionOffset string    // a  human friendly time string sent to the database ie 10 minutes
}
