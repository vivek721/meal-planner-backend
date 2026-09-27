package router

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/meal-planner/backend/internal/config"
	"github.com/meal-planner/backend/internal/testutil"
)

func get(t *testing.T, engine *gin.Engine, path string) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, httptest.NewRequestWithContext(t.Context(), http.MethodGet, path, http.NoBody))
	return w
}

func TestPublicEndpoints(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := New(testutil.NewUserRepo(), testConfig("test"))

	w := get(t, engine, "/health")
	if w.Code != http.StatusOK || w.Body.String() != `{"service":"meal-planner-api","status":"healthy"}` {
		t.Errorf("/health: got %d %s", w.Code, w.Body.String())
	}

	w = get(t, engine, "/api")
	var info struct {
		Endpoints struct {
			Auth map[string]string `json:"auth"`
		} `json:"endpoints"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &info); err != nil || w.Code != http.StatusOK {
		t.Fatalf("/api: got %d %s", w.Code, w.Body.String())
	}
	if len(info.Endpoints.Auth) != 9 {
		t.Errorf("/api lists %d auth endpoints, want 9", len(info.Endpoints.Auth))
	}

	if w := get(t, engine, "/nope"); w.Code != http.StatusNotFound {
		t.Errorf("/nope: status = %d, want 404", w.Code)
	}
}

func TestProtectedRoutesRequireAuth(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := Setup(nil, testConfig("test"))

	routes := []struct{ method, path string }{
		{http.MethodGet, "/api/auth/me"},
		{http.MethodPost, "/api/auth/logout"},
		{http.MethodPut, "/api/auth/profile"},
		{http.MethodPut, "/api/auth/password"},
		{http.MethodPut, "/api/auth/preferences"},
		{http.MethodPost, "/api/auth/onboarding/complete"},
	}
	for _, r := range routes {
		w := httptest.NewRecorder()
		engine.ServeHTTP(w, httptest.NewRequestWithContext(t.Context(), r.method, r.path, http.NoBody))
		if w.Code != http.StatusUnauthorized {
			t.Errorf("%s %s without token: status = %d, want 401", r.method, r.path, w.Code)
		}
	}
}

func TestProductionSetsReleaseMode(t *testing.T) {
	t.Cleanup(func() { gin.SetMode(gin.TestMode) })
	New(testutil.NewUserRepo(), testConfig("production"))
	if gin.Mode() != gin.ReleaseMode {
		t.Errorf("gin.Mode() = %q, want release", gin.Mode())
	}
}

func testConfig(env string) *config.Config {
	return &config.Config{
		Environment:        env,
		JWTSecret:          "test-secret-key-for-unit-tests-32-chars",
		CORSAllowedOrigins: []string{"http://localhost:3000"},
	}
}
