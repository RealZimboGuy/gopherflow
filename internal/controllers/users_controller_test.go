package controllers

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/RealZimboGuy/gopherflow/pkg/gopherflow/domain"
	"golang.org/x/crypto/bcrypt"
)

func TestUsersController_GetUsers(t *testing.T) {
	mockUserRepo := &MockUserRepo{
		FindAllFunc: func() (*[]domain.User, error) {
			return &[]domain.User{
				{ID: 1, Username: "user1"},
			}, nil
		},
	}

	c := NewUsersController(mockUserRepo)

	req := httptest.NewRequest("GET", "/api/users", nil)
	w := httptest.NewRecorder()

	c.handleGetUsers(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var users []domain.User
	if err := json.NewDecoder(resp.Body).Decode(&users); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	if len(users) != 1 {
		t.Errorf("Expected 1 user, got %d", len(users))
	}
}

func TestUsersController_CreateUser(t *testing.T) {
	var capturedPassword string
	var capturedEnabled sql.NullBool
	mockUserRepo := &MockUserRepo{
		SaveFunc: func(user *domain.User) (int64, error) {
			capturedPassword = user.Password
			capturedEnabled = user.Enabled
			return 123, nil
		},
	}

	c := NewUsersController(mockUserRepo)

	body := []byte(`{"username":"newuser","password":"password"}`)
	req := httptest.NewRequest("POST", "/api/users", bytes.NewReader(body))
	w := httptest.NewRecorder()

	c.handleCreateUser(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusCreated {
		t.Errorf("Expected status 201, got %d", resp.StatusCode)
	}

	var user domain.User
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	if user.ID != 123 {
		t.Errorf("Expected ID 123, got %d", user.ID)
	}
	if capturedPassword == "password" || capturedPassword == "" {
		t.Errorf("expected password to be bcrypt-hashed, got %q", capturedPassword)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(capturedPassword), []byte("password")); err != nil {
		t.Errorf("stored hash does not verify against plaintext password: %v", err)
	}
	if !capturedEnabled.Valid || !capturedEnabled.Bool {
		t.Errorf("expected new user to default to enabled=true, got %+v", capturedEnabled)
	}
}

func TestUsersController_DeleteUser(t *testing.T) {
	mockUserRepo := &MockUserRepo{
		DeleteByIdFunc: func(id int64) error {
			if id == 1 {
				return nil
			}
			return sql.ErrNoRows // Simulating error for verification
		},
	}

	c := NewUsersController(mockUserRepo)

	req := httptest.NewRequest("DELETE", "/api/users/1", nil)
	req.SetPathValue("id", "1")
	w := httptest.NewRecorder()

	c.handleDeleteUser(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusNoContent {
		t.Errorf("Expected status 204, got %d", resp.StatusCode)
	}
}

func TestUsersController_GetUserById(t *testing.T) {
	mockUserRepo := &MockUserRepo{
		FindByIdFunc: func(id int64) (*domain.User, error) {
			if id == 1 {
				return &domain.User{ID: 1, Username: "found"}, nil
			}
			return nil, nil
		},
	}

	c := NewUsersController(mockUserRepo)

	// Success case
	req := httptest.NewRequest("GET", "/api/users/1", nil)
	req.SetPathValue("id", "1")
	w := httptest.NewRecorder()
	c.handleGetUserById(w, req)
	if w.Result().StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Result().StatusCode)
	}

	// Not found case
	req = httptest.NewRequest("GET", "/api/users/999", nil)
	req.SetPathValue("id", "999")
	w = httptest.NewRecorder()
	c.handleGetUserById(w, req)
	if w.Result().StatusCode != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Result().StatusCode)
	}
}
