package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/meal-planner/backend/internal/services"
)

const msgNutritionDown = "nutrition is temporarily unavailable, please try again shortly"

// NutritionHandler serves USDA-based nutrition estimates.
type NutritionHandler struct {
	nutrition services.NutritionService
}

// NewNutritionHandler creates a NutritionHandler.
func NewNutritionHandler(nutrition services.NutritionService) *NutritionHandler {
	return &NutritionHandler{nutrition: nutrition}
}

// Get returns the whole-recipe nutrition estimate for one recipe id.
func (h *NutritionHandler) Get(c *gin.Context) {
	id := c.Param("id")
	if !isPositiveInt(id) {
		c.JSON(http.StatusBadRequest, gin.H{"error": msgInvalidRecipeID})
		return
	}
	est, err := h.nutrition.Estimate(c.Request.Context(), id)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrRecipeNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": msgRecipeNotFound})
		case errors.Is(err, services.ErrUpstreamUnavailable):
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": msgNutritionDown})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": msgInternalServerErr})
		}
		return
	}
	c.JSON(http.StatusOK, est)
}
