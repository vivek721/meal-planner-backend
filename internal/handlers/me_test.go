package handlers_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetMe(t *testing.T) {
	s := newTestServer(t)
	user, token := s.seedUser(t, "me@example.com")

	code, resp := s.do(t, http.MethodGet, "/api/auth/me", nil, token)
	if code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (%v)", code, resp)
	}
	if got := userField(t, resp, "id"); got != user.ID {
		t.Errorf("user.id = %v, want %q", got, user.ID)
	}
	if got := userField(t, resp, "email"); got != "me@example.com" {
		t.Errorf("user.email = %v, want me@example.com", got)
	}
	if _, leaked := resp["user"].(map[string]any)["passwordHash"]; leaked {
		t.Error("response leaks passwordHash")
	}
}

func TestGetMe_UserDeleted(t *testing.T) {
	s := newTestServer(t)
	user, token := s.seedUser(t, "gone@example.com")
	delete(s.repo.Users, user.ID)

	code, resp := s.do(t, http.MethodGet, "/api/auth/me", nil, token)
	if code != http.StatusUnauthorized || resp["error"] != "invalid token" {
		t.Errorf("got %d %v, want 401 invalid token", code, resp)
	}
}

func TestGetMe_RepositoryError(t *testing.T) {
	s := newTestServer(t)
	_, token := s.seedUser(t, "me@example.com")
	s.repo.Err = errors.New("db down")

	code, _ := s.do(t, http.MethodGet, "/api/auth/me", nil, token)
	if code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", code)
	}
}

func TestGetMe_RequiresValidBearerToken(t *testing.T) {
	s := newTestServer(t)
	s.seedUser(t, "me@example.com")

	for _, header := range []string{"", "Bearer", "Basic abc", "Bearer not-a-jwt", "x"} {
		req := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
		if header != "" {
			req.Header.Set("Authorization", header)
		}
		w := httptest.NewRecorder()
		s.engine.ServeHTTP(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Errorf("Authorization %q: status = %d, want 401", header, w.Code)
		}
	}
}
