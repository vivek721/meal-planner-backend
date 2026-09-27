// Package handlers implements the Gin HTTP handlers for the API.
package handlers

import (
	"errors"

	"github.com/meal-planner/backend/internal/utils"
)

const (
	msgInvalidRequestBody = "invalid request body"
	msgUnauthorized       = "unauthorized"
)

// validationErrors are input-validation errors whose messages are safe to
// return to the client with a 400 status.
var validationErrors = []error{
	utils.ErrEmailRequired,
	utils.ErrInvalidEmail,
	utils.ErrPasswordRequired,
	utils.ErrPasswordTooShort,
	utils.ErrPasswordTooWeak,
}

func isValidationError(err error) bool {
	for _, target := range validationErrors {
		if errors.Is(err, target) {
			return true
		}
	}
	return false
}
