package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/meal-planner/backend/internal/middleware"
	"github.com/meal-planner/backend/internal/models"
	"github.com/meal-planner/backend/internal/services"
)

type UserHandler struct {
	userService services.UserService
}

func NewUserHandler(userService services.UserService) *UserHandler {
	return &UserHandler{
		userService: userService,
	}
}

// UpdateProfileRequest represents the update profile request body
type UpdateProfileRequest struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

// ChangePasswordRequest represents the change password request body
type ChangePasswordRequest struct {
	CurrentPassword string `json:"currentPassword" binding:"required"`
	NewPassword     string `json:"newPassword" binding:"required"`
}

// UpdatePreferencesRequest represents the update preferences request body
type UpdatePreferencesRequest struct {
	Theme         string `json:"theme"`
	Notifications *bool  `json:"notifications"`
}

// UpdateProfile updates the user profile
// PUT /api/auth/profile
func (h *UserHandler) UpdateProfile(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "unauthorized",
		})
		return
	}

	var req UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request body",
		})
		return
	}

	user, err := h.userService.UpdateProfile(userID, req.Name, req.Email)
	if err != nil {
		statusCode := http.StatusInternalServerError
		errorMsg := "failed to update profile"

		switch err {
		case services.ErrUserNotFound:
			statusCode = http.StatusNotFound
			errorMsg = "user not found"
		case services.ErrUserAlreadyExists:
			statusCode = http.StatusConflict
			errorMsg = "email already in use"
		default:
			if err.Error() == "invalid email format" {
				statusCode = http.StatusBadRequest
				errorMsg = err.Error()
			}
		}

		c.JSON(statusCode, gin.H{
			"error": errorMsg,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user": user.ToPublicUser(),
	})
}

// ChangePassword changes the user password
// PUT /api/auth/password
func (h *UserHandler) ChangePassword(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "unauthorized",
		})
		return
	}

	var req ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request body",
		})
		return
	}

	err := h.userService.ChangePassword(userID, req.CurrentPassword, req.NewPassword)
	if err != nil {
		statusCode := http.StatusInternalServerError
		errorMsg := "failed to change password"

		switch err {
		case services.ErrUserNotFound:
			statusCode = http.StatusNotFound
			errorMsg = "user not found"
		case services.ErrCurrentPasswordIncorrect:
			statusCode = http.StatusBadRequest
			errorMsg = "current password is incorrect"
		default:
			if err.Error() == "password must be at least 8 characters" ||
				err.Error() == "password must contain uppercase, lowercase, number, and special character" {
				statusCode = http.StatusBadRequest
				errorMsg = err.Error()
			}
		}

		c.JSON(statusCode, gin.H{
			"error": errorMsg,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "password changed successfully",
	})
}

// CompleteOnboarding marks onboarding as complete
// POST /api/auth/onboarding/complete
func (h *UserHandler) CompleteOnboarding(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "unauthorized",
		})
		return
	}

	user, err := h.userService.CompleteOnboarding(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to complete onboarding",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user": user.ToPublicUser(),
	})
}

// UpdatePreferences updates user preferences
// PUT /api/auth/preferences
func (h *UserHandler) UpdatePreferences(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "unauthorized",
		})
		return
	}

	var req UpdatePreferencesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request body",
		})
		return
	}

	preferences := &models.UserPreferences{}
	if req.Theme != "" {
		preferences.Theme = req.Theme
	}
	if req.Notifications != nil {
		preferences.Notifications = *req.Notifications
	}

	user, err := h.userService.UpdatePreferences(userID, preferences)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to update preferences",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user": user.ToPublicUser(),
	})
}
