package services

import (
	"errors"
	"testing"
	"time"

	"github.com/meal-planner/backend/internal/utils"
)

func TestAccountLockedError(t *testing.T) {
	tests := []struct {
		name      string
		remaining time.Duration
		want      string
	}{
		{"full lock duration", LockDuration, "account is locked. Please try again in 5 minute(s)"},
		{"partial minute rounds up", 4*time.Minute + time.Second, "account is locked. Please try again in 5 minute(s)"},
		{"under a minute", 10 * time.Second, "account is locked. Please try again in 1 minute(s)"},
		{"already expired", -time.Second, "account is locked. Please try again in 1 minute(s)"},
		{"double digits", 12 * time.Minute, "account is locked. Please try again in 12 minute(s)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := &AccountLockedError{Remaining: tt.remaining}
			if got := err.Error(); got != tt.want {
				t.Errorf("Error() = %q, want %q", got, tt.want)
			}
			if !errors.Is(err, ErrAccountLocked) {
				t.Error("errors.Is(err, ErrAccountLocked) = false, want true")
			}
		})
	}
}

func TestLogin_LockedAccountMessage(t *testing.T) {
	repo := newFakeUserRepo()
	user := seedUser(repo, "locked@example.com")
	lockedUntil := time.Now().Add(3*time.Minute + 30*time.Second)
	user.AccountLockedUntil = &lockedUntil

	svc := NewAuthService(repo, testConfig())
	_, _, err := svc.Login("locked@example.com", testPassword)

	var lockedErr *AccountLockedError
	if !errors.As(err, &lockedErr) {
		t.Fatalf("Login() error = %v, want *AccountLockedError", err)
	}
	want := "account is locked. Please try again in 4 minute(s)"
	if err.Error() != want {
		t.Errorf("Login() error = %q, want %q", err.Error(), want)
	}
}

func TestLogin_LocksAfterMaxAttempts(t *testing.T) {
	repo := newFakeUserRepo()
	seedUser(repo, "user@example.com")
	svc := NewAuthService(repo, testConfig())

	for i := 1; i < MaxLoginAttempts; i++ {
		_, _, err := svc.Login("user@example.com", "wrong")
		if !errors.Is(err, ErrInvalidCredentials) {
			t.Fatalf("attempt %d: error = %v, want ErrInvalidCredentials", i, err)
		}
	}

	_, _, err := svc.Login("user@example.com", "wrong")
	if !errors.Is(err, ErrAccountLocked) {
		t.Fatalf("attempt %d: error = %v, want ErrAccountLocked", MaxLoginAttempts, err)
	}

	// Even the correct password is rejected while locked.
	_, _, err = svc.Login("user@example.com", testPassword)
	var lockedErr *AccountLockedError
	if !errors.As(err, &lockedErr) {
		t.Fatalf("login while locked: error = %v, want *AccountLockedError", err)
	}
	if got := lockedErr.Minutes(); got != 5 {
		t.Errorf("Minutes() = %d, want 5", got)
	}
}

func TestLogin_Success(t *testing.T) {
	repo := newFakeUserRepo()
	user := seedUser(repo, "user@example.com")
	user.LoginAttempts = 2
	svc := NewAuthService(repo, testConfig())

	got, token, err := svc.Login("  USER@example.com ", testPassword)
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	if got.ID != user.ID {
		t.Errorf("Login() user ID = %q, want %q", got.ID, user.ID)
	}
	if got.LoginAttempts != 0 {
		t.Errorf("LoginAttempts = %d, want reset to 0", got.LoginAttempts)
	}
	claims, err := utils.ValidateToken(token, testSecret)
	if err != nil {
		t.Fatalf("returned token is invalid: %v", err)
	}
	if claims.UserID != user.ID {
		t.Errorf("token UserID = %q, want %q", claims.UserID, user.ID)
	}
}

func TestLogin_Errors(t *testing.T) {
	repo := newFakeUserRepo()
	svc := NewAuthService(repo, testConfig())

	if _, _, err := svc.Login("nobody@example.com", testPassword); !errors.Is(err, ErrInvalidCredentials) {
		t.Errorf("unknown user: error = %v, want ErrInvalidCredentials", err)
	}

	repo.err = errRepo
	if _, _, err := svc.Login("nobody@example.com", testPassword); !errors.Is(err, errRepo) {
		t.Errorf("repo failure: error = %v, want errRepo", err)
	}
}

func TestRegister(t *testing.T) {
	repo := newFakeUserRepo()
	svc := NewAuthService(repo, testConfig())

	user, token, err := svc.Register(" New@Example.com ", testPassword, "New User")
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	if user.Email != "new@example.com" {
		t.Errorf("email = %q, want normalized %q", user.Email, "new@example.com")
	}
	if user.PasswordHash == testPassword || !utils.VerifyPassword(testPassword, user.PasswordHash) {
		t.Error("password was not hashed correctly")
	}
	if _, err := utils.ValidateToken(token, testSecret); err != nil {
		t.Errorf("returned token is invalid: %v", err)
	}

	tests := []struct {
		name     string
		email    string
		password string
		wantErr  error
	}{
		{"duplicate email", "new@example.com", testPassword, ErrUserAlreadyExists},
		{"invalid email", "not-an-email", testPassword, utils.ErrInvalidEmail},
		{"weak password", "other@example.com", "password", utils.ErrPasswordTooWeak},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, _, err := svc.Register(tt.email, tt.password, ""); !errors.Is(err, tt.wantErr) {
				t.Errorf("Register() error = %v, want %v", err, tt.wantErr)
			}
		})
	}

	repo.err = errRepo
	if _, _, err := svc.Register("x@example.com", testPassword, ""); !errors.Is(err, errRepo) {
		t.Errorf("repo failure: error = %v, want errRepo", err)
	}
}

func TestRefreshToken(t *testing.T) {
	svc := NewAuthService(newFakeUserRepo(), testConfig())

	token, err := utils.GenerateToken("user_1", "a@example.com", testSecret, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	newToken, err := svc.RefreshToken(token)
	if err != nil {
		t.Fatalf("RefreshToken() error = %v", err)
	}
	claims, err := utils.ValidateToken(newToken, testSecret)
	if err != nil {
		t.Fatalf("refreshed token is invalid: %v", err)
	}
	if claims.UserID != "user_1" || claims.Email != "a@example.com" {
		t.Errorf("refreshed claims = %+v, want same user", claims)
	}

	if _, err := svc.RefreshToken("garbage"); !errors.Is(err, utils.ErrInvalidToken) {
		t.Errorf("RefreshToken(garbage) error = %v, want ErrInvalidToken", err)
	}
}

func TestValidateToken(t *testing.T) {
	repo := newFakeUserRepo()
	user := seedUser(repo, "user@example.com")
	svc := NewAuthService(repo, testConfig())

	token, _ := utils.GenerateToken(user.ID, user.Email, testSecret, time.Hour)
	got, err := svc.ValidateToken(token)
	if err != nil || got.ID != user.ID {
		t.Fatalf("ValidateToken() = %v, %v; want user %q", got, err, user.ID)
	}

	orphan, _ := utils.GenerateToken("missing", "m@example.com", testSecret, time.Hour)
	if _, err := svc.ValidateToken(orphan); !errors.Is(err, ErrUserNotFound) {
		t.Errorf("unknown user: error = %v, want ErrUserNotFound", err)
	}

	if _, err := svc.ValidateToken("garbage"); !errors.Is(err, utils.ErrInvalidToken) {
		t.Errorf("garbage token: error = %v, want ErrInvalidToken", err)
	}
}
