package workflows

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/RealZimboGuy/gopherflow/pkg/gopherflow/core"
	domain "github.com/RealZimboGuy/gopherflow/pkg/gopherflow/domain"
	"github.com/RealZimboGuy/gopherflow/pkg/gopherflow/models"
	"github.com/RealZimboGuy/gopherflow/pkg/gopherflow/workflow_helpers"
)

// Define a named string type
var StateFinish string = "Finish"
var StateInit string = "Init"
var StateReview string = "Review"
var StateApprove string = "Approve"
var StateApproveError string = "ApproveError"

const VAR_NAME = "name"
const VAR_AGE = "age"

type DemoWorkflow struct {
	core.BaseWorkflow
	Clock core.Clock
}

type EnrollmentStruct struct {
	Name string `json:"name"`
	Age  int    `json:"age"`
}

func (m *DemoWorkflow) Setup(wf *domain.Workflow) {
	m.BaseWorkflow.Setup(wf)
}
func (m *DemoWorkflow) GetWorkflowData() *domain.Workflow {
	return m.WorkflowState
}
func (m *DemoWorkflow) GetStateVariables() map[string]string {
	return m.StateVariables
}
func (m *DemoWorkflow) InitialState() string {
	return StateInit
}

func (m *DemoWorkflow) Description() string {
	return "This is a Demo Workflow showing how it can be used"
}

func (m *DemoWorkflow) GetRetryConfig() models.RetryConfig {
	return models.RetryConfig{
		MaxRetryCount:    10,
		RetryIntervalMin: time.Second * 10,
		RetryIntervalMax: time.Minute * 60,
	}
}

func (m *DemoWorkflow) StateTransitions() map[string][]string {
	return map[string][]string{
		StateInit:    []string{StateReview},                     // Init -> review
		StateReview:  []string{StateApprove, StateApproveError}, // review -> approve OR approve error
		StateApprove: []string{StateFinish},                     // approve -> finish
	}
}
func (m *DemoWorkflow) GetAllStates() []models.WorkflowState {
	states := []models.WorkflowState{
		{Name: StateInit, StateType: models.StateStart},
		{Name: StateReview, StateType: models.StateNormal},
		{Name: StateApprove, StateType: models.StateNormal},
		{Name: StateApproveError, StateType: models.StateError},
		{Name: StateFinish, StateType: models.StateEnd},
	}
	return states
}

// Each method returns the next state
func (m *DemoWorkflow) Init(ctx context.Context) (*models.NextState, error) {
	slog.Info("Starting workflow")
	m.StateVariables[VAR_AGE] = "33"
	m.StateVariables[VAR_NAME] = "Julian"

	enrollment := EnrollmentStruct{
		Name: "Julian",
		Age:  33,
	}
	workflow_helpers.SaveStructToStateVars(m.StateVariables, "enrollment", enrollment)

	return &models.NextState{
		Name:                StateReview,
		NextExecutionOffset: "300 seconds",
	}, nil
}

func (m *DemoWorkflow) Review(ctx context.Context) (*models.NextState, error) {
	slog.Info("Reviewing workflow, changing name")
	m.StateVariables[VAR_NAME] = "Julian2"
	//return an error
	//return nil, fmt.Errorf("shit gone fubar")

	loaded, _ := workflow_helpers.LoadStructFromStateVars[EnrollmentStruct](m.StateVariables, "enrollment")
	fmt.Println("Loaded enrollment struct", loaded)

	//block for 20 seconds
	fmt.Println("Sleeping for 200 seconds")
	m.Clock.Sleep(200 * time.Second)

	//wait 30 seconds
	return &models.NextState{
		Name:          StateApprove,
		NextExecution: time.Now().Add(30 * time.Second),
	}, nil
}

func (m *DemoWorkflow) Approve(ctx context.Context) (*models.NextState, error) {
	slog.Info("Approving workflow")
	// print the name state var
	slog.Info("State Variables", "vars", m.StateVariables)
	return &models.NextState{
		Name: StateFinish,
	}, nil
}
