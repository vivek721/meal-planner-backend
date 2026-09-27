package handlers_test

import (
	"errors"
	"net/http"
	"testing"

	"github.com/meal-planner/backend/internal/utils"
)

func TestUpdateProfile(t *testing.T) {
	s := newTestServer(t)
	_, token := s.seedUser(t, "user@example.com")
	s.seedUser(t, "taken@example.com")

	code, resp := s.do(t, http.MethodPut, "/api/auth/profile", map[string]string{
		"name": "Renamed", "email": "renamed@example.com",
	}, token)
	if code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (%v)", code, resp)
	}
	if userField(t, resp, "name") != "Renamed" || userField(t, resp, "email") != "renamed@example.com" {
		t.Errorf("user = %v", resp["user"])
	}

	tests := []struct {
		name     string
		body     any
		wantCode int
		wantErr  string
	}{
		{"email taken", map[string]string{"email": "taken@example.com"}, http.StatusConflict, "email already in use"},
		{"invalid email", map[string]string{"email": "nope"}, http.StatusBadRequest, "invalid email format"},
		{"malformed JSON", "{", http.StatusBadRequest, "invalid request body"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, resp := s.do(t, http.MethodPut, "/api/auth/profile", tt.body, token)
			if code != tt.wantCode || resp["error"] != tt.wantErr {
				t.Errorf("got %d %v, want %d %q", code, resp, tt.wantCode, tt.wantErr)
			}
		})
	}
}

func TestUpdateProfile_UserNotFound(t *testing.T) {
	s := newTestServer(t)
	user, token := s.seedUser(t, "user@example.com")
	delete(s.repo.Users, user.ID)

	code, resp := s.do(t, http.MethodPut, "/api/auth/profile", map[string]string{"name": "x"}, token)
	if code != http.StatusNotFound || resp["error"] != "user not found" {
		t.Errorf("got %d %v, want 404 user not found", code, resp)
	}
}

func TestChangePassword(t *testing.T) {
	s := newTestServer(t)
	user, token := s.seedUser(t, "user@example.com")

	tests := []struct {
		name     string
		body     any
		wantCode int
		wantErr  string
	}{
		{"wrong current", map[string]string{"currentPassword": "Wrong1!!", "newPassword": "NewPassword1!"}, http.StatusBadRequest, "current password is incorrect"},
		{"too short", map[string]string{"currentPassword": testPassword, "newPassword": "Aa1!"}, http.StatusBadRequest, "password must be at least 8 characters"},
		{"too weak", map[string]string{"currentPassword": testPassword, "newPassword": "password"}, http.StatusBadRequest, "password must contain uppercase, lowercase, number, and special character"},
		{"missing fields", map[string]string{}, http.StatusBadRequest, "invalid request body"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, resp := s.do(t, http.MethodPut, "/api/auth/password", tt.body, token)
			if code != tt.wantCode || resp["error"] != tt.wantErr {
				t.Errorf("got %d %v, want %d %q", code, resp, tt.wantCode, tt.wantErr)
			}
		})
	}

	code, resp := s.do(t, http.MethodPut, "/api/auth/password", map[string]string{
		"currentPassword": testPassword, "newPassword": "NewPassword1!",
	}, token)
	if code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (%v)", code, resp)
	}
	if !utils.VerifyPassword("NewPassword1!", s.repo.Users[user.ID].PasswordHash) {
		t.Error("new password was not stored")
	}
}

func TestCompleteOnboarding(t *testing.T) {
	s := newTestServer(t)
	_, token := s.seedUser(t, "user@example.com")

	code, resp := s.do(t, http.MethodPost, "/api/auth/onboarding/complete", nil, token)
	if code != http.StatusOK || userField(t, resp, "hasCompletedOnboarding") != true {
		t.Errorf("got %d %v, want 200 with hasCompletedOnboarding=true", code, resp)
	}

	s.repo.Err = errors.New("db down")
	if code, _ := s.do(t, http.MethodPost, "/api/auth/onboarding/complete", nil, token); code != http.StatusInternalServerError {
		t.Errorf("repo failure: status = %d, want 500", code)
	}
}

func TestUpdatePreferences(t *testing.T) {
	s := newTestServer(t)
	_, token := s.seedUser(t, "user@example.com")

	code, resp := s.do(t, http.MethodPut, "/api/auth/preferences", map[string]any{
		"theme": "dark", "notifications": true,
	}, token)
	if code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (%v)", code, resp)
	}
	prefs, _ := userField(t, resp, "preferences").(map[string]any)
	if prefs["theme"] != "dark" || prefs["notifications"] != true {
		t.Errorf("preferences = %v", prefs)
	}

	if code, _ := s.do(t, http.MethodPut, "/api/auth/preferences", "{", token); code != http.StatusBadRequest {
		t.Errorf("malformed body: status = %d, want 400", code)
	}

	s.repo.Err = errors.New("db down")
	if code, _ := s.do(t, http.MethodPut, "/api/auth/preferences", map[string]any{}, token); code != http.StatusInternalServerError {
		t.Errorf("repo failure: status = %d, want 500", code)
	}
}
