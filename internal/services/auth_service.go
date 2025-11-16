package services

import (
	"errors"
	"time"

	"github.com/meal-planner/backend/internal/config"
	"github.com/meal-planner/backend/internal/models"
	"github.com/meal-planner/backend/internal/repository"
	"github.com/meal-planner/backend/internal/utils"
)

const (
	MaxLoginAttempts = 3
	LockDuration     = 5 * time.Minute
)

var (
	ErrUserAlreadyExists   = errors.New("user with this email already exists")
	ErrInvalidCredentials  = errors.New("invalid email or password")
	ErrAccountLocked       = errors.New("account is locked due to too many failed login attempts")
	ErrUserNotFound        = errors.New("user not found")
)

type AuthService interface {
	Register(email, password, name string) (*models.User, string, error)
	Login(email, password string) (*models.User, string, error)
	RefreshToken(token string) (string, error)
	ValidateToken(token string) (*models.User, error)
}

type authService struct {
	userRepo repository.UserRepository
	config   *config.Config
}

func NewAuthService(userRepo repository.UserRepository, cfg *config.Config) AuthService {
	return &authService{
		userRepo: userRepo,
		config:   cfg,
	}
}

func (s *authService) Register(email, password, name string) (*models.User, string, error) {
	// Normalize email
	email = repository.NormalizeEmail(email)

	// Validate email format
	if err := utils.ValidateEmail(email); err != nil {
		return nil, "", err
	}

	// Validate password strength
	if err := utils.ValidatePassword(password); err != nil {
		return nil, "", err
	}

	// Check if user already exists
	existingUser, err := s.userRepo.FindByEmail(email)
	if err != nil {
		return nil, "", err
	}
	if existingUser != nil {
		return nil, "", ErrUserAlreadyExists
	}

	// Hash password
	passwordHash, err := utils.HashPassword(password, s.config.BcryptCost)
	if err != nil {
		return nil, "", err
	}

	// Create user
	user := &models.User{
		Email:                  email,
		Name:                   name,
		PasswordHash:           passwordHash,
		HasCompletedOnboarding: false,
	}

	if err := s.userRepo.Create(user); err != nil {
		return nil, "", err
	}

	// Generate JWT token
	token, err := utils.GenerateToken(user.ID, user.Email, s.config.JWTSecret, s.config.GetJWTExpiration())
	if err != nil {
		return nil, "", err
	}

	return user, token, nil
}

func (s *authService) Login(email, password string) (*models.User, string, error) {
	// Normalize email
	email = repository.NormalizeEmail(email)

	// Find user by email
	user, err := s.userRepo.FindByEmail(email)
	if err != nil {
		return nil, "", err
	}
	if user == nil {
		return nil, "", ErrInvalidCredentials
	}

	// Check if account is locked
	if user.IsAccountLocked() {
		remainingTime := time.Until(*user.AccountLockedUntil)
		minutes := int(remainingTime.Minutes()) + 1
		return nil, "", errors.New("account is locked. Please try again in " + string(rune(minutes)) + " minute(s)")
	}

	// Verify password
	if !utils.VerifyPassword(password, user.PasswordHash) {
		// Increment failed login attempts
		user.IncrementLoginAttempts(MaxLoginAttempts, LockDuration)
		s.userRepo.Update(user)

		if user.IsAccountLocked() {
			return nil, "", ErrAccountLocked
		}
		return nil, "", ErrInvalidCredentials
	}

	// Reset login attempts on successful login
	user.ResetLoginAttempts()
	if err := s.userRepo.Update(user); err != nil {
		return nil, "", err
	}

	// Generate JWT token
	token, err := utils.GenerateToken(user.ID, user.Email, s.config.JWTSecret, s.config.GetJWTExpiration())
	if err != nil {
		return nil, "", err
	}

	return user, token, nil
}

func (s *authService) RefreshToken(token string) (string, error) {
	// Validate the existing token
	claims, err := utils.ValidateToken(token, s.config.JWTSecret)
	if err != nil {
		return "", err
	}

	// Generate new token
	newToken, err := utils.GenerateToken(claims.UserID, claims.Email, s.config.JWTSecret, s.config.GetJWTExpiration())
	if err != nil {
		return "", err
	}

	return newToken, nil
}

func (s *authService) ValidateToken(token string) (*models.User, error) {
	// Validate token and extract claims
	claims, err := utils.ValidateToken(token, s.config.JWTSecret)
	if err != nil {
		return nil, err
	}

	// Fetch user from database
	user, err := s.userRepo.FindByID(claims.UserID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrUserNotFound
	}

	return user, nil
}
