package engine

import (
	"fmt"
	"github.com/RealZimboGuy/gopherflow/internal/config"
	"github.com/RealZimboGuy/gopherflow/internal/domain"
	"github.com/RealZimboGuy/gopherflow/internal/models"
	"github.com/RealZimboGuy/gopherflow/internal/repository"
	"log/slog"
	"os"
	"reflect"
	"strings"
	"time"
)

var workflowQueue chan Workflow // Initialized in StartEngine using system setting

type WorkflowManager struct {
	WorkflowRegistry   *map[string]reflect.Type
	WorkflowRepo       *repository.WorkflowRepository
	WorkflowActionRepo *repository.WorkflowActionRepository
	executorRepo       *repository.ExecutorRepository
	DefinitionRepo     *repository.WorkflowDefinitionRepository
	executorID         int64
	wakeup             chan struct{}
}

// ListWorkflowDefinitions exposes repository list for web/API layers.
func (wm *WorkflowManager) ListWorkflowDefinitions() (*[]domain.WorkflowDefinition, error) {
	return wm.DefinitionRepo.FindAll()
}

// GetWorkflowDefinitionByName exposes repository get by name for web/API layers.
func (wm *WorkflowManager) GetWorkflowDefinitionByName(name string) (*domain.WorkflowDefinition, error) {
	return wm.DefinitionRepo.FindByName(name)
}

// ListExecutors returns recent executors ordered by last_active desc.
func (wm *WorkflowManager) ListExecutors(limit int) ([]*domain.Executor, error) {
	return wm.executorRepo.GetExecutorsByLastActive(limit)
}

// SearchWorkflows delegates to the repository to search based on request filters.
func (wm *WorkflowManager) SearchWorkflows(req models.SearchWorkflowRequest) (*[]domain.Workflow, error) {
	return wm.WorkflowRepo.SearchWorkflows(req)
}

// TopExecuting exposes repository method for dashboard
func (wm *WorkflowManager) TopExecuting(limit int) (*[]domain.Workflow, error) {
	return wm.WorkflowRepo.GetTopExecuting(limit)
}

// NextToExecute exposes repository method for dashboard
func (wm *WorkflowManager) NextToExecute(limit int) (*[]domain.Workflow, error) {
	return wm.WorkflowRepo.GetNextToExecute(limit)
}

// Overview exposes grouped counts for home dashboard
func (wm *WorkflowManager) Overview() ([]repository.WorkflowOverviewRow, error) {
	return wm.WorkflowRepo.GetWorkflowOverview()
}

// DefinitionOverview exposes counts by state for a workflow type
func (wm *WorkflowManager) DefinitionOverview(workflowType string) ([]repository.DefinitionStateRow, error) {
	return wm.WorkflowRepo.GetDefinitionStateOverview(workflowType)
}

func NewWorkflowManager(workflowRepo *repository.WorkflowRepository, workflowActionRepo *repository.WorkflowActionRepository, executorRepo *repository.ExecutorRepository,
	definitionRepo *repository.WorkflowDefinitionRepository, WorkflowRegistry *map[string]reflect.Type) *WorkflowManager {
	return &WorkflowManager{
		WorkflowRegistry:   WorkflowRegistry,
		WorkflowRepo:       workflowRepo,
		WorkflowActionRepo: workflowActionRepo,
		executorRepo:       executorRepo,
		DefinitionRepo:     definitionRepo,
		wakeup:             make(chan struct{}, 1),
	}
}

// StartEngine starts polling for new workflows at the given interval
func (wm *WorkflowManager) StartEngine(pollInterval time.Duration) {
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	// Register this executor instance
	registerExecutorInstance(wm)

	registerWorkflowDefinitions(wm)

	go startWorkflowRepairService(wm)

	// Initialize workflow queue size from system setting ENGINE_BATCH_SIZE
	queueSize := config.GetSystemSettingInteger(config.ENGINE_BATCH_SIZE)
	if queueSize <= 0 {
		queueSize = 10 // fallback default
	}
	workflowQueue = make(chan Workflow, queueSize)

	// log starting and number of workers
	slog.Info("Starting workflow engine", "workers", config.GetSystemSettingInteger(config.ENGINE_EXECUTOR_SIZE), "queue_size", queueSize)
	for i := 0; i < config.GetSystemSettingInteger(config.ENGINE_EXECUTOR_SIZE); i++ {
		go Worker(i, wm.executorID, *wm.WorkflowRepo, *wm.WorkflowActionRepo, workflowQueue)
	}

	slog.Info("Workflow engine started", "poll_interval", pollInterval.String())

	for {
		select {
		case <-ticker.C:
			wm.pollAndRunWorkflows()
		case <-wm.wakeup:
			wm.pollAndRunWorkflows()
		}

	}
}

// responsible for finding workflows that might have crashed half way and waking them up again
// these workflows will be in a state of SCHEDULED or EXECUTING and the executor will be last active more than 5 minutes ago
func startWorkflowRepairService(wm *WorkflowManager) {
	dur, _ := time.ParseDuration(config.GetSystemSettingString(config.ENGINE_STUCK_WORKFLOWS_INTERVAL))
	ticker := time.NewTicker(dur)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// Find stuck workflows and attempt to wake them up
			stuckWorkflows, err := wm.WorkflowRepo.FindStuckWorkflows(
				config.GetSystemSettingString(config.ENGINE_STUCK_WORKFLOWS_REPAIR_AFTER_MINUTES), config.GetSystemSettingString(config.ENGINE_EXECUTOR_GROUP),
				100)
			if err != nil {
				slog.Error("Error finding stuck workflows", "error", err)
				continue
			}
			for _, wf := range *stuckWorkflows {
				slog.Warn("Repairing stuck workflow", "workflow_id", wf.ID, "business_key", wf.BusinessKey, "Current State", wf.State, "Status", wf.Status)
				// Mark as scheduled and add to queue
				previousExecutorId := wf.ExecutorID
				exclusiveLock := wm.WorkflowRepo.ClearStateAndExecutorAndSetNextExecution(wf.ID, wf.Modified)
				if exclusiveLock {
					_, _ = wm.WorkflowActionRepo.Save(&domain.WorkflowAction{
						WorkflowID:     wf.ID,
						ExecutorID:     wm.executorID,
						ExecutionCount: 1,
						Type:           "REPAIRED",
						Name:           "REPAIRED",
						Text:           "Repaired and scheduled, previous executor was: " + fmt.Sprint(previousExecutorId),
						DateTime:       time.Now(),
					})
					instance, _ := createWorkflow(wm, wf.WorkflowType)
					ptr := instance.(Workflow)
					ptr.Setup(&wf)
					workflowQueue <- ptr
				}
			}
		}
	}

}

func registerWorkflowDefinitions(wm *WorkflowManager) {

	for name := range *wm.WorkflowRegistry {
		def, err := wm.DefinitionRepo.FindByName(name)
		if err != nil {
			// If not found, we'll create it; for other errors, log and continue
			slog.Warn("Workflow definition lookup error, will attempt create", "name", name, "error", err)
			def = nil
		}

		flow := buildFlowChart(wm, name)
		instance, _ := CreateWorkflowInstance(wm, name)
		desc := fmt.Sprintf("%s", instance.Description())

		if def == nil {
			// Create new definition
			def = &domain.WorkflowDefinition{
				Name:        name,
				Description: desc,
				Created:     time.Now(),
				Updated:     time.Now(),
				FlowChart:   flow,
			}
			slog.Info("Saving workflow definition", "name", name)
			if err := wm.DefinitionRepo.Save(def); err != nil {
				slog.Error("Failed to save workflow definition", "name", name, "error", err)
			}
			continue
		}

		// Update existing definition
		slog.Info("Updating workflow definition", "name", name)
		def.Description = desc
		def.Updated = time.Now()
		def.FlowChart = flow
		if err := wm.DefinitionRepo.Save(def); err != nil {
			slog.Error("Failed to update workflow definition", "name", name, "error", err)
		}
	}

}
func buildFlowChart(wm *WorkflowManager, name string) string {
	var sb strings.Builder

	// Modern class styles
	errorClass := "fill:#FF6B6B,stroke:#C53030,stroke-width:2px,color:#fff,stroke-dasharray: 4 2,rx:10,ry:10;"
	doneClass := "fill:#4ECDC4,stroke:#1F9C8C,stroke-width:2px,color:#fff,stroke-dasharray: 4 2,rx:10,ry:10;"
	startClass := "fill:#5568FE,stroke:#3346FF,stroke-width:2px,color:#fff,stroke-dasharray: 4 2,rx:10,ry:10;"
	manualClass := "fill:#FFD93D,stroke:#E6C200,stroke-width:2px,color:#333,stroke-dasharray: 4 2,rx:10,ry:10;"
	normalClass := "fill:#F0F4F8,stroke:#B0C4DE,stroke-width:1px,color:#333,rx:10,ry:10;"

	inst, err := createWorkflow(wm, name)
	if err != nil {
		return fmt.Sprintf("flowchart TD\n    %s[Uninitialized]\n", name)
	}
	wf, ok := inst.(Workflow)
	if !ok {
		return fmt.Sprintf("flowchart TD\n    %s[Invalid Workflow]\n", name)
	}

	states := wf.GetAllStates()
	transitions := wf.StateTransitions()
	//start := wf.InitialState()

	sb.WriteString("flowchart TD\n")

	// Build edges based on transitions (one-to-many)
	for from, tos := range transitions {
		for _, to := range tos {
			sb.WriteString(fmt.Sprintf("    %s --> %s\n", from, to))
		}
	}

	// classDefs
	sb.WriteString(fmt.Sprintf("    classDef errorClass %s\n", errorClass))
	sb.WriteString(fmt.Sprintf("    classDef doneClass %s\n", doneClass))
	sb.WriteString(fmt.Sprintf("    classDef startClass %s\n", startClass))
	sb.WriteString(fmt.Sprintf("    classDef manualClass %s\n", manualClass))
	sb.WriteString(fmt.Sprintf("    classDef normalClass %s\n", normalClass))

	// Assign classes based on state types
	for _, st := range states {
		switch st.StateType {
		case models.StateStart:
			sb.WriteString(fmt.Sprintf("    class %s startClass;\n", st.Name))
		case models.StateEnd:
			sb.WriteString(fmt.Sprintf("    class %s doneClass;\n", st.Name))
		case models.StateManual:
			sb.WriteString(fmt.Sprintf("    class %s manualClass;\n", st.Name))
		case models.StateError:
			sb.WriteString(fmt.Sprintf("    class %s errorClass;\n", st.Name))
		default:
			sb.WriteString(fmt.Sprintf("    class %s normalClass;\n", st.Name))
		}
	}

	return sb.String()
}

func registerExecutorInstance(wm *WorkflowManager) {
	name := config.GetSystemSettingString("EXECUTOR_NAME")
	if name == "" {
		hostname, err := os.Hostname()
		if err != nil {
			name = "workflow-engine"
		} else {
			name = hostname
		}
	}
	exec := &domain.Executor{Name: name, Started: time.Now(), LastActive: time.Now()}
	id, err := wm.executorRepo.Save(exec)
	if err != nil {
		slog.Error("Failed to register executor", "error", err)
	} else {
		wm.executorID = id
		slog.Info("Registered executor", "executor_id", id, "name", name)
		// Start heartbeat ticker to update last_active every 30s
		hb := time.NewTicker(30 * time.Second)
		go func(executorID int64) {
			for range hb.C {
				if err := wm.executorRepo.UpdateLastActive(executorID, time.Now()); err != nil {
					slog.Error("Failed to update executor last_active", "executor_id", executorID, "error", err)
				} else {
					slog.Debug("Updated executor last_active", "executor_id", executorID)
				}
			}
		}(id)
	}
}

// pollAndRunWorkflows queries the repository for new workflows and runs them
func (wm *WorkflowManager) pollAndRunWorkflows() {

	slog.Debug("Polling for new workflows")
	workflows, err := wm.WorkflowRepo.FindPendingWorkflows(
		config.GetSystemSettingInteger(config.ENGINE_BATCH_SIZE),
		config.GetSystemSettingString(config.ENGINE_EXECUTOR_GROUP),
	)
	if err != nil {
		slog.Error("Error fetching workflows", "error", err)
		return
	}

	for _, wf := range *workflows {

		// first we mark the workflow as running
		slog.Info("Marking workflow as scheduled for execution", "business_key", wf.BusinessKey)
		exclusiveLock := wm.WorkflowRepo.MarkWorkflowAsScheduledForExecution(wf.ID, wm.executorID, wf.Modified)

		if exclusiveLock == false {
			slog.Info("Workflow already running unable to gain lock", "business_key", wf.BusinessKey, "externalId", wf.ExternalID)
			_, _ = wm.WorkflowActionRepo.Save(&domain.WorkflowAction{WorkflowID: wf.ID, ExecutorID: wm.executorID, ExecutionCount: 1, Type: "LOCK_FAILED", Name: "LOCK_FAILED", Text: "Failed to Acquier a lock on the workflow", DateTime: time.Now()})
			continue
		}
		_, _ = wm.WorkflowActionRepo.Save(&domain.WorkflowAction{WorkflowID: wf.ID, ExecutorID: wm.executorID, ExecutionCount: 1, Type: "SCHEDULED", Name: "SCHEDULED", Text: "Scheduled for Execution", DateTime: time.Now()})

		// create an instance of the workflow based on the type
		instance, _ := createWorkflow(wm, wf.WorkflowType)

		slog.Info("Add workflow to execution channel", "business_key", wf.BusinessKey)
		ptr := instance.(Workflow)
		ptr.Setup(&wf)
		workflowQueue <- ptr

		slog.Info("Running workflow", "business_key", wf.BusinessKey)
		// RunWorkflow(wf) // call your workflow runner here
	}

}

func createWorkflow(wm *WorkflowManager, name string) (interface{}, error) {
	t, ok := (*wm.WorkflowRegistry)[name]
	if !ok {
		return nil, fmt.Errorf("workflow not found: %s", name)
	}
	return reflect.New(t).Interface(), nil
}

func CreateWorkflowInstance(wm *WorkflowManager, name string) (Workflow, error) {
	inst, err := createWorkflow(wm, name)
	if err != nil {
		return nil, err
	}
	wf, ok := inst.(Workflow)
	if !ok {
		return nil, fmt.Errorf("workflow type does not implement engine.Workflow: %s", name)
	}
	return wf, nil
}

func (wm *WorkflowManager) Wakeup() {
	slog.Info("Wakeup Manager called")
	select {
	case wm.wakeup <- struct{}{}:
	default:
	}
}
