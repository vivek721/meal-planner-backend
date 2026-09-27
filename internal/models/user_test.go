package models

import (
	"testing"
	"time"
)

func TestBeforeCreate(t *testing.T) {
	u := &User{Email: "a@example.com"}
	if err := u.BeforeCreate(nil); err != nil {
		t.Fatalf("BeforeCreate() error = %v", err)
	}
	if !idPattern.MatchString(u.ID) {
		t.Errorf("ID = %q, want generated user ID", u.ID)
	}
	if u.CreatedAt.IsZero() {
		t.Error("CreatedAt was not set")
	}

	created := time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC)
	existing := &User{ID: "user_fixed", CreatedAt: created}
	if err := existing.BeforeCreate(nil); err != nil {
		t.Fatal(err)
	}
	if existing.ID != "user_fixed" || !existing.CreatedAt.Equal(created) {
		t.Errorf("BeforeCreate overwrote preset fields: %+v", existing)
	}
}

func TestToPublicUser(t *testing.T) {
	prefs := &UserPreferences{Theme: "dark", Notifications: true}
	u := &User{
		ID:                     "user_1",
		Email:                  "a@example.com",
		Name:                   "Alice",
		PasswordHash:           "secret-hash",
		HasCompletedOnboarding: true,
		CreatedAt:              time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC),
		Preferences:            prefs,
	}

	got := u.ToPublicUser()
	want := PublicUser{
		ID:                     "user_1",
		Email:                  "a@example.com",
		Name:                   "Alice",
		HasCompletedOnboarding: true,
		CreatedAt:              "2024-01-02T03:04:05Z",
		Preferences:            prefs,
	}
	if *got != want {
		t.Errorf("ToPublicUser() = %+v, want %+v", *got, want)
	}
}

func TestLoginAttempts(t *testing.T) {
	u := &User{}
	if u.IsAccountLocked() {
		t.Fatal("new user is locked")
	}

	u.IncrementLoginAttempts(3, time.Minute)
	u.IncrementLoginAttempts(3, time.Minute)
	if u.IsAccountLocked() {
		t.Fatal("locked after 2 of 3 attempts")
	}
	if u.LoginAttempts != 2 || u.LastLoginAttempt == nil {
		t.Errorf("after 2 attempts: %+v", u.GetLoginAttemptInfo())
	}

	u.IncrementLoginAttempts(3, time.Minute)
	if !u.IsAccountLocked() {
		t.Fatal("not locked after 3 of 3 attempts")
	}
	info := u.GetLoginAttemptInfo()
	if info.Count != 3 || info.LockedUntil == nil {
		t.Errorf("GetLoginAttemptInfo() = %+v", info)
	}

	u.ResetLoginAttempts()
	if u.IsAccountLocked() || u.LoginAttempts != 0 || u.AccountLockedUntil != nil {
		t.Errorf("after reset: %+v", u.GetLoginAttemptInfo())
	}
}

func TestIsAccountLocked_Expired(t *testing.T) {
	past := time.Now().Add(-time.Second)
	u := &User{AccountLockedUntil: &past}
	if u.IsAccountLocked() {
		t.Error("IsAccountLocked() = true for a lock that has expired")
	}
}
