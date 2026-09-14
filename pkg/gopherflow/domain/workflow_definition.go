package domain

import "time"

// WorkflowDefinition is the registered description of a workflow type, including
// the flow chart generated from its state transitions. The engine writes one per
// registered workflow at startup.
type WorkflowDefinition struct {
	Name        string
	Description string
	Created     time.Time
	Updated     time.Time
	FlowChart   string
}
