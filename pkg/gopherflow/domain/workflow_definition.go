package domain

import "time"

type WorkflowDefinition struct {
	Name        string
	Description string
	Created     time.Time
	Updated     time.Time
	FlowChart   string
}
