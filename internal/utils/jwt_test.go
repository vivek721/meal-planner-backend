package utils

import (
	"errors"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const jwtTestSecret = "test-secret-key-for-unit-tests-32-chars"

func TestGenerateToken_RoundTrip(t *testing.T) {
	before := time.Now().Add(-time.Second)
	token, err := GenerateToken("user_1", "user@example.com", jwtTestSecret, time.Hour)
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}

	claims, err := ValidateToken(token, jwtTestSecret)
	if err != nil {
		t.Fatalf("ValidateToken() error = %v", err)
	}
	if claims.UserID != "user_1" {
		t.Errorf("UserID = %q, want user_1", claims.UserID)
	}
	if claims.Email != "user@example.com" {
		t.Errorf("Email = %q, want user@example.com", claims.Email)
	}
	if claims.IssuedAt == nil || claims.IssuedAt.Before(before) {
		t.Errorf("IssuedAt = %v, want around now", claims.IssuedAt)
	}
	wantExpiry := time.Now().Add(time.Hour)
	if claims.ExpiresAt == nil || claims.ExpiresAt.Sub(wantExpiry).Abs() > 5*time.Second {
		t.Errorf("ExpiresAt = %v, want about %v", claims.ExpiresAt, wantExpiry)
	}
}

func TestValidateToken_Expired(t *testing.T) {
	token, err := GenerateToken("user_1", "user@example.com", jwtTestSecret, -time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ValidateToken(token, jwtTestSecret); !errors.Is(err, ErrExpiredToken) {
		t.Errorf("ValidateToken(expired) error = %v, want ErrExpiredToken", err)
	}
}

func TestValidateToken_WrongSecret(t *testing.T) {
	token, err := GenerateToken("user_1", "user@example.com", jwtTestSecret, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ValidateToken(token, "a-different-secret-that-is-long-enough"); !errors.Is(err, ErrInvalidToken) {
		t.Errorf("ValidateToken(wrong secret) error = %v, want ErrInvalidToken", err)
	}
}

func TestValidateToken_Malformed(t *testing.T) {
	for _, token := range []string{"", "not-a-jwt", "a.b.c"} {
		if _, err := ValidateToken(token, jwtTestSecret); !errors.Is(err, ErrInvalidToken) {
			t.Errorf("ValidateToken(%q) error = %v, want ErrInvalidToken", token, err)
		}
	}
}

func TestValidateToken_RejectsNonHMACAlgorithms(t *testing.T) {
	claims := JWTClaims{
		UserID: "user_1",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	}
	// An unsigned "alg: none" token must never be accepted.
	token, err := jwt.NewWithClaims(jwt.SigningMethodNone, claims).SignedString(jwt.UnsafeAllowNoneSignatureType)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ValidateToken(token, jwtTestSecret); !errors.Is(err, ErrInvalidToken) {
		t.Errorf("ValidateToken(alg=none) error = %v, want ErrInvalidToken", err)
	}
}

func TestValidateToken_NotYetValid(t *testing.T) {
	claims := JWTClaims{
		UserID: "user_1",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(2 * time.Hour)),
			NotBefore: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(jwtTestSecret))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ValidateToken(token, jwtTestSecret); !errors.Is(err, ErrInvalidToken) {
		t.Errorf("ValidateToken(nbf in future) error = %v, want ErrInvalidToken", err)
	}
}
