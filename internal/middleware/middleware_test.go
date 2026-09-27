package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/meal-planner/backend/internal/config"
	"github.com/meal-planner/backend/internal/utils"
)

const testSecret = "test-secret-key-for-unit-tests-32-chars"

func init() {
	gin.SetMode(gin.TestMode)
}

func serve(t *testing.T, engine *gin.Engine, req *http.Request) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)
	return w
}

func TestAuthMiddleware(t *testing.T) {
	cfg := &config.Config{JWTSecret: testSecret}
	engine := gin.New()
	engine.GET("/protected", AuthMiddleware(cfg), func(c *gin.Context) {
		userID, ok := GetUserID(c)
		if !ok {
			t.Error("GetUserID() ok = false inside protected handler")
		}
		c.String(http.StatusOK, userID)
	})

	valid, err := utils.GenerateToken("user_1", "a@example.com", testSecret, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	expired, err := utils.GenerateToken("user_1", "a@example.com", testSecret, -time.Minute)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name     string
		header   string
		wantCode int
		wantBody string
	}{
		{"valid token", "Bearer " + valid, http.StatusOK, "user_1"},
		{"missing header", "", http.StatusUnauthorized, `{"error":"authorization header is required"}`},
		{"wrong scheme", "Basic " + valid, http.StatusUnauthorized, `{"error":"invalid authorization header format"}`},
		{"no token", "Bearer", http.StatusUnauthorized, `{"error":"invalid authorization header format"}`},
		{"extra parts", "Bearer a b", http.StatusUnauthorized, `{"error":"invalid authorization header format"}`},
		{"expired token", "Bearer " + expired, http.StatusUnauthorized, `{"error":"invalid or expired token"}`},
		{"garbage token", "Bearer garbage", http.StatusUnauthorized, `{"error":"invalid or expired token"}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/protected", http.NoBody)
			if tt.header != "" {
				req.Header.Set("Authorization", tt.header)
			}
			w := serve(t, engine, req)
			if w.Code != tt.wantCode || w.Body.String() != tt.wantBody {
				t.Errorf("got %d %s, want %d %s", w.Code, w.Body.String(), tt.wantCode, tt.wantBody)
			}
		})
	}
}

func TestGetUserID(t *testing.T) {
	tests := []struct {
		name   string
		value  any
		set    bool
		wantID string
		wantOK bool
	}{
		{"not set", nil, false, "", false},
		{"string", "user_1", true, "user_1", true},
		{"empty string", "", true, "", false},
		{"wrong type", 42, true, "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			if tt.set {
				c.Set(ContextKeyUserID, tt.value)
			}
			id, ok := GetUserID(c)
			if id != tt.wantID || ok != tt.wantOK {
				t.Errorf("GetUserID() = %q, %v; want %q, %v", id, ok, tt.wantID, tt.wantOK)
			}
		})
	}
}

func TestErrorHandlerMiddleware_RecoversPanics(t *testing.T) {
	engine := gin.New()
	engine.Use(ErrorHandlerMiddleware())
	engine.GET("/panic", func(*gin.Context) { panic("boom") })
	engine.GET("/ok", func(c *gin.Context) { c.Status(http.StatusNoContent) })

	w := serve(t, engine, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/panic", http.NoBody))
	if w.Code != http.StatusInternalServerError || w.Body.String() != `{"error":"internal server error"}` {
		t.Errorf("panic: got %d %s", w.Code, w.Body.String())
	}

	w = serve(t, engine, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/ok", http.NoBody))
	if w.Code != http.StatusNoContent {
		t.Errorf("ok: status = %d, want 204", w.Code)
	}
}

func TestLoggerMiddleware_PassesThrough(t *testing.T) {
	engine := gin.New()
	engine.Use(LoggerMiddleware())
	engine.GET("/teapot", func(c *gin.Context) { c.Status(http.StatusTeapot) })

	w := serve(t, engine, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/teapot", http.NoBody))
	if w.Code != http.StatusTeapot {
		t.Errorf("status = %d, want 418", w.Code)
	}
}

func TestCORSMiddleware(t *testing.T) {
	cfg := &config.Config{CORSAllowedOrigins: []string{"http://localhost:3000"}}
	engine := gin.New()
	engine.Use(CORSMiddleware(cfg))
	engine.GET("/x", func(c *gin.Context) { c.Status(http.StatusOK) })

	preflight := func(origin string) *httptest.ResponseRecorder {
		req := httptest.NewRequestWithContext(t.Context(), http.MethodOptions, "/x", http.NoBody)
		req.Header.Set("Origin", origin)
		req.Header.Set("Access-Control-Request-Method", http.MethodPut)
		req.Header.Set("Access-Control-Request-Headers", "Authorization")
		return serve(t, engine, req)
	}

	w := preflight("http://localhost:3000")
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:3000" {
		t.Errorf("allowed origin: Access-Control-Allow-Origin = %q", got)
	}
	if got := w.Header().Get("Access-Control-Allow-Credentials"); got != "true" {
		t.Errorf("Access-Control-Allow-Credentials = %q, want true", got)
	}

	w = preflight("https://evil.example.com")
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("disallowed origin: Access-Control-Allow-Origin = %q, want empty", got)
	}
}
