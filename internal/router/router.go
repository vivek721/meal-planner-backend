package router

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/meal-planner/backend/internal/config"
	"github.com/meal-planner/backend/internal/handlers"
	"github.com/meal-planner/backend/internal/middleware"
	"github.com/meal-planner/backend/internal/repository"
	"github.com/meal-planner/backend/internal/services"
	"gorm.io/gorm"
)

// Setup initializes and configures the router
func Setup(db *gorm.DB, cfg *config.Config) *gin.Engine {
	// Set Gin mode based on environment
	if cfg.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()

	// Apply global middleware
	router.Use(middleware.ErrorHandlerMiddleware())
	router.Use(middleware.LoggerMiddleware())
	router.Use(middleware.CORSMiddleware(cfg))

	// Health check endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "healthy",
			"service": "meal-planner-api",
		})
	})

	// API info endpoint
	router.GET("/api", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"service": "Meal Planner API",
			"version": "1.0.0",
			"endpoints": gin.H{
				"health": "/health",
				"auth": gin.H{
					"register": "POST /api/auth/register",
					"login": "POST /api/auth/login",
					"refresh": "POST /api/auth/refresh",
					"me": "GET /api/auth/me (protected)",
					"logout": "POST /api/auth/logout (protected)",
					"profile": "PUT /api/auth/profile (protected)",
					"password": "PUT /api/auth/password (protected)",
					"onboarding": "POST /api/auth/onboarding/complete (protected)",
					"preferences": "PUT /api/auth/preferences (protected)",
				},
			},
		})
	})

	// Initialize repositories
	userRepo := repository.NewUserRepository(db)

	// Initialize services
	authService := services.NewAuthService(userRepo, cfg)
	userService := services.NewUserService(userRepo, cfg)

	// Initialize handlers
	authHandler := handlers.NewAuthHandler(authService)
	userHandler := handlers.NewUserHandler(userService)

	// API routes
	api := router.Group("/api")
	{
		// Auth routes (public)
		auth := api.Group("/auth")
		{
			auth.POST("/register", authHandler.Register)
			auth.POST("/login", authHandler.Login)
			auth.POST("/refresh", authHandler.RefreshToken)

			// Protected auth routes
			protected := auth.Group("")
			protected.Use(middleware.AuthMiddleware(cfg))
			{
				protected.GET("/me", authHandler.GetMe)
				protected.POST("/logout", authHandler.Logout)
				protected.PUT("/profile", userHandler.UpdateProfile)
				protected.PUT("/password", userHandler.ChangePassword)
				protected.PUT("/preferences", userHandler.UpdatePreferences)

				// Onboarding
				protected.POST("/onboarding/complete", userHandler.CompleteOnboarding)
			}
		}
	}

	return router
}
