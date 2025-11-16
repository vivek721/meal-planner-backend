package services

import (
	"errors"

	"github.com/meal-planner/backend/internal/config"
	"github.com/meal-planner/backend/internal/models"
	"github.com/meal-planner/backend/internal/repository"
	"github.com/meal-planner/backend/internal/utils"
)

var (
	ErrCurrentPasswordIncorrect = errors.New("current password is incorrect")
)

type UserService interface {
	GetUserByID(userID string) (*models.User, error)
	UpdateProfile(userID, name, email string) (*models.User, error)
	ChangePassword(userID, currentPassword, newPassword string) error
	CompleteOnboarding(userID string) (*models.User, error)
	UpdatePreferences(userID string, preferences *models.UserPreferences) (*models.User, error)
}

type userService struct {
	userRepo repository.UserRepository
	config   *config.Config
}

func NewUserService(userRepo repository.UserRepository, cfg *config.Config) UserService {
	return &userService{
		userRepo: userRepo,
		config:   cfg,
	}
}

func (s *userService) GetUserByID(userID string) (*models.User, error) {
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrUserNotFound
	}
	return user, nil
}

func (s *userService) UpdateProfile(userID, name, email string) (*models.User, error) {
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrUserNotFound
	}

	// Update name if provided
	if name != "" {
		user.Name = name
	}

	// Update email if provided and different
	if email != "" && email != user.Email {
		// Validate email
		if err := utils.ValidateEmail(email); err != nil {
			return nil, err
		}

		// Normalize email
		email = repository.NormalizeEmail(email)

		// Check if email is already taken
		existingUser, err := s.userRepo.FindByEmail(email)
		if err != nil {
			return nil, err
		}
		if existingUser != nil && existingUser.ID != userID {
			return nil, ErrUserAlreadyExists
		}

		user.Email = email
	}

	if err := s.userRepo.Update(user); err != nil {
		return nil, err
	}

	return user, nil
}

func (s *userService) ChangePassword(userID, currentPassword, newPassword string) error {
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return err
	}
	if user == nil {
		return ErrUserNotFound
	}

	// Verify current password
	if !utils.VerifyPassword(currentPassword, user.PasswordHash) {
		return ErrCurrentPasswordIncorrect
	}

	// Validate new password
	if err := utils.ValidatePassword(newPassword); err != nil {
		return err
	}

	// Hash new password
	passwordHash, err := utils.HashPassword(newPassword, s.config.BcryptCost)
	if err != nil {
		return err
	}

	user.PasswordHash = passwordHash
	return s.userRepo.Update(user)
}

func (s *userService) CompleteOnboarding(userID string) (*models.User, error) {
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrUserNotFound
	}

	user.HasCompletedOnboarding = true
	if err := s.userRepo.Update(user); err != nil {
		return nil, err
	}

	return user, nil
}

func (s *userService) UpdatePreferences(userID string, preferences *models.UserPreferences) (*models.User, error) {
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrUserNotFound
	}

	user.Preferences = preferences
	if err := s.userRepo.Update(user); err != nil {
		return nil, err
	}

	return user, nil
}
