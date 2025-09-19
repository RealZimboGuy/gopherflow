package controllers

import (
	"database/sql"
	"encoding/json"
	"gopherflow/internal/domain"
	"gopherflow/internal/engine"
	"gopherflow/internal/models"
	"gopherflow/internal/repository"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// WorkflowsController holds dependencies for workflow HTTP endpoints.
type WorkflowsController struct {
	AuthController
	WorkflowRepo       *repository.WorkflowRepository
	WorkflowActionRepo *repository.WorkflowActionRepository
	WorkflowManager    *engine.WorkflowManager
}

func NewWorkflowsController(workflowRepo *repository.WorkflowRepository, workflowActionsRepo *repository.WorkflowActionRepository, workflowManager *engine.WorkflowManager,
	userRepo *repository.UserRepository) *WorkflowsController {
	return &WorkflowsController{WorkflowRepo: workflowRepo, WorkflowActionRepo: workflowActionsRepo, WorkflowManager: workflowManager, AuthController: AuthController{
		UserRepo: userRepo,
	}}
}

// createWorkflowRequest is the payload for creating a workflow.
type createWorkflowRequest struct {
	ExternalID    string            `json:"externalId"`
	ExecutorGroup string            `json:"executorGroup"`
	WorkflowType  string            `json:"workflowType"`
	BusinessKey   string            `json:"businessKey"`
	StateVars     map[string]string `json:"stateVars"`
	// Optional scheduling inputs
	NextActivation       *time.Time `json:"nextActivation,omitempty"`
	NextActivationOffset string     `json:"nextActivationOffset,omitempty"`
}

// createWorkflowResponse is returned on successful creation.
type createWorkflowResponse struct {
	ID int64 `json:"id"`
}

type updateWorkflowStateRequest struct {
	State          string     `json:"state"`
	NextActivation *time.Time `json:"nextActivation,omitempty"`
}

type updateWorkflowStateResponse struct {
	OK bool `json:"ok"`
}

type updateStateVarRequest struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type updateStateVarResponse struct {
	OK bool `json:"ok"`
}

func (c *WorkflowsController) handleCreateWorkflow(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var req createWorkflowRequest
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		http.Error(w, "invalid JSON payload", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.ExternalID == "" || req.ExecutorGroup == "" || req.WorkflowType == "" || req.BusinessKey == "" {
		http.Error(w, "externalId, executorGroup, workflowType and businessKey are required", http.StatusBadRequest)
		return
	}

	// Validate workflow type exists via engine registry and get initial state
	wfInstance, err := engine.CreateWorkflowInstance(req.WorkflowType)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	initialState := wfInstance.InitialState()

	//if the external id is a duplicate, we return the existing workflow
	existing, _ := c.WorkflowRepo.FindByExternalId(req.ExternalID)
	if existing != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(createWorkflowResponse{ID: existing.ID})
		return
	}

	// Serialize state vars
	var stateVarsJSON string
	if req.StateVars != nil {
		b, err := json.Marshal(req.StateVars)
		if err != nil {
			http.Error(w, "invalid stateVars", http.StatusBadRequest)
			return
		}
		stateVarsJSON = string(b)
	}

	now := time.Now().UTC()
	var nextActivation time.Time
	if req.NextActivation != nil {
		nextActivation = *req.NextActivation
	} else if req.NextActivationOffset != "" {
		// We cannot compute SQL interval here precisely like DB; schedule approximately using Go clock.
		// For simplicity, set to now (engine can reschedule later if workflows use offsets).
		nextActivation = now
	} else {
		// default to NOW if not specified
		nextActivation = now
	}

	wf := &domain.Workflow{
		Status:         "NEW",
		ExecutionCount: 0,
		RetryCount:     0,
		Created:        now,
		Modified:       now,
		NextActivation: sql.NullTime{Time: nextActivation, Valid: true},
		Started:        sql.NullTime{},
		ExecutorGroup:  req.ExecutorGroup,
		WorkflowType:   req.WorkflowType,
		ExternalID:     req.ExternalID,
		BusinessKey:    req.BusinessKey,
		State:          initialState,
	}
	if stateVarsJSON != "" {
		wf.StateVars.String = stateVarsJSON
		wf.StateVars.Valid = true
	}

	id, err := c.WorkflowRepo.Save(wf)
	if err != nil {
		slog.Error("Failed to save workflow", "error", err)
		http.Error(w, "failed to create workflow", http.StatusInternalServerError)
		return
	}

	c.WorkflowManager.Wakeup()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(createWorkflowResponse{ID: id})
}
func (c *WorkflowsController) handleSearchWorkflows(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var req models.SearchWorkflowRequest
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		slog.Error("Failed to decode request", "error", err)
		http.Error(w, "invalid JSON payload", http.StatusBadRequest)
		return
	}

	//max of 1000 results is allowed
	if req.Limit > 1000 {
		slog.Warn("limit cannot be greater than 1000")
		http.Error(w, "limit cannot be greater than 1000", http.StatusBadRequest)
		return
	}

	//if the external id is a duplicate, we return the existing workflow
	results, err := c.WorkflowRepo.SearchWorkflows(req)
	if err != nil {
		slog.Error("Failed to search workflows", "error", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(results)
		return
	}
	if results != nil {
		searchResponse := models.SearchWorkflowResponse{
			Results:   len(*results),
			Offset:    req.Offset,
			Workflows: *results,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(searchResponse)
		return
	}

}

// handleUpdateWorkflowState updates the workflow's state and optionally next activation, with optimistic lock semantics
func (c *WorkflowsController) handleUpdateWorkflowState(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	idStr := r.PathValue("id")
	if idStr == "" {
		http.Error(w, "id is required", http.StatusBadRequest)
		return
	}
	wf, err := c.WorkflowRepo.FindByID(parseInt64(idStr))
	if err != nil || wf == nil {
		http.Error(w, "workflow not found", http.StatusNotFound)
		return
	}
	var req updateWorkflowStateRequest
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		http.Error(w, "invalid JSON payload", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(req.State) == "" {
		http.Error(w, "state is required", http.StatusBadRequest)
		return
	}
	// Acquire lock via ClearStateAndExecutorAndSetNextExecution with current modified
	// We first update next activation to now (or provided) and set IN_PROGRESS status atomically guarding by modified
	next := time.Now()
	if req.NextActivation != nil {
		next = *req.NextActivation
	}
	locked := c.WorkflowRepo.ClearStateAndExecutorAndSetNextExecution(wf.ID, wf.Modified)
	if !locked {
		http.Error(w, "unable to acquire lock; workflow busy", http.StatusConflict)
		return
	}
	// Set new state and desired next activation
	if err := c.WorkflowRepo.UpdateState(wf.ID, req.State); err != nil {
		slog.Error("UpdateState failed", "error", err)
		http.Error(w, "failed to update state", http.StatusInternalServerError)
		return
	}
	//add a log action
	_, _ = c.WorkflowActionRepo.Save(&domain.WorkflowAction{WorkflowID: wf.ID, ExecutorID: 0, ExecutionCount: wf.RetryCount, Type: "LOG", Name: wf.State, Text: "User Manually Changed State :" + req.State, DateTime: time.Now()})

	if err := c.WorkflowRepo.UpdateNextActivationSpecific(wf.ID, next); err != nil {
		slog.Error("UpdateNextActivationSpecific failed", "error", err)
		http.Error(w, "failed to update next activation", http.StatusInternalServerError)
		return
	}
	c.WorkflowManager.Wakeup()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(updateWorkflowStateResponse{OK: true})
}

// handleUpdateStateVar upserts a single state var key/value; only modified date should change; action created.
func (c *WorkflowsController) handleUpdateStateVar(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	idStr := r.PathValue("id")
	if idStr == "" {
		http.Error(w, "id is required", http.StatusBadRequest)
		return
	}
	wf, err := c.WorkflowRepo.FindByID(parseInt64(idStr))
	if err != nil || wf == nil {
		http.Error(w, "workflow not found", http.StatusNotFound)
		return
	}
	var req updateStateVarRequest
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		http.Error(w, "invalid JSON payload", http.StatusBadRequest)
		return
	}
	key := strings.TrimSpace(req.Key)
	if key == "" {
		http.Error(w, "key is required", http.StatusBadRequest)
		return
	}
	// Parse current state vars JSON to map
	vars := map[string]string{}
	if wf.StateVars.Valid && wf.StateVars.String != "" {
		_ = json.Unmarshal([]byte(wf.StateVars.String), &vars)
	}
	vars[key] = req.Value
	b, err := json.Marshal(vars)
	if err != nil {
		http.Error(w, "failed to serialize state vars", http.StatusInternalServerError)
		return
	}
	if err := c.WorkflowRepo.SaveWorkflowVariablesAndTouch(wf.ID, string(b)); err != nil {
		slog.Error("SaveWorkflowVariablesAndTouch failed", "error", err)
		http.Error(w, "failed to update state var", http.StatusInternalServerError)
		return
	}
	// Record action indicating which state var was updated
	_, _ = c.WorkflowActionRepo.Save(&domain.WorkflowAction{WorkflowID: wf.ID, ExecutorID: 0, ExecutionCount: wf.RetryCount, Type: "LOG", Name: wf.State, Text: "Updated state var: " + key, DateTime: time.Now()})
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(updateStateVarResponse{OK: true})
}

func parseInt64(s string) int64 {
	v, _ := strconv.ParseInt(s, 10, 64)
	return v
}
