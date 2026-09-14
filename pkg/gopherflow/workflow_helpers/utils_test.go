package workflow_helpers

import "testing"

type sample struct {
	Name string `json:"name"`
	Age  int    `json:"age"`
}

func TestSaveStructToStateVars(t *testing.T) {
	vars := map[string]string{}

	if err := SaveStructToStateVars(vars, "person", sample{Name: "Ada", Age: 36}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := `{"name":"Ada","age":36}`
	if got := vars["person"]; got != want {
		t.Errorf("stateVars[person] = %q, want %q", got, want)
	}
}

func TestSaveStructToStateVarsOverwritesExistingKey(t *testing.T) {
	vars := map[string]string{"person": "stale"}

	if err := SaveStructToStateVars(vars, "person", sample{Name: "Grace", Age: 45}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if vars["person"] == "stale" {
		t.Error("existing value was not overwritten")
	}
}

func TestSaveStructToStateVarsMarshalError(t *testing.T) {
	vars := map[string]string{}

	// A channel cannot be marshalled to JSON.
	err := SaveStructToStateVars(vars, "bad", make(chan int))
	if err == nil {
		t.Fatal("expected an error for an unmarshalable value, got nil")
	}
	if _, ok := vars["bad"]; ok {
		t.Error("key was written despite the marshal failing")
	}
}

func TestLoadStructFromStateVars(t *testing.T) {
	vars := map[string]string{"person": `{"name":"Ada","age":36}`}

	got, err := LoadStructFromStateVars[sample](vars, "person")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Name != "Ada" || got.Age != 36 {
		t.Errorf("got %+v, want {Name:Ada Age:36}", *got)
	}
}

func TestLoadStructFromStateVarsMissingKey(t *testing.T) {
	got, err := LoadStructFromStateVars[sample](map[string]string{}, "absent")
	if err == nil {
		t.Fatal("expected an error for a missing key, got nil")
	}
	if got != nil {
		t.Errorf("expected nil result, got %+v", got)
	}
}

func TestLoadStructFromStateVarsInvalidJSON(t *testing.T) {
	vars := map[string]string{"person": "{not json"}

	got, err := LoadStructFromStateVars[sample](vars, "person")
	if err == nil {
		t.Fatal("expected an error for invalid JSON, got nil")
	}
	if got != nil {
		t.Errorf("expected nil result, got %+v", got)
	}
}

func TestRoundTrip(t *testing.T) {
	vars := map[string]string{}
	in := sample{Name: "Alan", Age: 41}

	if err := SaveStructToStateVars(vars, "k", in); err != nil {
		t.Fatalf("save: %v", err)
	}
	out, err := LoadStructFromStateVars[sample](vars, "k")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if *out != in {
		t.Errorf("round trip changed the value: got %+v, want %+v", *out, in)
	}
}
