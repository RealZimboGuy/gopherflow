package controllers

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/RealZimboGuy/gopherflow/pkg/gopherflow/core"
	"github.com/RealZimboGuy/gopherflow/pkg/gopherflow/domain"
)

// MockUserRepo implements engine.UserRepo for testing
type MockUserRepo struct {
	FindBySessionIDFunc         func(sessionID string, now time.Time) (*domain.User, error)
	FindByApiKeyFunc            func(apiKey string) (*domain.User, error)
	FindAllFunc                 func() (*[]domain.User, error)
	SaveFunc                    func(user *domain.User) (int64, error)
	FindByIdFunc                func(id int64) (*domain.User, error)
	DeleteByIdFunc              func(id int64) error
	FindByUsernameFunc          func(username string) (*domain.User, error)
	UpdateSessionFunc           func(userID int64, sessionID string, expiry time.Time) error
	ClearSessionBySessionIDFunc func(sessionID string) error
	UpdateUserFunc              func(id int64, username string, apiKey sql.NullString, enabled sql.NullBool) error
}

func (m *MockUserRepo) FindBySessionID(sessionID string, now time.Time) (*domain.User, error) {
	if m.FindBySessionIDFunc != nil {
		return m.FindBySessionIDFunc(sessionID, now)
	}
	return nil, nil
}
func (m *MockUserRepo) FindByApiKey(apiKey string) (*domain.User, error) {
	if m.FindByApiKeyFunc != nil {
		return m.FindByApiKeyFunc(apiKey)
	}
	return nil, nil
}
func (m *MockUserRepo) FindAll() (*[]domain.User, error) {
	if m.FindAllFunc != nil {
		return m.FindAllFunc()
	}
	return nil, nil
}
func (m *MockUserRepo) Save(user *domain.User) (int64, error) {
	if m.SaveFunc != nil {
		return m.SaveFunc(user)
	}
	return 0, nil
}
func (m *MockUserRepo) FindById(id int64) (*domain.User, error) {
	if m.FindByIdFunc != nil {
		return m.FindByIdFunc(id)
	}
	return nil, nil
}
func (m *MockUserRepo) DeleteById(id int64) error {
	if m.DeleteByIdFunc != nil {
		return m.DeleteByIdFunc(id)
	}
	return nil
}
func (m *MockUserRepo) FindByUsername(username string) (*domain.User, error) {
	if m.FindByUsernameFunc != nil {
		return m.FindByUsernameFunc(username)
	}
	return nil, nil
}
func (m *MockUserRepo) UpdateSession(userID int64, sessionID string, expiry time.Time) error {
	if m.UpdateSessionFunc != nil {
		return m.UpdateSessionFunc(userID, sessionID, expiry)
	}
	return nil
}
func (m *MockUserRepo) ClearSessionBySessionID(sessionID string) error {
	if m.ClearSessionBySessionIDFunc != nil {
		return m.ClearSessionBySessionIDFunc(sessionID)
	}
	return nil
}
func (m *MockUserRepo) UpdateUser(id int64, username string, apiKey sql.NullString, enabled sql.NullBool) error {
	if m.UpdateUserFunc != nil {
		return m.UpdateUserFunc(id, username, apiKey, enabled)
	}
	return nil
}

func TestAuthController_RequireAuth_LoginPath(t *testing.T) {
	ac := NewBaseController(&MockUserRepo{})

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/login", nil)
	w := httptest.NewRecorder()

	ac.RequireAuth(nextHandler).ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestAuthController_RequireAuth_SessionCookie(t *testing.T) {
	mockRepo := &MockUserRepo{
		FindBySessionIDFunc: func(sessionID string, now time.Time) (*domain.User, error) {
			if sessionID == "valid_session" {
				return &domain.User{Username: "testuser"}, nil
			}
			return nil, nil
		},
	}
	ac := NewBaseController(mockRepo)

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		username := r.Context().Value(core.CtxKeyUsername)
		if username != "testuser" {
			t.Errorf("Expected username in context, got %v", username)
		}
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/protected", nil)
	req.AddCookie(&http.Cookie{Name: "sessionId", Value: "valid_session"})
	w := httptest.NewRecorder()

	ac.RequireAuth(nextHandler).ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestAuthController_RequireAuth_ApiKey(t *testing.T) {
	mockRepo := &MockUserRepo{
		FindByApiKeyFunc: func(apiKey string) (*domain.User, error) {
			if apiKey == "valid_key" {
				return &domain.User{Username: "api_user"}, nil
			}
			return nil, nil
		},
	}
	ac := NewBaseController(mockRepo)

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		username := r.Context().Value(core.CtxKeyUsername)
		if username != "api_user" {
			t.Errorf("Expected username in context, got %v", username)
		}
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("X-API-Key", "valid_key")
	w := httptest.NewRecorder()

	ac.RequireAuth(nextHandler).ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestAuthController_RequireAuth_Unauthorized(t *testing.T) {
	mockRepo := &MockUserRepo{
		FindBySessionIDFunc: func(sessionID string, now time.Time) (*domain.User, error) {
			return nil, nil
		},
		FindByApiKeyFunc: func(apiKey string) (*domain.User, error) {
			return nil, nil
		},
	}
	ac := NewBaseController(mockRepo)

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("Next handler should not be called")
	})

	// Case 1: No credentials - Redirect to login
	req := httptest.NewRequest("GET", "/protected", nil)
	w := httptest.NewRecorder()
	ac.RequireAuth(nextHandler).ServeHTTP(w, req)
	if w.Code != http.StatusSeeOther {
		t.Errorf("Expected redirect 303, got %d", w.Code)
	}

	// Case 2: Invalid API Key - Unauthorized
	req = httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("X-API-Key", "invalid_key")
	w = httptest.NewRecorder()
	ac.RequireAuth(nextHandler).ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected unauthorized 401, got %d", w.Code)
	}
}
