package web

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/RealZimboGuy/gopherflow/pkg/gopherflow/domain"
)

func timeNowPlusHour() time.Time { return time.Now().UTC().Add(time.Hour) }

func postForm(t *testing.T, target string, form url.Values) *http.Request {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, target, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return req
}

func TestUsersHandler(t *testing.T) {
	tw := newTestWeb(t)

	req := httptest.NewRequest(http.MethodGet, "/users", nil)
	w := httptest.NewRecorder()

	tw.Ctrl.usersHandler(w, req)

	assertRendered(t, w, "admin")
}

func TestCreateUserPageHandler(t *testing.T) {
	tw := newTestWeb(t)

	req := httptest.NewRequest(http.MethodGet, "/users/create", nil)
	w := httptest.NewRecorder()

	tw.Ctrl.createUserHandler(w, req)

	assertRendered(t, w)
}

func TestCreateUserSubmitHandler(t *testing.T) {
	tw := newTestWeb(t)

	t.Run("creates the user and hashes the password", func(t *testing.T) {
		req := postForm(t, "/users/create", url.Values{
			"username": {"weblogin"},
			"password": {"s3cret"},
			"apiKey":   {"key-weblogin"},
			"enabled":  {"on"},
		})
		w := httptest.NewRecorder()

		tw.Ctrl.createUserSubmitHandler(w, req)

		if w.Code != http.StatusOK && w.Code != http.StatusSeeOther && w.Code != http.StatusFound {
			t.Fatalf("status = %d, want a success or redirect; body %s", w.Code, w.Body.String())
		}
		stored, err := tw.Users.FindByUsername("weblogin")
		if err != nil {
			t.Fatalf("FindByUsername: %v", err)
		}
		if stored == nil {
			t.Fatal("user was not created")
		}
		if stored.Password == "s3cret" {
			t.Error("password was stored in plain text")
		}
	})

	t.Run("rejects a duplicate username", func(t *testing.T) {
		req := postForm(t, "/users/create", url.Values{
			"username": {"admin"},
			"password": {"pw"},
		})
		w := httptest.NewRecorder()

		tw.Ctrl.createUserSubmitHandler(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want 400 for a duplicate username", w.Code)
		}
	})

	t.Run("rejects a missing username", func(t *testing.T) {
		req := postForm(t, "/users/create", url.Values{"password": {"pw"}})
		w := httptest.NewRecorder()

		tw.Ctrl.createUserSubmitHandler(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want 400", w.Code)
		}
	})

	t.Run("rejects a missing password", func(t *testing.T) {
		req := postForm(t, "/users/create", url.Values{"username": {"nopw"}})
		w := httptest.NewRecorder()

		tw.Ctrl.createUserSubmitHandler(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want 400", w.Code)
		}
	})
}

func TestEditUserHandler(t *testing.T) {
	tw := newTestWeb(t)
	id, err := tw.Users.Save(&domain.User{Username: "editme", Password: "hashed"})
	if err != nil {
		t.Fatalf("seed user: %v", err)
	}

	t.Run("renders the form", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/users/1/edit", nil)
		req.SetPathValue("id", strconv.FormatInt(id, 10))
		w := httptest.NewRecorder()

		tw.Ctrl.editUserHandler(w, req)

		assertRendered(t, w, "editme")
	})

	t.Run("unknown user", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/users/9999/edit", nil)
		req.SetPathValue("id", "9999")
		w := httptest.NewRecorder()

		tw.Ctrl.editUserHandler(w, req)

		if w.Code == http.StatusOK {
			t.Error("status = 200 for an unknown user, want an error")
		}
	})

	t.Run("non-numeric id", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/users/abc/edit", nil)
		req.SetPathValue("id", "abc")
		w := httptest.NewRecorder()

		tw.Ctrl.editUserHandler(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want 400", w.Code)
		}
	})

	t.Run("missing id", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/users//edit", nil)
		w := httptest.NewRecorder()

		tw.Ctrl.editUserHandler(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want 400", w.Code)
		}
	})
}

func TestEditUserSubmitHandler(t *testing.T) {
	tw := newTestWeb(t)
	id, err := tw.Users.Save(&domain.User{Username: "beforerename", Password: "hashed"})
	if err != nil {
		t.Fatalf("seed user: %v", err)
	}

	t.Run("updates the user", func(t *testing.T) {
		req := postForm(t, "/users/1/edit", url.Values{
			"username": {"afterrename"},
			"apiKey":   {"new-key"},
			"enabled":  {"on"},
		})
		req.SetPathValue("id", strconv.FormatInt(id, 10))
		w := httptest.NewRecorder()

		tw.Ctrl.editUserSubmitHandler(w, req)

		if w.Code != http.StatusOK && w.Code != http.StatusSeeOther && w.Code != http.StatusFound {
			t.Fatalf("status = %d; body %s", w.Code, w.Body.String())
		}
		stored, err := tw.Users.FindById(id)
		if err != nil {
			t.Fatalf("FindById: %v", err)
		}
		if stored == nil || stored.Username != "afterrename" {
			t.Errorf("got %+v, want the renamed user", stored)
		}
	})

	t.Run("unknown user", func(t *testing.T) {
		req := postForm(t, "/users/9999/edit", url.Values{"username": {"x"}})
		req.SetPathValue("id", "9999")
		w := httptest.NewRecorder()

		tw.Ctrl.editUserSubmitHandler(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("status = %d, want 404", w.Code)
		}
	})

	t.Run("non-numeric id", func(t *testing.T) {
		req := postForm(t, "/users/abc/edit", url.Values{"username": {"x"}})
		req.SetPathValue("id", "abc")
		w := httptest.NewRecorder()

		tw.Ctrl.editUserSubmitHandler(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want 400", w.Code)
		}
	})

	t.Run("missing id", func(t *testing.T) {
		req := postForm(t, "/users//edit", url.Values{"username": {"x"}})
		w := httptest.NewRecorder()

		tw.Ctrl.editUserSubmitHandler(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want 400", w.Code)
		}
	})
}

func TestDeleteUserHandler(t *testing.T) {
	tw := newTestWeb(t)
	id, err := tw.Users.Save(&domain.User{Username: "deleteme", Password: "hashed"})
	if err != nil {
		t.Fatalf("seed user: %v", err)
	}

	t.Run("deletes a user", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/users/1/delete", nil)
		req.SetPathValue("id", strconv.FormatInt(id, 10))
		w := httptest.NewRecorder()

		tw.Ctrl.deleteUserHandler(w, req)

		if gone, _ := tw.Users.FindById(id); gone != nil {
			t.Error("user still present after delete")
		}
	})

	t.Run("refuses to delete the logged-in user", func(t *testing.T) {
		selfID, err := tw.Users.Save(&domain.User{Username: "self", Password: "hashed"})
		if err != nil {
			t.Fatalf("seed user: %v", err)
		}
		if err := tw.Users.UpdateSession(selfID, "sess-self", timeNowPlusHour()); err != nil {
			t.Fatalf("UpdateSession: %v", err)
		}

		req := httptest.NewRequest(http.MethodPost, "/users/1/delete", nil)
		req.SetPathValue("id", strconv.FormatInt(selfID, 10))
		req.AddCookie(&http.Cookie{Name: "sessionId", Value: "sess-self"})
		w := httptest.NewRecorder()

		tw.Ctrl.deleteUserHandler(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want 400 when deleting your own account", w.Code)
		}
		if still, _ := tw.Users.FindById(selfID); still == nil {
			t.Error("the logged-in user was deleted")
		}
	})

	t.Run("non-numeric id", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/users/abc/delete", nil)
		req.SetPathValue("id", "abc")
		w := httptest.NewRecorder()

		tw.Ctrl.deleteUserHandler(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want 400", w.Code)
		}
	})

	t.Run("missing id", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/users//delete", nil)
		w := httptest.NewRecorder()

		tw.Ctrl.deleteUserHandler(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want 400", w.Code)
		}
	})
}
