package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/meal-planner/backend/internal/config"
	"github.com/meal-planner/backend/internal/models"
	"github.com/meal-planner/backend/internal/router"
	"github.com/meal-planner/backend/internal/testutil"
	"github.com/meal-planner/backend/internal/utils"
)

const (
	testSecret   = "test-secret-key-for-unit-tests-32-chars"
	testPassword = "Password1!"
)

func init() {
	gin.SetMode(gin.TestMode)
}

type testServer struct {
	engine *gin.Engine
	repo   *testutil.UserRepo
}

func newTestServer(t *testing.T) *testServer {
	t.Helper()
	cfg := &config.Config{
		Environment:        "test",
		JWTSecret:          testSecret,
		JWTExpirationHours: 1,
		BcryptCost:         4,
		CORSAllowedOrigins: []string{"http://localhost:3000"},
	}
	repo := testutil.NewUserRepo()
	return &testServer{engine: router.New(repo, cfg), repo: repo}
}

// seedUser stores a user with testPassword and returns it with a valid token.
func (s *testServer) seedUser(t *testing.T, email string) (user *models.User, token string) {
	t.Helper()
	hash, err := utils.HashPassword(testPassword, 4)
	if err != nil {
		t.Fatal(err)
	}
	user = &models.User{Email: email, Name: "Test User", PasswordHash: hash}
	if err := s.repo.Create(user); err != nil {
		t.Fatal(err)
	}
	token, err = utils.GenerateToken(user.ID, user.Email, testSecret, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	return user, token
}

// do sends a request; body is JSON-encoded unless it is a string, and an
// empty token means no Authorization header.
func (s *testServer) do(t *testing.T, method, path string, body any, token string) (status int, resp map[string]any) {
	t.Helper()
	var buf bytes.Buffer
	switch b := body.(type) {
	case nil:
	case string:
		buf.WriteString(b)
	default:
		if err := json.NewEncoder(&buf).Encode(b); err != nil {
			t.Fatal(err)
		}
	}
	req := httptest.NewRequestWithContext(t.Context(), method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	s.engine.ServeHTTP(w, req)

	if w.Body.Len() > 0 {
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("response is not JSON: %q", w.Body.String())
		}
	}
	return w.Code, resp
}

func userField(t *testing.T, resp map[string]any, field string) any {
	t.Helper()
	user, ok := resp["user"].(map[string]any)
	if !ok {
		t.Fatalf("response has no user object: %v", resp)
	}
	return user[field]
}
