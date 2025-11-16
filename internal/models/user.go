package models

import (
	"time"

	"gorm.io/gorm"
)

// User represents a user in the system
type User struct {
	ID                      string         `gorm:"type:varchar(255);primaryKey" json:"id"`
	Email                   string         `gorm:"type:varchar(255);uniqueIndex;not null" json:"email"`
	Name                    string         `gorm:"type:varchar(255)" json:"name,omitempty"`
	PasswordHash            string         `gorm:"type:varchar(255);not null" json:"-"`
	HasCompletedOnboarding  bool           `gorm:"default:false" json:"hasCompletedOnboarding"`
	CreatedAt               time.Time      `json:"createdAt"`
	UpdatedAt               time.Time      `json:"updatedAt"`
	DeletedAt               gorm.DeletedAt `gorm:"index" json:"-"`

	// Login tracking
	LoginAttempts           int            `gorm:"default:0" json:"-"`
	LastLoginAttempt        *time.Time     `json:"-"`
	AccountLockedUntil      *time.Time     `json:"-"`

	// Preferences
	Preferences             *UserPreferences `gorm:"embedded;embeddedPrefix:pref_" json:"preferences,omitempty"`
}

// UserPreferences stores user preferences
type UserPreferences struct {
	Theme         string `gorm:"type:varchar(50);default:'light'" json:"theme,omitempty"`
	Notifications bool   `gorm:"default:true" json:"notifications,omitempty"`
}

// LoginAttemptInfo represents login attempt tracking information
type LoginAttemptInfo struct {
	Count       int        `json:"count"`
	LastAttempt *time.Time `json:"lastAttempt,omitempty"`
	LockedUntil *time.Time `json:"lockedUntil,omitempty"`
}

// BeforeCreate hook to generate ID if not set
func (u *User) BeforeCreate(tx *gorm.DB) error {
	if u.ID == "" {
		u.ID = generateID("user")
	}
	if u.CreatedAt.IsZero() {
		u.CreatedAt = time.Now()
	}
	return nil
}

// ToPublicUser returns a sanitized user object for API responses
func (u *User) ToPublicUser() *PublicUser {
	return &PublicUser{
		ID:                     u.ID,
		Email:                  u.Email,
		Name:                   u.Name,
		HasCompletedOnboarding: u.HasCompletedOnboarding,
		CreatedAt:              u.CreatedAt.Format(time.RFC3339),
		Preferences:            u.Preferences,
	}
}

// PublicUser represents user data safe for API responses (no sensitive fields)
type PublicUser struct {
	ID                     string            `json:"id"`
	Email                  string            `json:"email"`
	Name                   string            `json:"name,omitempty"`
	HasCompletedOnboarding bool              `json:"hasCompletedOnboarding"`
	CreatedAt              string            `json:"createdAt"`
	Preferences            *UserPreferences  `json:"preferences,omitempty"`
}

// GetLoginAttemptInfo returns login attempt information
func (u *User) GetLoginAttemptInfo() *LoginAttemptInfo {
	return &LoginAttemptInfo{
		Count:       u.LoginAttempts,
		LastAttempt: u.LastLoginAttempt,
		LockedUntil: u.AccountLockedUntil,
	}
}

// IsAccountLocked checks if the account is currently locked
func (u *User) IsAccountLocked() bool {
	if u.AccountLockedUntil == nil {
		return false
	}
	return time.Now().Before(*u.AccountLockedUntil)
}

// ResetLoginAttempts resets login attempt counter
func (u *User) ResetLoginAttempts() {
	u.LoginAttempts = 0
	u.AccountLockedUntil = nil
	now := time.Now()
	u.LastLoginAttempt = &now
}

// IncrementLoginAttempts increments failed login attempts
func (u *User) IncrementLoginAttempts(maxAttempts int, lockDuration time.Duration) {
	u.LoginAttempts++
	now := time.Now()
	u.LastLoginAttempt = &now

	if u.LoginAttempts >= maxAttempts {
		lockedUntil := now.Add(lockDuration)
		u.AccountLockedUntil = &lockedUntil
	}
}
