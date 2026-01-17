package controllers

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/RealZimboGuy/gopherflow/internal/engine"
	"github.com/RealZimboGuy/gopherflow/internal/repository"
	"github.com/RealZimboGuy/gopherflow/pkg/gopherflow/core"
	"github.com/RealZimboGuy/gopherflow/pkg/gopherflow/domain"
	"github.com/RealZimboGuy/gopherflow/pkg/gopherflow/models"

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

func (c *WorkflowsController) handleGetWorkflowById(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	idStr := r.PathValue("id")
	if idStr == "" {
		http.Error(w, "id is required", http.StatusBadRequest)
		return
	}

	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "id is an integer", http.StatusBadRequest)
		return
	}

	id64 := int64(id)
	result, err := c.WorkflowManager.WorkflowRepo.FindByID(id64)
	if err != nil {
		http.Error(w, "workflow not found", http.StatusNotFound)
		return
	}
	apiResult := mapWorkflowToApiWorkflow(result, id64)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(apiResult)

}

func (c *WorkflowsController) handleGetWorkflowByExternalId(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	externalId := r.PathValue("externalId")
	if externalId == "" {
		http.Error(w, "externalId is required", http.StatusBadRequest)
		return
	}

	result, err := c.WorkflowRepo.FindByExternalId(externalId)
	if err != nil || result == nil {
		http.Error(w, "workflow not found", http.StatusNotFound)
		return
	}
	apiResult := mapWorkflowToApiWorkflow(result, result.ID)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(apiResult)
}

func (c *WorkflowsController) handleCreateWorkflow(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var req models.CreateWorkflowRequest
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		http.Error(w, "invalid JSON payload", http.StatusBadRequest)
		return
	}

	err := validateCreateWorkflow(r.Context(), req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err, id := createWorkflow(r.Context(), c, req)

	if err != nil {
		slog.Error("Failed to save workflow", "error", err)
		http.Error(w, "failed to create workflow", http.StatusInternalServerError)
		return
	}

	c.WorkflowManager.Wakeup()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(models.CreateWorkflowResponse{ID: id})
}

func validateCreateWorkflow(ctx context.Context, req models.CreateWorkflowRequest) error {
	// Validate required fields
	if req.ExternalID == "" || req.ExecutorGroup == "" || req.WorkflowType == "" || req.BusinessKey == "" {
		return errors.New("externalId, executorGroup, workflowType and businessKey are required")
	}
	return nil
}

func createWorkflow(ctx context.Context, c *WorkflowsController, req models.CreateWorkflowRequest) (error, int64) {
	// Validate workflow type exists via engine registry and get initial stateA

	slog.InfoContext(ctx, "Creating workflow", "externalId", req.ExternalID, "businessKey", req.BusinessKey, "workflowType", req.WorkflowType)

	//add the username of the creating user to the workflow statevars
	if userName := ctx.Value(core.CtxKeyUsername); userName != nil {
		if s, ok := userName.(string); ok && s != "" {
			if req.StateVars == nil {
				req.StateVars = make(map[string]string)
			}
			req.StateVars["createdBy"] = s
		}
	}

	wfInstance, err := engine.CreateWorkflowInstance(c.WorkflowManager, req.WorkflowType)
	if err != nil {
		return err, 0
	}
	initialState := wfInstance.InitialState()

	//if the external id is a duplicate, we return the existing workflow
	existing, _ := c.WorkflowRepo.FindByExternalId(req.ExternalID)
	if existing != nil {
		slog.WarnContext(ctx, "Workflow already exists", "externalId", req.ExternalID)
		return nil, existing.ID
	}

	// Serialize state vars
	var stateVarsJSON string
	if req.StateVars != nil {
		b, err := json.Marshal(req.StateVars)
		if err != nil {
			return err, 0
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
	return err, id
}

func (c *WorkflowsController) handleCreateAndWaitWorkflow(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var req models.CreateAndWaitRequest
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		http.Error(w, "invalid JSON payload", http.StatusBadRequest)
		return
	}
	err := validateCreateWorkflow(r.Context(), req.CreateWorkflowRequest)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	//enforce a mimimum of the check and wait seconds
	if req.CheckSeconds < 1 {
		req.CheckSeconds = 1
	}
	if req.WaitSeconds < 1 {
		req.WaitSeconds = 1
	}

	err, id := createWorkflow(r.Context(), c, req.CreateWorkflowRequest)
	c.WorkflowManager.Wakeup()

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(req.WaitSeconds)*time.Second)
	defer cancel()
	ticker := time.NewTicker(time.Duration(req.CheckSeconds) * time.Second) // check every
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			// Timeout reached
			http.Error(w, "timeout waiting for workflow result", http.StatusGatewayTimeout)
			return
		case <-ticker.C:
			// Try to fetch workflow result by ID
			result, err := c.WorkflowManager.WorkflowRepo.FindByID(id)
			if err == nil {
				if len(req.WaitForStates) == 0 || contains(req.WaitForStates, result.State) {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusOK)
					apiResult := mapWorkflowToApiWorkflow(result, id)
					json.NewEncoder(w).Encode(apiResult)
					return
				}
			}
		}
	}
}

func mapWorkflowToApiWorkflow(result *domain.Workflow, id int64) models.WorkflowApiResponse {
	stateVars := make(map[string]string)
	if result.StateVars.Valid && len(result.StateVars.String) > 0 {
		if err := json.Unmarshal([]byte(result.StateVars.String), &stateVars); err != nil {
			slog.Warn("Failed to parse state vars", "id", id, "error", err)
		}
	}
	apiResult := models.WorkflowApiResponse{
		ID:             result.ID,
		Status:         result.Status,
		ExecutionCount: result.ExecutionCount,
		RetryCount:     result.RetryCount,
		Created:        result.Created,
		Modified:       result.Modified,
		NextActivation: func() time.Time {
			if result.NextActivation.Valid {
				return result.NextActivation.Time
			}
			return time.Time{}
		}(),
		Started: func() time.Time {
			if result.Started.Valid {
				return result.Started.Time
			}
			return time.Time{}
		}(),
		ExecutorID: func() string {
			if result.ExecutorID.Valid {
				return result.ExecutorID.String
			}
			return ""
		}(),
		ExecutorGroup: result.ExecutorGroup,
		WorkflowType:  result.WorkflowType,
		ExternalID:    result.ExternalID,
		BusinessKey:   result.BusinessKey,
		State:         result.State,
		StateVars:     stateVars,
	}
	return apiResult
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
		w.WriteHeader(http.StatusOK)
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
	var wf *domain.Workflow
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err == nil {
		wf, _ = c.WorkflowRepo.FindByID(id)
	}

	// If not found by numeric ID, try as external ID
	if wf == nil {
		wf, _ = c.WorkflowRepo.FindByExternalId(idStr)
	}
	if wf == nil {
		http.Error(w, "workflow not found", http.StatusNotFound)
		return
	}
	var req models.UpdateWorkflowStateRequest
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
	// Acquire lock via LockWorkflowByModified with current modified
	// We first update next activation to now (or provided) and set IN_PROGRESS status atomically guarding by modified
	next := time.Now()
	if req.NextActivation != nil {
		next = *req.NextActivation
	}
	locked := c.WorkflowRepo.LockWorkflowByModified(wf.ID, wf.Modified)
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
	json.NewEncoder(w).Encode(models.UpdateWorkflowStateResponse{OK: true})
}

// handleUpdateWorkflowState updates the workflow's state and optionally next activation, with optimistic lock semantics
func (c *WorkflowsController) handleUpdateWorkflowStateAndWait(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	idStr := r.PathValue("id")
	if idStr == "" {
		http.Error(w, "id is required", http.StatusBadRequest)
		return
	}
	// Try to parse id as int64
	var wf *domain.Workflow
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err == nil {
		wf, _ = c.WorkflowRepo.FindByID(id)
	}

	// If not found by numeric ID, try as external ID
	if wf == nil {
		wf, _ = c.WorkflowRepo.FindByExternalId(idStr)
	}
	if wf == nil {
		http.Error(w, "workflow not found", http.StatusNotFound)
		return
	}

	var req models.UpdateWorkflowStateAndWaitRequest
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		http.Error(w, "invalid JSON payload", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(req.UpdateWorkflowStateRequest.State) == "" {
		http.Error(w, "state is required", http.StatusBadRequest)
		return
	}
	//check the from state
	if len(req.FromStates) > 0 && !contains(req.FromStates, wf.State) {
		http.Error(w, fmt.Sprintf("current State: %s is not in the expected from states: %s", wf.State, req.FromStates), http.StatusBadRequest)
		return
	}

	// Acquire lock via LockWorkflowByModified with current modified
	// We first update next activation to now (or provided) and set IN_PROGRESS status atomically guarding by modified
	next := time.Now()
	if req.UpdateWorkflowStateRequest.NextActivation != nil {
		next = *req.UpdateWorkflowStateRequest.NextActivation
	}
	locked := c.WorkflowRepo.LockWorkflowByModified(wf.ID, wf.Modified)
	if !locked {
		http.Error(w, "unable to acquire lock; workflow busy", http.StatusConflict)
		return
	}
	// Set new state and desired next activation
	if err := c.WorkflowRepo.UpdateState(wf.ID, req.UpdateWorkflowStateRequest.State); err != nil {
		slog.Error("UpdateState failed", "error", err)
		http.Error(w, "failed to update state", http.StatusInternalServerError)
		return
	}
	//add a log action
	_, _ = c.WorkflowActionRepo.Save(&domain.WorkflowAction{WorkflowID: wf.ID, ExecutorID: 0, ExecutionCount: wf.RetryCount, Type: "LOG", Name: wf.State, Text: "User Manually Changed State :" + req.UpdateWorkflowStateRequest.State, DateTime: time.Now()})

	if req.UpdateStateVarRequest.Key != "" {
		// Parse current state vars JSON to map
		vars := map[string]string{}
		if wf.StateVars.Valid && wf.StateVars.String != "" {
			_ = json.Unmarshal([]byte(wf.StateVars.String), &vars)
		}
		vars[req.UpdateStateVarRequest.Key] = req.UpdateStateVarRequest.Value
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
		_, _ = c.WorkflowActionRepo.Save(&domain.WorkflowAction{WorkflowID: wf.ID, ExecutorID: 0, ExecutionCount: wf.RetryCount, Type: "LOG", Name: wf.State, Text: "Updated state var: " + req.UpdateStateVarRequest.Key, DateTime: time.Now()})
	}

	if err := c.WorkflowRepo.UpdateNextActivationSpecific(wf.ID, next); err != nil {
		slog.Error("UpdateNextActivationSpecific failed", "error", err)
		http.Error(w, "failed to update next activation", http.StatusInternalServerError)
		return
	}
	c.WorkflowManager.Wakeup()

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(req.WaitSeconds)*time.Second)
	defer cancel()
	ticker := time.NewTicker(time.Duration(req.CheckSeconds) * time.Second) // check every
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			// Timeout reached
			http.Error(w, "timeout waiting for workflow result", http.StatusGatewayTimeout)
			return
		case <-ticker.C:
			// Try to fetch workflow result by ID
			result, err := c.WorkflowManager.WorkflowRepo.FindByID(wf.ID)
			if err == nil {
				if len(req.WaitForStates) == 0 || contains(req.WaitForStates, result.State) {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusOK)
					apiResult := mapWorkflowToApiWorkflow(result, wf.ID)
					json.NewEncoder(w).Encode(apiResult)
					return
				}
			}
		}
	}
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
	var wf *domain.Workflow
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err == nil {
		wf, _ = c.WorkflowRepo.FindByID(id)
	}

	// If not found by numeric ID, try as external ID
	if wf == nil {
		wf, _ = c.WorkflowRepo.FindByExternalId(idStr)
	}
	if wf == nil {
		http.Error(w, "workflow not found", http.StatusNotFound)
		return
	}
	var req models.UpdateStateVarRequest
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
	json.NewEncoder(w).Encode(models.UpdateStateVarResponse{OK: true})
}

func parseInt64(s string) int64 {
	v, _ := strconv.ParseInt(s, 10, 64)
	return v
}

// handleListWorkflowDefinitions returns a list of all workflow definitions
func (c *WorkflowsController) handleListWorkflowDefinitions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	defs, err := c.WorkflowManager.ListWorkflowDefinitions()
	if err != nil {
		slog.Error("Failed to list workflow definitions", "error", err)
		http.Error(w, "Failed to load definitions", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(defs)
}

// handleGetWorkflowDefinitionByName returns a specific workflow definition by name
func (c *WorkflowsController) handleGetWorkflowDefinitionByName(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	name := r.PathValue("name")
	if name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}

	def, err := c.WorkflowManager.GetWorkflowDefinitionByName(name)
	if err != nil {
		slog.Error("Failed to get workflow definition", "name", name, "error", err)
		http.Error(w, "Definition not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(def)
}

func contains(arr []string, val string) bool {
	for _, item := range arr {
		if item == val {
			return true
		}
	}
	return false
}
