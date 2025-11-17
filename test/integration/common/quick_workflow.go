package common

import (
	"context"

	"github.com/RealZimboGuy/gopherflow/pkg/gopherflow/core"
	domain "github.com/RealZimboGuy/gopherflow/pkg/gopherflow/domain"
	models "github.com/RealZimboGuy/gopherflow/pkg/gopherflow/models"

	"log/slog"
	"time"
)

// Define a named string type
var StateGetIpData string = "StateGetIpData"
var StateFinish string = "Finish"
var StateInit string = "Init"
var StateReview string = "Review"
var StateApprove string = "Approve"
var StateApproveError string = "ApproveError"

const VAR_IP = "ip"

type QuickWorkflow struct {
	core.BaseWorkflow
}

func (m *QuickWorkflow) Setup(wf *domain.Workflow) {
	m.BaseWorkflow.Setup(wf)
}
func (m *QuickWorkflow) GetWorkflowData() *domain.Workflow {
	return m.WorkflowState
}
func (m *QuickWorkflow) GetStateVariables() map[string]string {
	return m.StateVariables
}
func (m *QuickWorkflow) InitialState() string {
	return StateInit
}

func (m *QuickWorkflow) Description() string {
	return "This is a quick Workflow showing how it can be used"
}

func (m *QuickWorkflow) GetRetryConfig() models.RetryConfig {
	return models.RetryConfig{
		MaxRetryCount:    10,
		RetryIntervalMin: time.Second * 10,
		RetryIntervalMax: time.Minute * 60,
	}
}

func (m *QuickWorkflow) StateTransitions() map[string][]string {
	return map[string][]string{
		StateInit:      []string{StateGetIpData}, // Init -> StateGetIpData
		StateGetIpData: []string{StateFinish},    // StateGetIpData -> finish
	}
}
func (m *QuickWorkflow) GetAllStates() []models.WorkflowState {
	states := []models.WorkflowState{
		{Name: StateInit, StateType: models.StateStart},
		{Name: StateGetIpData, StateType: models.StateNormal},
		{Name: StateFinish, StateType: models.StateEnd},
	}
	return states
}

// Each method returns the next state
func (m *QuickWorkflow) Init(ctx context.Context) (*models.NextState, error) {
	slog.InfoContext(ctx, "Starting workflow")

	return &models.NextState{
		Name: StateGetIpData,
	}, nil
}

func (m *QuickWorkflow) StateGetIpData(ctx context.Context) (*models.NextState, error) {

	m.StateVariables[VAR_IP] = "127.0.0.1"

	return &models.NextState{
		Name: StateFinish,
	}, nil
}
