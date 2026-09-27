package services

import (
	"errors"
	"testing"

	"github.com/meal-planner/backend/internal/models"
	"github.com/meal-planner/backend/internal/testutil"
	"github.com/meal-planner/backend/internal/utils"
)

func TestGetUserByID(t *testing.T) {
	repo := testutil.NewUserRepo()
	user := seedUser(repo, "a@example.com")
	svc := NewUserService(repo, testConfig())

	got, err := svc.GetUserByID(user.ID)
	if err != nil || got.ID != user.ID {
		t.Fatalf("GetUserByID() = %v, %v", got, err)
	}
	if _, err := svc.GetUserByID("missing"); !errors.Is(err, ErrUserNotFound) {
		t.Errorf("missing user: error = %v, want ErrUserNotFound", err)
	}
	repo.Err = errRepo
	if _, err := svc.GetUserByID(user.ID); !errors.Is(err, errRepo) {
		t.Errorf("repo failure: error = %v, want errRepo", err)
	}
}

func TestUpdateProfile(t *testing.T) {
	repo := testutil.NewUserRepo()
	user := seedUser(repo, "a@example.com")
	seedUser(repo, "taken@example.com")
	svc := NewUserService(repo, testConfig())

	got, err := svc.UpdateProfile(user.ID, "New Name", " New@Example.com ")
	if err != nil {
		t.Fatalf("UpdateProfile() error = %v", err)
	}
	if got.Name != "New Name" || got.Email != "new@example.com" {
		t.Errorf("UpdateProfile() = %q %q, want New Name new@example.com", got.Name, got.Email)
	}

	// Empty fields leave the profile unchanged.
	got, err = svc.UpdateProfile(user.ID, "", "")
	if err != nil || got.Name != "New Name" || got.Email != "new@example.com" {
		t.Errorf("no-op update = %+v, %v", got, err)
	}

	tests := []struct {
		name    string
		userID  string
		email   string
		wantErr error
	}{
		{"email taken", user.ID, "taken@example.com", ErrUserAlreadyExists},
		{"invalid email", user.ID, "nope", utils.ErrInvalidEmail},
		{"unknown user", "missing", "", ErrUserNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := svc.UpdateProfile(tt.userID, "", tt.email); !errors.Is(err, tt.wantErr) {
				t.Errorf("UpdateProfile() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestChangePassword(t *testing.T) {
	repo := testutil.NewUserRepo()
	user := seedUser(repo, "a@example.com")
	svc := NewUserService(repo, testConfig())

	tests := []struct {
		name    string
		userID  string
		current string
		next    string
		wantErr error
	}{
		{"wrong current password", user.ID, "Wrong1!!", "NewPassword1!", ErrCurrentPasswordIncorrect},
		{"weak new password", user.ID, testPassword, "short", utils.ErrPasswordTooShort},
		{"unknown user", "missing", testPassword, "NewPassword1!", ErrUserNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := svc.ChangePassword(tt.userID, tt.current, tt.next); !errors.Is(err, tt.wantErr) {
				t.Errorf("ChangePassword() error = %v, want %v", err, tt.wantErr)
			}
		})
	}

	if err := svc.ChangePassword(user.ID, testPassword, "NewPassword1!"); err != nil {
		t.Fatalf("ChangePassword() error = %v", err)
	}
	if !utils.VerifyPassword("NewPassword1!", repo.Users[user.ID].PasswordHash) {
		t.Error("new password was not stored")
	}
}

func TestCompleteOnboarding(t *testing.T) {
	repo := testutil.NewUserRepo()
	user := seedUser(repo, "a@example.com")
	svc := NewUserService(repo, testConfig())

	got, err := svc.CompleteOnboarding(user.ID)
	if err != nil || !got.HasCompletedOnboarding {
		t.Fatalf("CompleteOnboarding() = %+v, %v", got, err)
	}
	if _, err := svc.CompleteOnboarding("missing"); !errors.Is(err, ErrUserNotFound) {
		t.Errorf("unknown user: error = %v, want ErrUserNotFound", err)
	}
}

func TestUpdatePreferences(t *testing.T) {
	repo := testutil.NewUserRepo()
	user := seedUser(repo, "a@example.com")
	svc := NewUserService(repo, testConfig())

	prefs := &models.UserPreferences{Theme: "dark", Notifications: true}
	got, err := svc.UpdatePreferences(user.ID, prefs)
	if err != nil || got.Preferences != prefs {
		t.Fatalf("UpdatePreferences() = %+v, %v", got, err)
	}
	if _, err := svc.UpdatePreferences("missing", prefs); !errors.Is(err, ErrUserNotFound) {
		t.Errorf("unknown user: error = %v, want ErrUserNotFound", err)
	}
}
