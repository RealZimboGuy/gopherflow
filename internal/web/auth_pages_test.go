package web

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/RealZimboGuy/gopherflow/pkg/gopherflow/domain"
	"golang.org/x/crypto/bcrypt"
)

// seedLogin creates a user with a real bcrypt hash so the login path can be
// exercised end to end.
func (tw *testWeb) seedLogin(t *testing.T, username, password string, enabled bool) int64 {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	id, err := tw.Users.Save(&domain.User{
		Username: username,
		Password: string(hash),
		Enabled:  sql.NullBool{Bool: enabled, Valid: true},
	})
	if err != nil {
		t.Fatalf("seed login user: %v", err)
	}
	return id
}

func TestLoginPageHandler(t *testing.T) {
	tw := newTestWeb(t)

	req := httptest.NewRequest(http.MethodGet, "/login", nil)
	w := httptest.NewRecorder()

	tw.Ctrl.loginPageHandler(w, req)

	assertRendered(t, w)
}

func TestLoginSubmitSuccess(t *testing.T) {
	tw := newTestWeb(t)
	tw.seedLogin(t, "loginok", "correct-horse", true)

	req := postForm(t, "/login", url.Values{
		"username": {"loginok"},
		"password": {"correct-horse"},
	})
	w := httptest.NewRecorder()

	tw.Ctrl.loginSubmitHandler(w, req)

	if w.Code != http.StatusSeeOther && w.Code != http.StatusFound && w.Code != http.StatusOK {
		t.Fatalf("status = %d; body %s", w.Code, w.Body.String())
	}

	var sessionCookie *http.Cookie
	for _, c := range w.Result().Cookies() {
		if c.Name == "sessionId" {
			sessionCookie = c
		}
	}
	if sessionCookie == nil || sessionCookie.Value == "" {
		t.Fatal("no sessionId cookie was set on a successful login")
	}
	if !sessionCookie.HttpOnly {
		t.Error("the session cookie is not HttpOnly")
	}

	stored, err := tw.Users.FindBySessionID(sessionCookie.Value, timeNowPlusHour().Add(-time.Hour))
	if err != nil {
		t.Fatalf("FindBySessionID: %v", err)
	}
	if stored == nil {
		t.Error("the session was not persisted")
	}
}

func TestLoginSubmitFailures(t *testing.T) {
	tw := newTestWeb(t)
	tw.seedLogin(t, "loginuser", "rightpass", true)
	tw.seedLogin(t, "disableduser", "rightpass", false)

	tests := []struct {
		name string
		form url.Values
		want int
	}{
		{"wrong password", url.Values{"username": {"loginuser"}, "password": {"wrong"}}, http.StatusUnauthorized},
		{"unknown user", url.Values{"username": {"ghost"}, "password": {"any"}}, http.StatusUnauthorized},
		{"disabled user", url.Values{"username": {"disableduser"}, "password": {"rightpass"}}, http.StatusUnauthorized},
		{"missing username", url.Values{"password": {"rightpass"}}, http.StatusUnauthorized},
		{"missing password", url.Values{"username": {"loginuser"}}, http.StatusUnauthorized},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := postForm(t, "/login", tt.form)
			w := httptest.NewRecorder()

			tw.Ctrl.loginSubmitHandler(w, req)

			if w.Code != tt.want {
				t.Errorf("status = %d, want %d; body %s", w.Code, tt.want, w.Body.String())
			}
			for _, c := range w.Result().Cookies() {
				if c.Name == "sessionId" && c.Value != "" {
					t.Error("a session cookie was issued for a failed login")
				}
			}
		})
	}
}

func TestLogoutHandler(t *testing.T) {
	tw := newTestWeb(t)
	id := tw.seedLogin(t, "logoutuser", "pw", true)
	if err := tw.Users.UpdateSession(id, "sess-logout", timeNowPlusHour()); err != nil {
		t.Fatalf("UpdateSession: %v", err)
	}

	t.Run("clears the session and redirects", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/logout", nil)
		req.AddCookie(&http.Cookie{Name: "sessionId", Value: "sess-logout"})
		w := httptest.NewRecorder()

		tw.Ctrl.logoutHandler(w, req)

		if w.Code != http.StatusSeeOther {
			t.Errorf("status = %d, want 303", w.Code)
		}
		if loc := w.Header().Get("Location"); loc != "/login" {
			t.Errorf("Location = %q, want /login", loc)
		}

		still, err := tw.Users.FindBySessionID("sess-logout", timeNowPlusHour().Add(-time.Hour))
		if err != nil {
			t.Fatalf("FindBySessionID: %v", err)
		}
		if still != nil {
			t.Error("the session still resolves after logout")
		}
	})

	t.Run("redirects even without a cookie", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/logout", nil)
		w := httptest.NewRecorder()

		tw.Ctrl.logoutHandler(w, req)

		if w.Code != http.StatusSeeOther {
			t.Errorf("status = %d, want 303", w.Code)
		}
	})
}
