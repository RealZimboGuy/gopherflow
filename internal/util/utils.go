package util

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
)

func DecodeJSONBody[T any](r *http.Request) (T, error) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		var zero T
		return zero, fmt.Errorf("read body error: %w", err)
	}
	defer r.Body.Close()

	var data T
	if err := json.Unmarshal(body, &data); err != nil {
		var zero T
		return zero, fmt.Errorf("json unmarshal error: %w", err)
	}
	return data, nil
}

func DecodeJSONBodyResponse[T any](r *http.Response) (T, error) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		var zero T
		return zero, fmt.Errorf("read body error: %w", err)
	}
	defer r.Body.Close()

	var data T
	if err := json.Unmarshal(body, &data); err != nil {
		var zero T
		return zero, fmt.Errorf("json unmarshal error: %w", err)
	}
	return data, nil
}

func WriteJSONResponse[T any](w http.ResponseWriter, status int, data T) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		slog.Error("Failed to encode response", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}
