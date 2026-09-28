package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/meal-planner/backend/internal/services"
)

const (
	msgInvalidRecipeID   = "invalid recipe id"
	msgInvalidPaging     = "page and limit must be positive integers"
	msgRecipesDown       = "recipes are temporarily unavailable, please try again shortly"
	msgRecipeNotFound    = "recipe not found"
	msgInternalServerErr = "internal server error"
)

// RecipeHandler serves TheMealDB recipes.
type RecipeHandler struct {
	recipes services.RecipeService
}

// NewRecipeHandler creates a RecipeHandler.
func NewRecipeHandler(recipes services.RecipeService) *RecipeHandler {
	return &RecipeHandler{recipes: recipes}
}

// Categories returns all recipe categories.
func (h *RecipeHandler) Categories(c *gin.Context) {
	cats, err := h.recipes.Categories(c.Request.Context())
	if err != nil {
		recipeError(c, err)
		return
	}
	c.JSON(http.StatusOK, cats)
}

// Cuisines returns all cuisine names, sorted.
func (h *RecipeHandler) Cuisines(c *gin.Context) {
	cuisines, err := h.recipes.Cuisines(c.Request.Context())
	if err != nil {
		recipeError(c, err)
		return
	}
	c.JSON(http.StatusOK, cuisines)
}

// Get returns one recipe by its numeric TheMealDB id.
func (h *RecipeHandler) Get(c *gin.Context) {
	id := c.Param("id")
	if !isPositiveInt(id) {
		c.JSON(http.StatusBadRequest, gin.H{"error": msgInvalidRecipeID})
		return
	}
	meal, err := h.recipes.Get(c.Request.Context(), id)
	if err != nil {
		recipeError(c, err)
		return
	}
	c.JSON(http.StatusOK, meal)
}

// Search returns a page of recipes matching q/category/cuisine/ingredient.
func (h *RecipeHandler) Search(c *gin.Context) {
	page, okPage := optionalPositiveInt(c.Query("page"))
	limit, okLimit := optionalPositiveInt(c.Query("limit"))
	if !okPage || !okLimit {
		c.JSON(http.StatusBadRequest, gin.H{"error": msgInvalidPaging})
		return
	}
	result, err := h.recipes.Search(c.Request.Context(), services.RecipeQuery{
		Q: c.Query("q"), Category: c.Query("category"), Cuisine: c.Query("cuisine"), Ingredient: c.Query("ingredient"),
		Page: page, Limit: limit,
	})
	if err != nil {
		recipeError(c, err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func recipeError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, services.ErrInvalidSearch):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	case errors.Is(err, services.ErrRecipeNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": msgRecipeNotFound})
	case errors.Is(err, services.ErrUpstreamUnavailable):
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": msgRecipesDown})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": msgInternalServerErr})
	}
}

func isPositiveInt(s string) bool {
	n, err := strconv.Atoi(s)
	return err == nil && n > 0 && strconv.Itoa(n) == s
}

// optionalPositiveInt parses an optional query value: "" gives (0, true).
func optionalPositiveInt(s string) (int, bool) {
	if s == "" {
		return 0, true
	}
	if !isPositiveInt(s) {
		return 0, false
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0, false
	}
	return n, true
}
