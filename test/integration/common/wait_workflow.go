package common

import (
	"context"

	"github.com/RealZimboGuy/gopherflow/pkg/gopherflow/core"
	domain "github.com/RealZimboGuy/gopherflow/pkg/gopherflow/domain"
	models "github.com/RealZimboGuy/gopherflow/pkg/gopherflow/models"

	"log/slog"
	"time"
)

type WaitWorkflow struct {
	core.BaseWorkflow
}

func (m *WaitWorkflow) Setup(wf *domain.Workflow) {
	m.BaseWorkflow.Setup(wf)
}
func (m *WaitWorkflow) GetWorkflowData() *domain.Workflow {
	return m.WorkflowState
}
func (m *WaitWorkflow) GetStateVariables() map[string]string {
	return m.StateVariables
}
func (m *WaitWorkflow) InitialState() string {
	return StateInit
}

func (m *WaitWorkflow) Description() string {
	return "This is a quick Workflow showing how it can be used"
}

func (m *WaitWorkflow) GetRetryConfig() models.RetryConfig {
	return models.RetryConfig{
		MaxRetryCount:    10,
		RetryIntervalMin: time.Second * 10,
		RetryIntervalMax: time.Minute * 60,
	}
}

func (m *WaitWorkflow) StateTransitions() map[string][]string {
	return map[string][]string{
		StateInit:      []string{StateGetIpData}, // Init -> StateGetIpData
		StateGetIpData: []string{StateFinish},    // StateGetIpData -> finish
	}
}
func (m *WaitWorkflow) GetAllStates() []models.WorkflowState {
	states := []models.WorkflowState{
		{Name: StateInit, StateType: models.StateStart},
		{Name: StateGetIpData, StateType: models.StateNormal},
		{Name: StateFinish, StateType: models.StateEnd},
	}
	return states
}

// Each method returns the next state
func (m *WaitWorkflow) Init(ctx context.Context) (*models.NextState, error) {
	slog.InfoContext(ctx, "Starting workflow")

	return &models.NextState{
		Name:          StateGetIpData,
		NextExecution: time.Now().Add(10 * time.Minute),
	}, nil
}

func (m *WaitWorkflow) StateGetIpData(ctx context.Context) (*models.NextState, error) {

	m.StateVariables[VAR_IP] = "127.0.0.1"

	return &models.NextState{
		Name: StateFinish,
	}, nil
}
