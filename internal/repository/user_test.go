package repository

import (
	"database/sql"
	"testing"
	"time"
)

func TestUserSaveAndFindByUsername(t *testing.T) {
	repo, _ := newUserRepo(t)

	u := newUser("alice")
	id, err := repo.Save(u)
	if err != nil {
		t.Fatalf("Save: %v", err)
	}
	if id == 0 {
		t.Fatal("Save returned id 0")
	}
	if u.ID != id {
		t.Errorf("Save did not write the id back: u.ID = %d, want %d", u.ID, id)
	}

	got, err := repo.FindByUsername("alice")
	if err != nil {
		t.Fatalf("FindByUsername: %v", err)
	}
	if got == nil {
		t.Fatal("FindByUsername returned nil for a saved user")
	}
	if got.Username != "alice" || got.ID != id {
		t.Errorf("got %+v, want username alice with id %d", got, id)
	}
	if got.ApiKey.String != u.ApiKey.String {
		t.Errorf("api key = %q, want %q", got.ApiKey.String, u.ApiKey.String)
	}
}

func TestUserSaveDefaultsCreatedTimestamp(t *testing.T) {
	repo, clock := newUserRepo(t)

	u := newUser("bob")
	u.Created = sql.NullTime{} // deliberately unset

	if _, err := repo.Save(u); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if !u.Created.Valid {
		t.Fatal("Save did not default the created timestamp")
	}
	if !u.Created.Time.Equal(clock.Now().UTC()) {
		t.Errorf("created = %v, want the clock time %v", u.Created.Time, clock.Now().UTC())
	}
}

func TestUserFindByUsernameMissingReturnsNilNil(t *testing.T) {
	repo, _ := newUserRepo(t)

	got, err := repo.FindByUsername("nobody")
	if err != nil {
		t.Fatalf("expected no error for a missing user, got %v", err)
	}
	if got != nil {
		t.Errorf("expected nil user, got %+v", got)
	}
}

func TestUserFindById(t *testing.T) {
	repo, _ := newUserRepo(t)

	id, err := repo.Save(newUser("carol"))
	if err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := repo.FindById(id)
	if err != nil {
		t.Fatalf("FindById: %v", err)
	}
	if got == nil || got.Username != "carol" {
		t.Fatalf("got %+v, want user carol", got)
	}
}

func TestUserFindByIdMissing(t *testing.T) {
	repo, _ := newUserRepo(t)

	got, err := repo.FindById(4242)
	if err != nil {
		t.Fatalf("expected no error for a missing id, got %v", err)
	}
	if got != nil {
		t.Errorf("expected nil user, got %+v", got)
	}
}

func TestUserFindByApiKey(t *testing.T) {
	repo, _ := newUserRepo(t)

	u := newUser("dave")
	u.ApiKey = sql.NullString{String: "key-dave", Valid: true}
	if _, err := repo.Save(u); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := repo.FindByApiKey("key-dave")
	if err != nil {
		t.Fatalf("FindByApiKey: %v", err)
	}
	if got == nil || got.Username != "dave" {
		t.Fatalf("got %+v, want user dave", got)
	}

	missing, err := repo.FindByApiKey("no-such-key")
	if err != nil {
		t.Fatalf("expected no error for an unknown key, got %v", err)
	}
	if missing != nil {
		t.Errorf("expected nil for an unknown key, got %+v", missing)
	}
}

func TestUserFindAll(t *testing.T) {
	repo, _ := newUserRepo(t)

	// The initial migration seeds a default admin account, so count from there
	// rather than assuming an empty table.
	baseline, err := repo.FindAll()
	if err != nil {
		t.Fatalf("FindAll baseline: %v", err)
	}
	before := len(*baseline)

	for _, name := range []string{"u1", "u2", "u3"} {
		if _, err := repo.Save(newUser(name)); err != nil {
			t.Fatalf("Save(%s): %v", name, err)
		}
	}

	all, err := repo.FindAll()
	if err != nil {
		t.Fatalf("FindAll: %v", err)
	}
	if all == nil {
		t.Fatal("FindAll returned nil")
	}
	if got, want := len(*all), before+3; got != want {
		t.Errorf("FindAll returned %d users, want %d", got, want)
	}
}

func TestUserFindAllReturnsSeededAdmin(t *testing.T) {
	repo, _ := newUserRepo(t)

	all, err := repo.FindAll()
	if err != nil {
		t.Fatalf("FindAll: %v", err)
	}
	if all == nil || len(*all) == 0 {
		t.Fatal("expected the migration-seeded admin user, got none")
	}

	var found bool
	for _, u := range *all {
		if u.Username == "admin" {
			found = true
		}
	}
	if !found {
		t.Errorf("seeded admin user not present in %+v", *all)
	}
}

func TestUserSessionLifecycle(t *testing.T) {
	repo, clock := newUserRepo(t)

	id, err := repo.Save(newUser("erin"))
	if err != nil {
		t.Fatalf("Save: %v", err)
	}

	expiry := clock.Now().Add(time.Hour)
	if err := repo.UpdateSession(id, "sess-erin", expiry); err != nil {
		t.Fatalf("UpdateSession: %v", err)
	}

	got, err := repo.FindBySessionID("sess-erin", clock.Now())
	if err != nil {
		t.Fatalf("FindBySessionID: %v", err)
	}
	if got == nil || got.ID != id {
		t.Fatalf("got %+v, want the user with id %d", got, id)
	}

	if err := repo.ClearSessionBySessionID("sess-erin"); err != nil {
		t.Fatalf("ClearSessionBySessionID: %v", err)
	}

	cleared, err := repo.FindBySessionID("sess-erin", clock.Now())
	if err != nil {
		t.Fatalf("FindBySessionID after clear: %v", err)
	}
	if cleared != nil {
		t.Errorf("session still resolves after being cleared: %+v", cleared)
	}
}

func TestUserFindBySessionIDRejectsExpiredSession(t *testing.T) {
	repo, clock := newUserRepo(t)

	id, err := repo.Save(newUser("frank"))
	if err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Session expired an hour ago.
	if err := repo.UpdateSession(id, "sess-frank", clock.Now().Add(-time.Hour)); err != nil {
		t.Fatalf("UpdateSession: %v", err)
	}

	got, err := repo.FindBySessionID("sess-frank", clock.Now())
	if err != nil {
		t.Fatalf("FindBySessionID: %v", err)
	}
	if got != nil {
		t.Errorf("an expired session was accepted: %+v", got)
	}
}

func TestUserFindBySessionIDUnknown(t *testing.T) {
	repo, clock := newUserRepo(t)

	got, err := repo.FindBySessionID("no-such-session", clock.Now())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if got != nil {
		t.Errorf("expected nil, got %+v", got)
	}
}

func TestUserUpdateUser(t *testing.T) {
	repo, _ := newUserRepo(t)

	id, err := repo.Save(newUser("grace"))
	if err != nil {
		t.Fatalf("Save: %v", err)
	}

	err = repo.UpdateUser(id, "grace-renamed",
		sql.NullString{String: "new-key", Valid: true},
		sql.NullBool{Bool: false, Valid: true})
	if err != nil {
		t.Fatalf("UpdateUser: %v", err)
	}

	got, err := repo.FindById(id)
	if err != nil {
		t.Fatalf("FindById: %v", err)
	}
	if got == nil {
		t.Fatal("user disappeared after update")
	}
	if got.Username != "grace-renamed" {
		t.Errorf("username = %q, want %q", got.Username, "grace-renamed")
	}
	if got.ApiKey.String != "new-key" {
		t.Errorf("api key = %q, want %q", got.ApiKey.String, "new-key")
	}
	if got.Enabled.Valid && got.Enabled.Bool {
		t.Error("user should be disabled after the update")
	}
}

func TestUserDeleteById(t *testing.T) {
	repo, _ := newUserRepo(t)

	id, err := repo.Save(newUser("heidi"))
	if err != nil {
		t.Fatalf("Save: %v", err)
	}

	if err := repo.DeleteById(id); err != nil {
		t.Fatalf("DeleteById: %v", err)
	}

	got, err := repo.FindById(id)
	if err != nil {
		t.Fatalf("FindById after delete: %v", err)
	}
	if got != nil {
		t.Errorf("user still present after delete: %+v", got)
	}
}

func TestUserDeleteByIdUnknownIsNotAnError(t *testing.T) {
	repo, _ := newUserRepo(t)

	if err := repo.DeleteById(9999); err != nil {
		t.Errorf("deleting an unknown id returned %v, want nil", err)
	}
}

func TestUserSaveDuplicateUsernameFails(t *testing.T) {
	repo, _ := newUserRepo(t)

	if _, err := repo.Save(newUser("ivan")); err != nil {
		t.Fatalf("first Save: %v", err)
	}
	if _, err := repo.Save(newUser("ivan")); err == nil {
		t.Error("saving a duplicate username succeeded; the unique constraint is not enforced")
	}
}
