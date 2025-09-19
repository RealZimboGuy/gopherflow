package models

type StateType string

const (
	StateStart  StateType = "Start"  // Initial state
	StateNormal StateType = "Normal" // Normal state
	StateManual StateType = "Manual" // Manual state
	StateError  StateType = "Error"  // Error state
	StateEnd    StateType = "End"    // End state
)
