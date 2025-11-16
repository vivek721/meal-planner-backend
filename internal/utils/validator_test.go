package utils

import (
	"testing"
)

func TestValidateEmail(t *testing.T) {
	tests := []struct {
		name    string
		email   string
		wantErr error
	}{
		{
			name:    "valid email",
			email:   "user@example.com",
			wantErr: nil,
		},
		{
			name:    "valid email with subdomain",
			email:   "user@mail.example.com",
			wantErr: nil,
		},
		{
			name:    "valid email with plus",
			email:   "user+tag@example.com",
			wantErr: nil,
		},
		{
			name:    "empty email",
			email:   "",
			wantErr: ErrEmailRequired,
		},
		{
			name:    "invalid email - no @",
			email:   "userexample.com",
			wantErr: ErrInvalidEmail,
		},
		{
			name:    "invalid email - no domain",
			email:   "user@",
			wantErr: ErrInvalidEmail,
		},
		{
			name:    "invalid email - no TLD",
			email:   "user@example",
			wantErr: ErrInvalidEmail,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateEmail(tt.email)
			if err != tt.wantErr {
				t.Errorf("ValidateEmail() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidatePassword(t *testing.T) {
	tests := []struct {
		name    string
		password string
		wantErr error
	}{
		{
			name:     "valid strong password",
			password: "SecurePass123!",
			wantErr:  nil,
		},
		{
			name:     "valid password with symbols",
			password: "P@ssw0rd#2024",
			wantErr:  nil,
		},
		{
			name:     "empty password",
			password: "",
			wantErr:  ErrPasswordRequired,
		},
		{
			name:     "too short",
			password: "Pass1!",
			wantErr:  ErrPasswordTooShort,
		},
		{
			name:     "no uppercase",
			password: "password123!",
			wantErr:  ErrPasswordTooWeak,
		},
		{
			name:     "no lowercase",
			password: "PASSWORD123!",
			wantErr:  ErrPasswordTooWeak,
		},
		{
			name:     "no number",
			password: "Password!@#",
			wantErr:  ErrPasswordTooWeak,
		},
		{
			name:     "no special char",
			password: "Password123",
			wantErr:  ErrPasswordTooWeak,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePassword(tt.password)
			if err != tt.wantErr {
				t.Errorf("ValidatePassword() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateRegistration(t *testing.T) {
	tests := []struct {
		name            string
		email           string
		password        string
		confirmPassword string
		wantErr         error
	}{
		{
			name:            "valid registration",
			email:           "user@example.com",
			password:        "SecurePass123!",
			confirmPassword: "SecurePass123!",
			wantErr:         nil,
		},
		{
			name:            "invalid email",
			email:           "invalid-email",
			password:        "SecurePass123!",
			confirmPassword: "SecurePass123!",
			wantErr:         ErrInvalidEmail,
		},
		{
			name:            "weak password",
			email:           "user@example.com",
			password:        "weak",
			confirmPassword: "weak",
			wantErr:         ErrPasswordTooShort,
		},
		{
			name:            "passwords do not match",
			email:           "user@example.com",
			password:        "SecurePass123!",
			confirmPassword: "DifferentPass123!",
			wantErr:         ErrPasswordsDoNotMatch,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateRegistration(tt.email, tt.password, tt.confirmPassword)
			if err != tt.wantErr {
				t.Errorf("ValidateRegistration() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
