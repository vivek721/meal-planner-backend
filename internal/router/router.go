// Package router wires middleware, handlers and routes into a Gin engine.
package router

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/meal-planner/backend/internal/config"
	"github.com/meal-planner/backend/internal/handlers"
	"github.com/meal-planner/backend/internal/mealdb"
	"github.com/meal-planner/backend/internal/middleware"
	"github.com/meal-planner/backend/internal/nutrition/overrides"
	"github.com/meal-planner/backend/internal/repository"
	"github.com/meal-planner/backend/internal/services"
	"github.com/meal-planner/backend/internal/usda"
)

// Setup initializes and configures the router backed by the given database.
func Setup(db *gorm.DB, cfg *config.Config, ov *overrides.Set) *gin.Engine {
	cache := repository.NewCacheRepository(db)
	recipes := services.NewRecipeService(
		mealdb.NewHTTPClient(cfg.MealDBBaseURL, cfg.MealDBTimeout),
		cache,
		services.RecipeCacheTTL{Detail: cfg.MealDBDetailTTL, Search: cfg.MealDBSearchTTL},
		time.Now,
	)
	nutrition := services.NewNutritionService(
		recipes,
		usda.NewHTTPClient(cfg.USDABaseURL, cfg.USDAAPIKey, cfg.USDATimeout),
		cache, ov,
		services.NutritionTTL{Match: cfg.USDAMatchTTL, Food: cfg.USDAFoodTTL, Result: cfg.NutritionTTL},
		time.Now,
	)
	return New(repository.NewUserRepository(db), recipes, nutrition, cfg)
}

// New builds the router on top of the given repositories and services.
func New(userRepo repository.UserRepository, recipes services.RecipeService, nutrition services.NutritionService, cfg *config.Config) *gin.Engine {
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
			"status":  "healthy",
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
					"register":    "POST /api/auth/register",
					"login":       "POST /api/auth/login",
					"refresh":     "POST /api/auth/refresh",
					"me":          "GET /api/auth/me (protected)",
					"logout":      "POST /api/auth/logout (protected)",
					"profile":     "PUT /api/auth/profile (protected)",
					"password":    "PUT /api/auth/password (protected)",
					"onboarding":  "POST /api/auth/onboarding/complete (protected)",
					"preferences": "PUT /api/auth/preferences (protected)",
				},
				"recipes": gin.H{
					"search":     "GET /api/recipes?q=&category=&cuisine=&ingredient=&page=&limit= (protected)",
					"detail":     "GET /api/recipes/:id (protected)",
					"categories": "GET /api/recipes/categories (protected)",
					"cuisines":   "GET /api/recipes/cuisines (protected)",
					"nutrition":  "GET /api/recipes/:id/nutrition (protected)",
				},
			},
		})
	})

	// Initialize services
	authService := services.NewAuthService(userRepo, cfg)
	userService := services.NewUserService(userRepo, cfg)

	// Initialize handlers
	authHandler := handlers.NewAuthHandler(authService)
	userHandler := handlers.NewUserHandler(userService)
	recipeHandler := handlers.NewRecipeHandler(recipes)
	nutritionHandler := handlers.NewNutritionHandler(nutrition)

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
			protected.GET("/me", userHandler.GetMe)
			protected.POST("/logout", authHandler.Logout)
			protected.PUT("/profile", userHandler.UpdateProfile)
			protected.PUT("/password", userHandler.ChangePassword)
			protected.PUT("/preferences", userHandler.UpdatePreferences)

			// Onboarding
			protected.POST("/onboarding/complete", userHandler.CompleteOnboarding)
		}

		// Recipe routes (protected)
		recipeRoutes := api.Group("/recipes")
		recipeRoutes.Use(middleware.AuthMiddleware(cfg))
		recipeRoutes.GET("", recipeHandler.Search)
		recipeRoutes.GET("/categories", recipeHandler.Categories)
		recipeRoutes.GET("/cuisines", recipeHandler.Cuisines)
		recipeRoutes.GET("/:id", recipeHandler.Get)
		recipeRoutes.GET("/:id/nutrition", nutritionHandler.Get)
	}

	return router
}
