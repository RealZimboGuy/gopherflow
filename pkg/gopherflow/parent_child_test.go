package gopherflow

import (
	"testing"
)

func TestCreateChildWorkflowRequest(t *testing.T) {
	vars := map[string]string{"input": "value"}

	req := CreateChildWorkflowRequest("MyChildWorkflow", "child-1", vars)

	if req.WorkflowType != "MyChildWorkflow" {
		t.Errorf("WorkflowType = %q, want MyChildWorkflow", req.WorkflowType)
	}
	if req.BusinessKey != "child-1" {
		t.Errorf("BusinessKey = %q, want child-1", req.BusinessKey)
	}
	if req.StateVariables["input"] != "value" {
		t.Errorf("StateVariables = %v, want the supplied map", req.StateVariables)
	}
}

func TestCreateChildWorkflowRequestWithNoStateVars(t *testing.T) {
	req := CreateChildWorkflowRequest("MyChildWorkflow", "child-2", nil)

	if req.StateVariables != nil {
		t.Errorf("StateVariables = %v, want nil to be preserved", req.StateVariables)
	}
}

func TestParseChildWorkflowResults(t *testing.T) {
	vars := map[string]string{"childResults": `{"ip":"1.2.3.4","status":"ok"}`}

	got, err := ParseChildWorkflowResults(vars, "childResults")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got["ip"] != "1.2.3.4" || got["status"] != "ok" {
		t.Errorf("got %v, want the parsed child results", got)
	}
}

func TestParseChildWorkflowResultsMissingKey(t *testing.T) {
	got, err := ParseChildWorkflowResults(map[string]string{}, "absent")
	if err != nil {
		t.Fatalf("a missing key should not be an error, got %v", err)
	}
	if got != nil {
		t.Errorf("got %v, want nil for a missing key", got)
	}
}

func TestParseChildWorkflowResultsInvalidJSON(t *testing.T) {
	vars := map[string]string{"childResults": "{not json"}

	got, err := ParseChildWorkflowResults(vars, "childResults")
	if err == nil {
		t.Fatal("expected an error for invalid JSON, got nil")
	}
	if got != nil {
		t.Errorf("got %v, want nil alongside the error", got)
	}
}
