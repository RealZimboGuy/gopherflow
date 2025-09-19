package engine

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"reflect"
	"time"

	"github.com/RealZimboGuy/gopherflow/internal/repository"
	"github.com/RealZimboGuy/gopherflow/pkg/gopherflow/domain"
	models "github.com/RealZimboGuy/gopherflow/pkg/gopherflow/models"
)

// Engine runs a workflow
func RunWorkflow(w Workflow, r repository.WorkflowRepository, wa repository.WorkflowActionRepository, executorID int64, workerID string) {

	slog.Info("Running workflow", "workflow_id", w.GetWorkflowData().ID, "worker_id", workerID)
	err := r.UpdateWorkflowStatus(w.GetWorkflowData().ID, "EXECUTING")
	_, _ = wa.Save(&domain.WorkflowAction{WorkflowID: w.GetWorkflowData().ID, ExecutorID: executorID, ExecutionCount: w.GetWorkflowData().ExecutionCount, Type: "EXECUTING", Name: "EXECUTING", Text: "EXECUTING", DateTime: time.Now()})

	if err != nil {
		slog.Error("Error updating workflow status", "error", err, "worker_id", workerID)
		return
	}

	stateMap := w.StateTransitions()

	//the database determines where we are and start at
	currentState := w.GetWorkflowData().State

	//if we are on the starting state then update the starting time
	if currentState == w.InitialState() {
		err := r.UpdateWorkflowStartingTime(w.GetWorkflowData().ID)
		_, _ = wa.Save(&domain.WorkflowAction{WorkflowID: w.GetWorkflowData().ID, ExecutorID: executorID, ExecutionCount: w.GetWorkflowData().ExecutionCount, Type: "STARTING", Name: "EXECUTING", Text: "Starting Workflow", DateTime: time.Now()})
		if err != nil {
			slog.Error("Error updating workflow starting time", "error", err, "worker_id", workerID)
			return
		}
	}

	val := reflect.ValueOf(w)

	for {

		isEndState := false
		for _, state := range w.GetAllStates() {
			if state.Name == currentState && (state.StateType == models.StateEnd ||
				state.StateType == models.StateManual ||
				state.StateType == models.StateError) {
				isEndState = true
				break
			}
		}
		if isEndState {
			if processWorflowCompleted(w, r, wa, executorID, workerID, currentState) {
				return
			}
			break
		}

		method := val.MethodByName(currentState)
		if !method.IsValid() {
			panic(fmt.Sprintf("method %s not found", currentState))
		}

		// Call the method and get the next state
		results := method.Call(nil)
		if len(results) != 2 || !(results[0].Type().AssignableTo(reflect.TypeOf(models.NextState{})) || results[0].Type().AssignableTo(reflect.TypeOf(&models.NextState{}))) {
			panic(fmt.Sprintf("method %s should return (NextState or *NextState, error)", currentState))
		}

		ns, ok := results[0].Interface().(*models.NextState)
		if !ok {
			panic(fmt.Sprintf("method %s did not return a NextState as first value", currentState))
		}
		// Second return value = error
		var callErr error
		if !results[1].IsNil() {
			callErr = results[1].Interface().(error)
		}
		if callErr != nil {
			processStateExecutionError(w, r, wa, executorID, workerID, currentState, callErr)
			return
		}

		nextState := ns.Name
		// Validate if the transition is allowed (one-to-many)
		if nextState != "END" { // keep legacy allowance for literal END if ever used
			allowedList, ok := stateMap[currentState]
			if !ok {
				_, _ = wa.Save(&domain.WorkflowAction{WorkflowID: w.GetWorkflowData().ID, ExecutorID: executorID, ExecutionCount: w.GetWorkflowData().RetryCount, Type: "ERROR", Name: "Invalid Transition", Text: "no transitions defined for current state", DateTime: time.Now()})
				panic(fmt.Sprintf("invalid state transition from %s to %s (no transitions)", currentState, nextState))
			}
			valid := false
			for _, t := range allowedList {
				if t == nextState {
					valid = true
					break
				}
			}
			if !valid {
				_, _ = wa.Save(&domain.WorkflowAction{WorkflowID: w.GetWorkflowData().ID, ExecutorID: executorID, ExecutionCount: w.GetWorkflowData().RetryCount, Type: "ERROR", Name: "Invalid Transition", Text: "transition is not allowed", DateTime: time.Now()})
				panic(fmt.Sprintf("invalid state transition from %s to %s", currentState, nextState))
			}
		}

		slog.Info("Transitioning state", "from", currentState, "to", nextState, "worker_id", workerID)
		_, _ = wa.Save(&domain.WorkflowAction{WorkflowID: w.GetWorkflowData().ID, ExecutorID: executorID, ExecutionCount: w.GetWorkflowData().RetryCount, Type: "TRANSITION", Name: currentState, Text: "From " + currentState + " to " + nextState, DateTime: time.Now()})
		currentState = nextState

		slog.Info("Updating workflow state", "workflow_id", w.GetWorkflowData().ID, "state", currentState, "worker_id", workerID)
		//this also resets the retry count
		err := r.UpdateState(w.GetWorkflowData().ID, currentState)
		if err != nil {
			return
		}

		if compareAndSaveWorkflowStateVars(w, r, workerID) {
			return
		}

		nextStateObject := results[0].Interface().(*models.NextState)

		if nextStateObject.ActionLog != "" {
			_, _ = wa.Save(&domain.WorkflowAction{WorkflowID: w.GetWorkflowData().ID, ExecutorID: executorID, ExecutionCount: w.GetWorkflowData().RetryCount, Type: "LOG", Name: currentState, Text: nextStateObject.ActionLog, DateTime: time.Now()})
		}

		nextExecution := nextStateObject.NextExecution
		// if the next execution is a valid date and time in the future then set it and break processing
		if !nextExecution.IsZero() {
			//if nextExecution.After(time.Now()) { // no need, if its in the past it will just run on the next pick up
			slog.Info("Setting next activation (specific)", "workflow_id", w.GetWorkflowData().ID, "next_activation", nextExecution, "worker_id", workerID)
			if err := r.UpdateNextActivationSpecific(w.GetWorkflowData().ID, nextExecution); err != nil {
				slog.Error("Error updating next activation", "error", err, "worker_id", workerID)
				return
			}
			_, _ = wa.Save(&domain.WorkflowAction{WorkflowID: w.GetWorkflowData().ID, ExecutorID: executorID, ExecutionCount: w.GetWorkflowData().RetryCount, Type: "SCHEDULE_ACTIVATION", Name: currentState, Text: nextExecution.String(), DateTime: time.Now()})
			break
			//}
		}
		nextExecutionOffset := results[0].Interface().(*models.NextState).NextExecutionOffset
		if nextExecutionOffset != "" {
			slog.Info("Setting next activation (offset)", "workflow_id", w.GetWorkflowData().ID, "offset", nextExecutionOffset, "worker_id", workerID)
			if err := r.UpdateNextActivationOffset(w.GetWorkflowData().ID, nextExecutionOffset); err != nil {
				slog.Error("Error updating next activation", "error", err, "worker_id", workerID)
				return
			}
			_, _ = wa.Save(&domain.WorkflowAction{WorkflowID: w.GetWorkflowData().ID, ExecutorID: executorID, ExecutionCount: w.GetWorkflowData().RetryCount, Type: "SCHEDULE_ACTIVATION", Name: currentState, Text: nextExecutionOffset, DateTime: time.Now()})
			break
		}

	}

	_, _ = wa.Save(&domain.WorkflowAction{WorkflowID: w.GetWorkflowData().ID, ExecutorID: executorID, ExecutionCount: w.GetWorkflowData().RetryCount, Type: "FINISHED", Name: currentState, Text: "FINISHED", DateTime: time.Now()})
	//clear out the executor id for another to possibly pick up the workflow
	r.ClearExecutorId(w.GetWorkflowData().ID)
	slog.Info("Workflow finished", "worker_id", workerID)

}

func processWorflowCompleted(w Workflow, r repository.WorkflowRepository, wa repository.WorkflowActionRepository, executorID int64, workerID string, currentState string) bool {
	slog.Info("Workflow completed", "worker_id", workerID)
	err := r.UpdateWorkflowStatus(w.GetWorkflowData().ID, "FINISHED")
	_, _ = wa.Save(&domain.WorkflowAction{WorkflowID: w.GetWorkflowData().ID, ExecutorID: executorID, ExecutionCount: w.GetWorkflowData().ExecutionCount, Type: "END", Name: currentState, Text: "workflow complete", DateTime: time.Now()})
	if err != nil {
		slog.Error("Error updating workflow status", "error", err, "worker_id", workerID)
		return true
	}
	return false
}

func processStateExecutionError(w Workflow, r repository.WorkflowRepository, wa repository.WorkflowActionRepository, executorID int64, workerID string, currentState string, callErr error) {
	slog.Error("Error executing state method", "state", currentState, "error", callErr, "worker_id", workerID)
	_, _ = wa.Save(&domain.WorkflowAction{
		WorkflowID:     w.GetWorkflowData().ID,
		ExecutorID:     executorID,
		ExecutionCount: w.GetWorkflowData().ExecutionCount,
		Type:           "ERROR",
		Name:           currentState,
		Text:           callErr.Error(),
		DateTime:       time.Now(),
	})
	//increment workflow retry counter
	if w.GetWorkflowData().RetryCount > w.GetRetryConfig().MaxRetryCount {
		slog.Error("Max retry count reached", "worker_id", workerID)
		_ = r.UpdateWorkflowStatus(w.GetWorkflowData().ID, "FAILED")
		_, _ = wa.Save(&domain.WorkflowAction{WorkflowID: w.GetWorkflowData().ID, ExecutorID: executorID, ExecutionCount: w.GetWorkflowData().ExecutionCount,
			Type: "FAILED", Name: currentState, Text: fmt.Sprintf("Max retry count reached :%s", w.GetWorkflowData().RetryCount), DateTime: time.Now()})
		return
	}

	if compareAndSaveWorkflowStateVars(w, r, workerID) {
		return
	}

	config := w.GetRetryConfig()
	nextActivation := time.Now().Add(config.SlidingInterval(w.GetWorkflowData().RetryCount))
	err := r.IncrementRetryCounterAndSetNextActivation(w.GetWorkflowData().ID, nextActivation)
	if err != nil {
		slog.Error("Error incrementing retry count", "error", err, "worker_id", workerID)
		return
	}
	_, _ = wa.Save(&domain.WorkflowAction{WorkflowID: w.GetWorkflowData().ID, ExecutorID: executorID, ExecutionCount: w.GetWorkflowData().ExecutionCount,
		Type: "RETRY", Name: currentState, Text: fmt.Sprintf("Retry at  :%s", nextActivation), DateTime: time.Now()})
	return
}

func compareAndSaveWorkflowStateVars(w Workflow, r repository.WorkflowRepository, workerID string) bool {
	jsonString, _ := json.Marshal(w.GetStateVariables())

	if string(jsonString) != w.GetWorkflowData().StateVars.String {
		slog.Info("Updating workflow variables", "workflow_id", w.GetWorkflowData().ID, "state_vars", string(jsonString), "worker_id", workerID)
		err2 := r.SaveWorkflowVariables(w.GetWorkflowData().ID, string(jsonString))
		if err2 != nil {
			slog.Error("Error saving workflow variables", "error", err2, "worker_id", workerID)
			return true
		}
	} else {
		slog.Info("Workflow variables unchanged", "worker_id", workerID)
	}
	return false
}
