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
	r := router.Setup(db, cfg)

	// Purge long-expired mealdb_cache rows once at startup and then every
	// CachePurgeInterval, stopping when the process shuts down.
	purgeCtx, stopPurge := context.WithCancel(context.Background())
	defer stopPurge()
	go services.PurgeExpiredCacheLoop(
		purgeCtx, repository.NewCacheRepository(db), cfg.MealDBCacheRetention, time.Now, services.CachePurgeInterval,
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
