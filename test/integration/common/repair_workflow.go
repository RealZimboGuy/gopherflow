package common

import (
	"context"

	"github.com/RealZimboGuy/gopherflow/pkg/gopherflow/core"
	domain "github.com/RealZimboGuy/gopherflow/pkg/gopherflow/domain"
	models "github.com/RealZimboGuy/gopherflow/pkg/gopherflow/models"

	"log/slog"
	"time"
)

type RepairWorkflow struct {
	core.BaseWorkflow
	Clock core.Clock
}

func (m *RepairWorkflow) Setup(wf *domain.Workflow) {
	m.BaseWorkflow.Setup(wf)
}
func (m *RepairWorkflow) GetWorkflowData() *domain.Workflow {
	return m.WorkflowState
}
func (m *RepairWorkflow) GetStateVariables() map[string]string {
	return m.StateVariables
}
func (m *RepairWorkflow) InitialState() string {
	return StateInit
}

func (m *RepairWorkflow) Description() string {
	return "This is a quick Workflow showing how it can be used"
}

func (m *RepairWorkflow) GetRetryConfig() models.RetryConfig {
	return models.RetryConfig{
		MaxRetryCount:    10,
		RetryIntervalMin: time.Second * 10,
		RetryIntervalMax: time.Minute * 60,
	}
}

func (m *RepairWorkflow) StateTransitions() map[string][]string {
	return map[string][]string{
		StateInit:      []string{StateGetIpData}, // Init -> StateGetIpData
		StateGetIpData: []string{StateFinish},    // StateGetIpData -> finish
	}
}
func (m *RepairWorkflow) GetAllStates() []models.WorkflowState {
	states := []models.WorkflowState{
		{Name: StateInit, StateType: models.StateStart},
		{Name: StateGetIpData, StateType: models.StateNormal},
		{Name: StateFinish, StateType: models.StateEnd},
	}
	return states
}

// Each method returns the next state
func (m *RepairWorkflow) Init(ctx context.Context) (*models.NextState, error) {
	slog.InfoContext(ctx, "Starting workflow")

	return &models.NextState{
		Name: StateGetIpData,
	}, nil
}

func (m *RepairWorkflow) StateGetIpData(ctx context.Context) (*models.NextState, error) {

	m.StateVariables[VAR_IP] = "127.0.0.1"

	slog.InfoContext(ctx, "Sleeping for 1 hour")
	select {
	case <-ctx.Done():
		return nil, ctx.Err() // normal cancel

	case <-m.Clock.After(time.Hour * 1):

	}

	return &models.NextState{
		Name: StateFinish,
	}, nil
}
