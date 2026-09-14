package models

// StateType classifies a workflow state. The engine uses it to decide which
// states start a workflow and which terminate it; the console uses it to colour
// the generated flow chart.
type StateType string

const (
	StateStart  StateType = "Start"  // Initial state
	StateNormal StateType = "Normal" // Normal state
	StateManual StateType = "Manual" // Manual state
	StateError  StateType = "Error"  // Error state
	StateEnd    StateType = "End"    // End state
)
