package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/meal-planner/backend/internal/services"
)

// AuthHandler serves the public authentication endpoints.
type AuthHandler struct {
	authService services.AuthService
}

// NewAuthHandler creates an AuthHandler.
func NewAuthHandler(authService services.AuthService) *AuthHandler {
	return &AuthHandler{
		authService: authService,
	}
}

// RegisterRequest represents the registration request body
type RegisterRequest struct {
	Email           string `json:"email" binding:"required"`
	Password        string `json:"password" binding:"required"`
	ConfirmPassword string `json:"confirmPassword"`
	Name            string `json:"name"`
}

// LoginRequest represents the login request body
type LoginRequest struct {
	Email      string `json:"email" binding:"required"`
	Password   string `json:"password" binding:"required"`
	RememberMe bool   `json:"rememberMe"`
}

// RefreshTokenRequest represents the refresh token request body
type RefreshTokenRequest struct {
	Token string `json:"token" binding:"required"`
}

// AuthResponse represents the authentication response
type AuthResponse struct {
	User  interface{} `json:"user"`
	Token string      `json:"token"`
}

// Register handles user registration
// POST /api/auth/register
func (h *AuthHandler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": msgInvalidRequestBody,
		})
		return
	}

	// Register user
	user, token, err := h.authService.Register(req.Email, req.Password, req.Name)
	if err != nil {
		statusCode := http.StatusInternalServerError
		errorMsg := "failed to register user"

		switch {
		case errors.Is(err, services.ErrUserAlreadyExists):
			statusCode = http.StatusConflict
			errorMsg = "Email already exists"
		case isValidationError(err):
			statusCode = http.StatusBadRequest
			errorMsg = err.Error()
		}

		c.JSON(statusCode, gin.H{
			"error": errorMsg,
		})
		return
	}

	c.JSON(http.StatusCreated, AuthResponse{
		User:  user.ToPublicUser(),
		Token: token,
	})
}

// Login handles user login
// POST /api/auth/login
func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": msgInvalidRequestBody,
		})
		return
	}

	// Login user
	user, token, err := h.authService.Login(req.Email, req.Password)
	if err != nil {
		statusCode := http.StatusUnauthorized
		errorMsg := "Invalid email or password"

		// Matches both ErrAccountLocked and *services.AccountLockedError
		// (whose message includes the remaining lock time).
		if errors.Is(err, services.ErrAccountLocked) {
			statusCode = http.StatusForbidden
			errorMsg = err.Error()
		}

		c.JSON(statusCode, gin.H{
			"error": errorMsg,
		})
		return
	}

	c.JSON(http.StatusOK, AuthResponse{
		User:  user.ToPublicUser(),
		Token: token,
	})
}

// RefreshToken handles token refresh
// POST /api/auth/refresh
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	var req RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": msgInvalidRequestBody,
		})
		return
	}

	newToken, err := h.authService.RefreshToken(req.Token)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "invalid or expired token",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token": newToken,
	})
}

// Logout handles user logout
// POST /api/auth/logout
func (h *AuthHandler) Logout(c *gin.Context) {
	// In a stateless JWT implementation, logout is handled client-side
	// However, we can add token blacklisting here if needed in the future
	c.JSON(http.StatusOK, gin.H{
		"message": "logged out successfully",
	})
}
