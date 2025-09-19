package workflows

import (
	"gopherflow/internal/core"
	"gopherflow/internal/domain"
	"gopherflow/internal/models"
	"io"
	"log/slog"
	"net/http"
	"time"
)

// Define a named string type
var StateGetIpData string = "StateGetIpData"

const VAR_IP = "ip"

type GetIpWorkflow struct {
	core.BaseWorkflow
}

func (m *GetIpWorkflow) Setup(wf *domain.Workflow) {
	m.BaseWorkflow.Setup(wf)
}
func (m *GetIpWorkflow) GetWorkflowData() *domain.Workflow {
	return m.WorkflowState
}
func (m *GetIpWorkflow) GetStateVariables() map[string]string {
	return m.StateVariables
}
func (m *GetIpWorkflow) InitialState() string {
	return StateInit
}

func (m *GetIpWorkflow) Description() string {
	return "This is a Demo Workflow showing how it can be used"
}

func (m *GetIpWorkflow) GetRetryConfig() models.RetryConfig {
	return models.RetryConfig{
		MaxRetryCount:    10,
		RetryIntervalMin: time.Second * 10,
		RetryIntervalMax: time.Minute * 60,
	}
}

func (m *GetIpWorkflow) StateTransitions() map[string][]string {
	return map[string][]string{
		StateInit:      []string{StateGetIpData}, // Init -> StateGetIpData
		StateGetIpData: []string{StateFinish},    // StateGetIpData -> finish
	}
}
func (m *GetIpWorkflow) GetAllStates() []models.WorkflowState {
	states := []models.WorkflowState{
		{Name: StateInit, StateType: models.StateStart},
		{Name: StateGetIpData, StateType: models.StateNormal},
		{Name: StateFinish, StateType: models.StateEnd},
	}
	return states
}

// Each method returns the next state
func (m *GetIpWorkflow) Init() (*models.NextState, error) {
	slog.Info("Starting workflow")

	return &models.NextState{
		Name: StateGetIpData,
	}, nil
}

func (m *GetIpWorkflow) StateGetIpData() (*models.NextState, error) {
	resp, err := http.Get("http://ifconfig.io")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	ipBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	ip := string(ipBytes)
	m.StateVariables[VAR_IP] = ip

	return &models.NextState{
		Name: StateFinish,
	}, nil
}
