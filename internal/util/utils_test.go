package util

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type payload struct {
	Name string `json:"name"`
	Age  int    `json:"age"`
}

// errReader fails on Read so the read-error branch can be exercised.
type errReader struct{}

func (errReader) Read([]byte) (int, error) { return 0, errors.New("boom") }
func (errReader) Close() error             { return nil }

func TestDecodeJSONBody(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"name":"Ada","age":36}`))

	got, err := DecodeJSONBody[payload](req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Name != "Ada" || got.Age != 36 {
		t.Errorf("got %+v, want {Ada 36}", got)
	}
}

func TestDecodeJSONBodyInvalidJSON(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("{not json"))

	if _, err := DecodeJSONBody[payload](req); err == nil {
		t.Fatal("expected an error for invalid JSON, got nil")
	}
}

func TestDecodeJSONBodyReadError(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Body = errReader{}

	if _, err := DecodeJSONBody[payload](req); err == nil {
		t.Fatal("expected an error when the body cannot be read, got nil")
	}
}

func TestDecodeJSONBodyResponse(t *testing.T) {
	resp := &http.Response{Body: io.NopCloser(strings.NewReader(`{"name":"Grace","age":45}`))}

	got, err := DecodeJSONBodyResponse[payload](resp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Name != "Grace" || got.Age != 45 {
		t.Errorf("got %+v, want {Grace 45}", got)
	}
}

func TestDecodeJSONBodyResponseInvalidJSON(t *testing.T) {
	resp := &http.Response{Body: io.NopCloser(strings.NewReader("nope"))}

	if _, err := DecodeJSONBodyResponse[payload](resp); err == nil {
		t.Fatal("expected an error for invalid JSON, got nil")
	}
}

func TestDecodeJSONBodyResponseReadError(t *testing.T) {
	resp := &http.Response{Body: errReader{}}

	if _, err := DecodeJSONBodyResponse[payload](resp); err == nil {
		t.Fatal("expected an error when the body cannot be read, got nil")
	}
}

func TestWriteJSONResponse(t *testing.T) {
	rec := httptest.NewRecorder()

	WriteJSONResponse(rec, http.StatusCreated, payload{Name: "Alan", Age: 41})

	if rec.Code != http.StatusCreated {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusCreated)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}

	var got payload
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("response body is not valid JSON: %v", err)
	}
	if got.Name != "Alan" || got.Age != 41 {
		t.Errorf("got %+v, want {Alan 41}", got)
	}
}

func TestWriteJSONResponseEncodeError(t *testing.T) {
	rec := httptest.NewRecorder()

	// A channel cannot be encoded; the handler must not panic.
	WriteJSONResponse(rec, http.StatusOK, make(chan int))

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d (already written before the encode failed)", rec.Code, http.StatusOK)
	}
}
