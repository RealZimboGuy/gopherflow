package controllers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/RealZimboGuy/gopherflow/pkg/gopherflow/domain"
)

func newUsersController(t *testing.T) (*UsersController, *realController) {
	t.Helper()
	rc := newRealController(t)
	return NewUsersController(rc.Users), rc
}

func TestHandleGetUsers(t *testing.T) {
	uc, _ := newUsersController(t)

	t.Run("returns the seeded admin", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/users", nil)
		w := httptest.NewRecorder()

		uc.handleGetUsers(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200; body %s", w.Code, w.Body.String())
		}
		var users []domain.User
		if err := json.Unmarshal(w.Body.Bytes(), &users); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if len(users) == 0 {
			t.Error("expected at least the seeded admin user")
		}
	})

	t.Run("wrong method", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/api/users", nil)
		w := httptest.NewRecorder()

		uc.handleGetUsers(w, req)

		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("status = %d, want 405", w.Code)
		}
	})
}

func TestHandleCreateUser(t *testing.T) {
	uc, rc := newUsersController(t)

	t.Run("creates a user with a hashed password", func(t *testing.T) {
		req := postJSON(t, "/api/users", createUserRequest{Username: "newbie", Password: "s3cret"})
		w := httptest.NewRecorder()

		uc.handleCreateUser(w, req)

		if w.Code != http.StatusOK && w.Code != http.StatusCreated {
			t.Fatalf("status = %d, want 200 or 201; body %s", w.Code, w.Body.String())
		}

		stored, err := rc.Users.FindByUsername("newbie")
		if err != nil {
			t.Fatalf("FindByUsername: %v", err)
		}
		if stored == nil {
			t.Fatal("user was not persisted")
		}
		if stored.Password == "s3cret" {
			t.Error("password was stored in plain text")
		}
	})

	t.Run("trims whitespace around the username", func(t *testing.T) {
		req := postJSON(t, "/api/users", createUserRequest{Username: "  spaced  ", Password: "pw"})
		w := httptest.NewRecorder()

		uc.handleCreateUser(w, req)

		if stored, _ := rc.Users.FindByUsername("spaced"); stored == nil {
			t.Error("username was not trimmed before saving")
		}
	})

	t.Run("rejects a blank username", func(t *testing.T) {
		req := postJSON(t, "/api/users", createUserRequest{Username: "   ", Password: "pw"})
		w := httptest.NewRecorder()

		uc.handleCreateUser(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want 400", w.Code)
		}
	})

	t.Run("rejects a missing password", func(t *testing.T) {
		req := postJSON(t, "/api/users", createUserRequest{Username: "nopass"})
		w := httptest.NewRecorder()

		uc.handleCreateUser(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want 400", w.Code)
		}
	})

	t.Run("rejects invalid json", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/users", bytes.NewReader([]byte("{bad")))
		w := httptest.NewRecorder()

		uc.handleCreateUser(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want 400", w.Code)
		}
	})

	t.Run("wrong method", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/users", nil)
		w := httptest.NewRecorder()

		uc.handleCreateUser(w, req)

		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("status = %d, want 405", w.Code)
		}
	})
}

func TestHandleGetUserById(t *testing.T) {
	uc, rc := newUsersController(t)

	id, err := rc.Users.Save(&domain.User{Username: "lookup", Password: "hashed"})
	if err != nil {
		t.Fatalf("seed user: %v", err)
	}

	t.Run("found", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/users/1", nil)
		req.SetPathValue("id", itoa(id))
		w := httptest.NewRecorder()

		uc.handleGetUserById(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200; body %s", w.Code, w.Body.String())
		}
	})

	t.Run("unknown id", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/users/9999", nil)
		req.SetPathValue("id", "9999")
		w := httptest.NewRecorder()

		uc.handleGetUserById(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("status = %d, want 404", w.Code)
		}
	})

	t.Run("non-numeric id", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/users/abc", nil)
		req.SetPathValue("id", "abc")
		w := httptest.NewRecorder()

		uc.handleGetUserById(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want 400", w.Code)
		}
	})

	t.Run("wrong method", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/users/1", nil)
		w := httptest.NewRecorder()

		uc.handleGetUserById(w, req)

		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("status = %d, want 405", w.Code)
		}
	})
}

func TestHandleDeleteUser(t *testing.T) {
	uc, rc := newUsersController(t)

	id, err := rc.Users.Save(&domain.User{Username: "doomed", Password: "hashed"})
	if err != nil {
		t.Fatalf("seed user: %v", err)
	}

	t.Run("deletes the user", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/api/users/1", nil)
		req.SetPathValue("id", itoa(id))
		w := httptest.NewRecorder()

		uc.handleDeleteUser(w, req)

		if w.Code != http.StatusOK && w.Code != http.StatusNoContent {
			t.Fatalf("status = %d, want 200 or 204; body %s", w.Code, w.Body.String())
		}
		if gone, _ := rc.Users.FindById(id); gone != nil {
			t.Error("user still present after delete")
		}
	})

	t.Run("non-numeric id", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/api/users/abc", nil)
		req.SetPathValue("id", "abc")
		w := httptest.NewRecorder()

		uc.handleDeleteUser(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want 400", w.Code)
		}
	})

	t.Run("wrong method", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/users/1", nil)
		w := httptest.NewRecorder()

		uc.handleDeleteUser(w, req)

		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("status = %d, want 405", w.Code)
		}
	})
}
