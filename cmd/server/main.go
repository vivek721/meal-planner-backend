// Command server runs the Meal Planner HTTP API.
package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"

	"github.com/meal-planner/backend/internal/config"
	"github.com/meal-planner/backend/internal/database"
	"github.com/meal-planner/backend/internal/nutrition/overrides"
	"github.com/meal-planner/backend/internal/repository"
	"github.com/meal-planner/backend/internal/router"
	"github.com/meal-planner/backend/internal/services"
)

func main() {
	// Load environment variables from .env file
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	// Load configuration
	cfg := config.Load()

	// Validate the committed nutrition overrides before serving anything:
	// running with silently-broken overrides would mis-estimate every recipe.
	ov, err := overrides.Load()
	if err != nil {
		log.Fatalf("Invalid nutrition overrides: %v", err)
	}
	if cfg.USDAAPIKey == "DEMO_KEY" {
		log.Println("USDA_API_KEY not set: using DEMO_KEY (10 requests/hour)")
	} else {
		log.Println("USDA_API_KEY is set")
	}

	// Initialize database connection
	db, err := database.NewConnection(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Run database migrations
	if err := database.Migrate(db); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	// Initialize router with dependencies
	r := router.Setup(db, cfg, ov)

	// Purge long-expired mealdb_cache rows once at startup and then every
	// CachePurgeInterval. The server has no graceful-shutdown path today, so
	// the loop simply runs for the lifetime of the process; PurgeExpiredCacheLoop
	// itself supports stopping via context cancellation (see its tests).
	go services.PurgeExpiredCacheLoop(
		context.Background(), repository.NewCacheRepository(db), cfg.MealDBCacheRetention, time.Now, services.CachePurgeInterval,
	)

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "3001"
	}

	log.Printf("Server starting on port %s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
