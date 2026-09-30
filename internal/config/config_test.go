package config

import (
	"reflect"
	"testing"
	"time"
)

func TestLoad_Defaults(t *testing.T) {
	for _, key := range []string{
		"PORT", "ENVIRONMENT", "DATABASE_URL", "DB_HOST", "JWT_SECRET",
		"JWT_EXPIRATION_HOURS", "BCRYPT_COST", "FRONTEND_URL",
		"RATE_LIMIT_ENABLED", "RATE_LIMIT_PER_MIN",
	} {
		t.Setenv(key, "")
	}

	cfg := Load()

	checks := []struct {
		name      string
		got, want any
	}{
		{"Port", cfg.Port, "3001"},
		{"Environment", cfg.Environment, "development"},
		{"DatabaseURL", cfg.DatabaseURL, ""},
		{"DatabaseHost", cfg.DatabaseHost, "localhost"},
		{"JWTExpirationHours", cfg.JWTExpirationHours, 24},
		{"BcryptCost", cfg.BcryptCost, 12},
		{"CORSAllowedOrigins", cfg.CORSAllowedOrigins, []string{"http://localhost:3000"}},
		{"RateLimitEnabled", cfg.RateLimitEnabled, true},
		{"RateLimitPerMin", cfg.RateLimitPerMin, 100},
	}
	for _, c := range checks {
		if !reflect.DeepEqual(c.got, c.want) {
			t.Errorf("%s = %v, want %v", c.name, c.got, c.want)
		}
	}
}

func TestLoad_FromEnvironment(t *testing.T) {
	t.Setenv("PORT", "8080")
	t.Setenv("ENVIRONMENT", "production")
	t.Setenv("DATABASE_URL", "postgres://u:p@db:5432/x")
	t.Setenv("JWT_SECRET", "from-env")
	t.Setenv("JWT_EXPIRATION_HOURS", "2")
	t.Setenv("BCRYPT_COST", "4")
	t.Setenv("FRONTEND_URL", "https://app.example.com")
	t.Setenv("RATE_LIMIT_ENABLED", "false")

	cfg := Load()

	if cfg.Port != "8080" || cfg.Environment != "production" || cfg.DatabaseURL != "postgres://u:p@db:5432/x" {
		t.Errorf("string settings not read from env: %+v", cfg)
	}
	if cfg.JWTSecret != "from-env" || cfg.JWTExpirationHours != 2 || cfg.BcryptCost != 4 {
		t.Errorf("auth settings not read from env: %+v", cfg)
	}
	if !reflect.DeepEqual(cfg.CORSAllowedOrigins, []string{"https://app.example.com"}) {
		t.Errorf("CORSAllowedOrigins = %v", cfg.CORSAllowedOrigins)
	}
	if cfg.RateLimitEnabled {
		t.Error("RateLimitEnabled = true, want false")
	}
}

func TestLoad_InvalidNumbersFallBackToDefaults(t *testing.T) {
	t.Setenv("JWT_EXPIRATION_HOURS", "a day")
	t.Setenv("RATE_LIMIT_ENABLED", "maybe")

	cfg := Load()

	if cfg.JWTExpirationHours != 24 {
		t.Errorf("JWTExpirationHours = %d, want default 24", cfg.JWTExpirationHours)
	}
	if !cfg.RateLimitEnabled {
		t.Error("RateLimitEnabled = false, want default true")
	}
}

func TestGetJWTExpiration(t *testing.T) {
	cfg := &Config{JWTExpirationHours: 3}
	if got := cfg.GetJWTExpiration(); got != 3*time.Hour {
		t.Errorf("GetJWTExpiration() = %v, want 3h", got)
	}
}

func TestEnvironmentHelpers(t *testing.T) {
	tests := []struct {
		env       string
		dev, prod bool
	}{
		{"development", true, false},
		{"production", false, true},
		{"test", false, false},
	}
	for _, tt := range tests {
		cfg := &Config{Environment: tt.env}
		if cfg.IsDevelopment() != tt.dev || cfg.IsProduction() != tt.prod {
			t.Errorf("%s: IsDevelopment=%v IsProduction=%v, want %v %v",
				tt.env, cfg.IsDevelopment(), cfg.IsProduction(), tt.dev, tt.prod)
		}
	}
}

func TestLoadMealDBDefaults(t *testing.T) {
	for _, k := range []string{"MEALDB_BASE_URL", "MEALDB_TIMEOUT_SECONDS", "MEALDB_DETAIL_TTL_HOURS", "MEALDB_SEARCH_TTL_HOURS"} {
		t.Setenv(k, "")
	}
	cfg := Load()
	if cfg.MealDBBaseURL != "https://www.themealdb.com/api/json/v1/1" {
		t.Errorf("MealDBBaseURL = %q", cfg.MealDBBaseURL)
	}
	if cfg.MealDBTimeout != 5*time.Second {
		t.Errorf("MealDBTimeout = %v", cfg.MealDBTimeout)
	}
	if cfg.MealDBDetailTTL != 168*time.Hour {
		t.Errorf("MealDBDetailTTL = %v", cfg.MealDBDetailTTL)
	}
	if cfg.MealDBSearchTTL != 24*time.Hour {
		t.Errorf("MealDBSearchTTL = %v", cfg.MealDBSearchTTL)
	}
}

func TestLoadMealDBNonPositiveFallBackToDefaults(t *testing.T) {
	for _, bad := range []string{"0", "-3"} {
		t.Setenv("MEALDB_TIMEOUT_SECONDS", bad)
		t.Setenv("MEALDB_DETAIL_TTL_HOURS", bad)
		t.Setenv("MEALDB_SEARCH_TTL_HOURS", bad)

		cfg := Load()

		if cfg.MealDBTimeout != 5*time.Second {
			t.Errorf("MEALDB_TIMEOUT_SECONDS=%s: MealDBTimeout = %v, want default 5s", bad, cfg.MealDBTimeout)
		}
		if cfg.MealDBDetailTTL != 168*time.Hour {
			t.Errorf("MEALDB_DETAIL_TTL_HOURS=%s: MealDBDetailTTL = %v, want default 168h", bad, cfg.MealDBDetailTTL)
		}
		if cfg.MealDBSearchTTL != 24*time.Hour {
			t.Errorf("MEALDB_SEARCH_TTL_HOURS=%s: MealDBSearchTTL = %v, want default 24h", bad, cfg.MealDBSearchTTL)
		}
	}
}

func TestLoadCacheRetentionDefault(t *testing.T) {
	t.Setenv("MEALDB_CACHE_RETENTION_DAYS", "")
	cfg := Load()
	if cfg.MealDBCacheRetention != 30*24*time.Hour {
		t.Errorf("MealDBCacheRetention = %v, want default 30 days", cfg.MealDBCacheRetention)
	}
}

func TestLoadCacheRetentionOverride(t *testing.T) {
	t.Setenv("MEALDB_CACHE_RETENTION_DAYS", "7")
	cfg := Load()
	if cfg.MealDBCacheRetention != 7*24*time.Hour {
		t.Errorf("MealDBCacheRetention = %v, want 7 days", cfg.MealDBCacheRetention)
	}
}

func TestLoadCacheRetentionNonPositiveFallsBackToDefault(t *testing.T) {
	for _, bad := range []string{"0", "-3"} {
		t.Setenv("MEALDB_CACHE_RETENTION_DAYS", bad)
		cfg := Load()
		if cfg.MealDBCacheRetention != 30*24*time.Hour {
			t.Errorf("MEALDB_CACHE_RETENTION_DAYS=%s: MealDBCacheRetention = %v, want default 30 days", bad, cfg.MealDBCacheRetention)
		}
	}
}

func TestLoadMealDBOverrides(t *testing.T) {
	t.Setenv("MEALDB_BASE_URL", "http://mealdb.test/api")
	t.Setenv("MEALDB_TIMEOUT_SECONDS", "2")
	t.Setenv("MEALDB_DETAIL_TTL_HOURS", "1")
	t.Setenv("MEALDB_SEARCH_TTL_HOURS", "3")
	cfg := Load()
	if cfg.MealDBBaseURL != "http://mealdb.test/api" || cfg.MealDBTimeout != 2*time.Second ||
		cfg.MealDBDetailTTL != time.Hour || cfg.MealDBSearchTTL != 3*time.Hour {

		t.Errorf("overrides not applied: %+v", cfg)
	}
}

func TestUSDADefaults(t *testing.T) {
	for _, key := range []string{
		"USDA_API_KEY", "USDA_BASE_URL", "USDA_TIMEOUT_SECONDS",
		"USDA_MATCH_TTL_HOURS", "USDA_FOOD_TTL_HOURS", "NUTRITION_TTL_HOURS",
	} {
		t.Setenv(key, "")
	}
	cfg := Load()
	if cfg.USDAAPIKey != "DEMO_KEY" {
		t.Errorf("USDAAPIKey = %q", cfg.USDAAPIKey)
	}
	if cfg.USDABaseURL != "https://api.nal.usda.gov/fdc/v1" {
		t.Errorf("USDABaseURL = %q", cfg.USDABaseURL)
	}
	if cfg.USDATimeout != 5*time.Second {
		t.Errorf("USDATimeout = %v", cfg.USDATimeout)
	}
	if cfg.USDAMatchTTL != 720*time.Hour || cfg.USDAFoodTTL != 2160*time.Hour || cfg.NutritionTTL != 168*time.Hour {
		t.Errorf("TTLs = %v %v %v", cfg.USDAMatchTTL, cfg.USDAFoodTTL, cfg.NutritionTTL)
	}
}

func TestUSDAOverrides(t *testing.T) {
	t.Setenv("USDA_API_KEY", "real-key")
	t.Setenv("USDA_TIMEOUT_SECONDS", "9")
	t.Setenv("NUTRITION_TTL_HOURS", "1")
	cfg := Load()
	if cfg.USDAAPIKey != "real-key" || cfg.USDATimeout != 9*time.Second || cfg.NutritionTTL != time.Hour {
		t.Errorf("USDA overrides not applied: key=%q timeout=%v ttl=%v", cfg.USDAAPIKey, cfg.USDATimeout, cfg.NutritionTTL)
	}
}
