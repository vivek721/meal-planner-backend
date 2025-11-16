package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	// Server configuration
	Port        string
	Environment string

	// Database configuration
	DatabaseURL      string
	DatabaseHost     string
	DatabasePort     string
	DatabaseUser     string
	DatabasePassword string
	DatabaseName     string
	DatabaseSSLMode  string

	// JWT configuration
	JWTSecret           string
	JWTExpirationHours  int
	JWTRefreshDays      int

	// Security configuration
	BcryptCost int

	// CORS configuration
	CORSAllowedOrigins []string

	// Rate limiting
	RateLimitEnabled bool
	RateLimitPerMin  int
}

// Load loads configuration from environment variables
func Load() *Config {
	return &Config{
		// Server
		Port:        getEnv("PORT", "3001"),
		Environment: getEnv("ENVIRONMENT", "development"),

		// Database - support both DATABASE_URL and individual params
		DatabaseURL:      getEnv("DATABASE_URL", ""),
		DatabaseHost:     getEnv("DB_HOST", "localhost"),
		DatabasePort:     getEnv("DB_PORT", "5432"),
		DatabaseUser:     getEnv("DB_USER", "postgres"),
		DatabasePassword: getEnv("DB_PASSWORD", "postgres"),
		DatabaseName:     getEnv("DB_NAME", "meal_planner"),
		DatabaseSSLMode:  getEnv("DB_SSLMODE", "disable"),

		// JWT
		JWTSecret:          getEnv("JWT_SECRET", "your-secret-key-change-this-in-production"),
		JWTExpirationHours: getEnvAsInt("JWT_EXPIRATION_HOURS", 24),
		JWTRefreshDays:     getEnvAsInt("JWT_REFRESH_DAYS", 30),

		// Security
		BcryptCost: getEnvAsInt("BCRYPT_COST", 12),

		// CORS
		CORSAllowedOrigins: []string{
			getEnv("FRONTEND_URL", "http://localhost:3000"),
		},

		// Rate limiting
		RateLimitEnabled: getEnvAsBool("RATE_LIMIT_ENABLED", true),
		RateLimitPerMin:  getEnvAsInt("RATE_LIMIT_PER_MIN", 100),
	}
}

// GetJWTExpiration returns the JWT token expiration duration
func (c *Config) GetJWTExpiration() time.Duration {
	return time.Hour * time.Duration(c.JWTExpirationHours)
}

// GetJWTRefreshExpiration returns the refresh token expiration duration
func (c *Config) GetJWTRefreshExpiration() time.Duration {
	return time.Hour * 24 * time.Duration(c.JWTRefreshDays)
}

// IsDevelopment checks if the environment is development
func (c *Config) IsDevelopment() bool {
	return c.Environment == "development"
}

// IsProduction checks if the environment is production
func (c *Config) IsProduction() bool {
	return c.Environment == "production"
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	valueStr := os.Getenv(key)
	if value, err := strconv.Atoi(valueStr); err == nil {
		return value
	}
	return defaultValue
}

func getEnvAsBool(key string, defaultValue bool) bool {
	valueStr := os.Getenv(key)
	if value, err := strconv.ParseBool(valueStr); err == nil {
		return value
	}
	return defaultValue
}
