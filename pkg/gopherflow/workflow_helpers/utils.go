package workflow_helpers

import (
	"encoding/json"
	"fmt"
)

// SaveStructToStateVars marshals data to JSON and stores it in stateVars under
// key, so a struct can be carried between workflow states. It returns an error
// if the value cannot be marshalled, leaving stateVars unchanged.
func SaveStructToStateVars[T any](stateVars map[string]string, key string, data T) error {
	bytes, err := json.Marshal(data)
	if err != nil {
		return err
	}
	stateVars[key] = string(bytes)
	return nil
}

// LoadStructFromStateVars reads the JSON value stored at key and unmarshals it
// into a T. It returns an error if the key is absent or the value does not parse.
func LoadStructFromStateVars[T any](stateVars map[string]string, key string) (*T, error) {
	data, ok := stateVars[key]
	if !ok {
		return nil, fmt.Errorf("key %s not found in stateVars", key)
	}
	var out T
	if err := json.Unmarshal([]byte(data), &out); err != nil {
		return nil, err
	}
	return &out, nil
}
