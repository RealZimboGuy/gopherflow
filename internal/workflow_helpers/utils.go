package workflow_helpers

import (
	"encoding/json"
	"fmt"
)

func SaveStructToStateVars[T any](stateVars map[string]string, key string, data T) error {
	bytes, err := json.Marshal(data)
	if err != nil {
		return err
	}
	stateVars[key] = string(bytes)
	return nil
}

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
