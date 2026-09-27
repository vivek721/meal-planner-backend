package handlers_test

import (
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/meal-planner/backend/internal/utils"
)

func TestRegister(t *testing.T) {
	s := newTestServer(t)

	code, resp := s.do(t, http.MethodPost, "/api/auth/register", map[string]string{
		"email": "new@example.com", "password": testPassword, "name": "New",
	}, "")
	if code != http.StatusCreated {
		t.Fatalf("status = %d, want 201 (%v)", code, resp)
	}
	token, _ := resp["token"].(string)
	if _, err := utils.ValidateToken(token, testSecret); err != nil {
		t.Errorf("returned token is invalid: %v", err)
	}
	if got := userField(t, resp, "email"); got != "new@example.com" {
		t.Errorf("user.email = %v", got)
	}

	tests := []struct {
		name     string
		body     any
		wantCode int
		wantErr  string
	}{
		{"duplicate", map[string]string{"email": "new@example.com", "password": testPassword}, http.StatusConflict, "Email already exists"},
		{"invalid email", map[string]string{"email": "nope", "password": testPassword}, http.StatusBadRequest, "invalid email format"},
		{"short password", map[string]string{"email": "b@example.com", "password": "Aa1!"}, http.StatusBadRequest, "password must be at least 8 characters"},
		{"weak password", map[string]string{"email": "b@example.com", "password": "password"}, http.StatusBadRequest, "password must contain uppercase, lowercase, number, and special character"},
		{"missing fields", map[string]string{}, http.StatusBadRequest, "invalid request body"},
		{"malformed JSON", "{", http.StatusBadRequest, "invalid request body"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, resp := s.do(t, http.MethodPost, "/api/auth/register", tt.body, "")
			if code != tt.wantCode || resp["error"] != tt.wantErr {
				t.Errorf("got %d %v, want %d %q", code, resp, tt.wantCode, tt.wantErr)
			}
		})
	}
}

func TestRegister_RepositoryError(t *testing.T) {
	s := newTestServer(t)
	s.repo.Err = errors.New("db down")

	code, resp := s.do(t, http.MethodPost, "/api/auth/register", map[string]string{
		"email": "new@example.com", "password": testPassword,
	}, "")
	if code != http.StatusInternalServerError || resp["error"] != "failed to register user" {
		t.Errorf("got %d %v, want 500 failed to register user", code, resp)
	}
}

func TestLogin(t *testing.T) {
	s := newTestServer(t)
	user, _ := s.seedUser(t, "user@example.com")

	code, resp := s.do(t, http.MethodPost, "/api/auth/login", map[string]string{
		"email": "user@example.com", "password": testPassword,
	}, "")
	if code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (%v)", code, resp)
	}
	if got := userField(t, resp, "id"); got != user.ID {
		t.Errorf("user.id = %v, want %s", got, user.ID)
	}

	code, resp = s.do(t, http.MethodPost, "/api/auth/login", map[string]string{
		"email": "user@example.com", "password": "Wrong1!!",
	}, "")
	if code != http.StatusUnauthorized || resp["error"] != "Invalid email or password" {
		t.Errorf("wrong password: got %d %v", code, resp)
	}

	code, _ = s.do(t, http.MethodPost, "/api/auth/login", "{", "")
	if code != http.StatusBadRequest {
		t.Errorf("malformed body: status = %d, want 400", code)
	}
}

func TestLogin_Lockout(t *testing.T) {
	s := newTestServer(t)
	s.seedUser(t, "user@example.com")
	wrong := map[string]string{"email": "user@example.com", "password": "Wrong1!!"}

	for i := 0; i < 2; i++ {
		if code, _ := s.do(t, http.MethodPost, "/api/auth/login", wrong, ""); code != http.StatusUnauthorized {
			t.Fatalf("attempt %d: status = %d, want 401", i+1, code)
		}
	}

	code, resp := s.do(t, http.MethodPost, "/api/auth/login", wrong, "")
	if code != http.StatusForbidden || resp["error"] != "account is locked due to too many failed login attempts" {
		t.Errorf("third attempt: got %d %v, want 403 lock message", code, resp)
	}

	code, resp = s.do(t, http.MethodPost, "/api/auth/login", map[string]string{
		"email": "user@example.com", "password": testPassword,
	}, "")
	want := "account is locked. Please try again in 5 minute(s)"
	if code != http.StatusForbidden || resp["error"] != want {
		t.Errorf("login while locked: got %d %v, want 403 %q", code, resp, want)
	}
}

func TestLogin_ShortErrorMessageDoesNotPanic(t *testing.T) {
	// The handler used to slice err.Error()[:15], panicking on short errors.
	s := newTestServer(t)
	s.repo.Err = errors.New("EOF")

	code, resp := s.do(t, http.MethodPost, "/api/auth/login", map[string]string{
		"email": "user@example.com", "password": testPassword,
	}, "")
	if code != http.StatusUnauthorized || resp["error"] != "Invalid email or password" {
		t.Errorf("got %d %v, want 401 Invalid email or password", code, resp)
	}
}

func TestRefreshToken(t *testing.T) {
	s := newTestServer(t)
	user, token := s.seedUser(t, "user@example.com")

	code, resp := s.do(t, http.MethodPost, "/api/auth/refresh", map[string]string{"token": token}, "")
	if code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (%v)", code, resp)
	}
	newToken, _ := resp["token"].(string)
	claims, err := utils.ValidateToken(newToken, testSecret)
	if err != nil || claims.UserID != user.ID {
		t.Errorf("refreshed token: claims=%v err=%v", claims, err)
	}

	expired, _ := utils.GenerateToken(user.ID, user.Email, testSecret, -time.Minute)
	code, resp = s.do(t, http.MethodPost, "/api/auth/refresh", map[string]string{"token": expired}, "")
	if code != http.StatusUnauthorized || resp["error"] != "invalid or expired token" {
		t.Errorf("expired token: got %d %v", code, resp)
	}

	if code, _ := s.do(t, http.MethodPost, "/api/auth/refresh", map[string]string{}, ""); code != http.StatusBadRequest {
		t.Errorf("missing token: status = %d, want 400", code)
	}
}

func TestLogout(t *testing.T) {
	s := newTestServer(t)
	_, token := s.seedUser(t, "user@example.com")

	code, resp := s.do(t, http.MethodPost, "/api/auth/logout", nil, token)
	if code != http.StatusOK || !strings.Contains(resp["message"].(string), "logged out") {
		t.Errorf("got %d %v, want 200 logged out", code, resp)
	}
}
