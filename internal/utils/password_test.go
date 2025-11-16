package utils

import (
	"testing"
)

func TestHashPassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
		cost     int
		wantErr  bool
	}{
		{
			name:     "valid password",
			password: "SecurePassword123!",
			cost:     10,
			wantErr:  false,
		},
		{
			name:     "empty password",
			password: "",
			cost:     10,
			wantErr:  false, // bcrypt allows empty passwords
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash, err := HashPassword(tt.password, tt.cost)
			if (err != nil) != tt.wantErr {
				t.Errorf("HashPassword() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && hash == "" {
				t.Error("HashPassword() returned empty hash")
			}
		})
	}
}

func TestVerifyPassword(t *testing.T) {
	password := "TestPassword123!"
	hash, _ := HashPassword(password, 10)

	tests := []struct {
		name     string
		password string
		hash     string
		want     bool
	}{
		{
			name:     "correct password",
			password: password,
			hash:     hash,
			want:     true,
		},
		{
			name:     "incorrect password",
			password: "WrongPassword123!",
			hash:     hash,
			want:     false,
		},
		{
			name:     "empty password",
			password: "",
			hash:     hash,
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := VerifyPassword(tt.password, tt.hash); got != tt.want {
				t.Errorf("VerifyPassword() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPasswordHashingRoundTrip(t *testing.T) {
	passwords := []string{
		"SimplePass123!",
		"Complex@Password#2024",
		"Th1s!s@V3ryL0ngP@ssw0rd",
	}

	for _, password := range passwords {
		t.Run(password, func(t *testing.T) {
			hash, err := HashPassword(password, 10)
			if err != nil {
				t.Fatalf("HashPassword() failed: %v", err)
			}

			if !VerifyPassword(password, hash) {
				t.Error("VerifyPassword() failed for correct password")
			}

			if VerifyPassword("wrong"+password, hash) {
				t.Error("VerifyPassword() succeeded for incorrect password")
			}
		})
	}
}
