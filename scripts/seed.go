package main

import (
	"log"
	"time"

	"github.com/joho/godotenv"
	"github.com/meal-planner/backend/internal/config"
	"github.com/meal-planner/backend/internal/database"
	"github.com/meal-planner/backend/internal/models"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	// Load environment variables
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

	log.Println("Starting database seeding...")

	// Check if users already exist
	var count int64
	db.Model(&models.User{}).Count(&count)
	if count > 0 {
		log.Printf("Database already has %d users. Skipping seed.", count)
		return
	}

	// Create test users
	testUsers := []struct {
		email    string
		name     string
		password string
		onboarded bool
	}{
		{
			email:    "test@example.com",
			name:     "Test User",
			password: "password123",
			onboarded: true,
		},
		{
			email:    "demo@example.com",
			name:     "Demo User",
			password: "demo123",
			onboarded: true,
		},
		{
			email:    "newuser@example.com",
			name:     "New User",
			password: "newpass123",
			onboarded: false,
		},
	}

	for _, testUser := range testUsers {
		// Hash password
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(testUser.password), bcrypt.DefaultCost)
		if err != nil {
			log.Fatalf("Failed to hash password: %v", err)
		}

		user := models.User{
			Email:                  testUser.email,
			Name:                   testUser.name,
			PasswordHash:           string(hashedPassword),
			HasCompletedOnboarding: testUser.onboarded,
			CreatedAt:              time.Now(),
			UpdatedAt:              time.Now(),
			Preferences: &models.UserPreferences{
				Theme:         "light",
				Notifications: true,
			},
		}

		if err := db.Create(&user).Error; err != nil {
			log.Fatalf("Failed to create user %s: %v", testUser.email, err)
		}

		log.Printf("Created user: %s (ID: %s)", user.Email, user.ID)
	}

	log.Printf("Successfully seeded %d test users", len(testUsers))
	log.Println("\nTest Credentials:")
	log.Println("  Email: test@example.com | Password: password123")
	log.Println("  Email: demo@example.com | Password: demo123")
	log.Println("  Email: newuser@example.com | Password: newpass123")
}
