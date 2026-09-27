package services

import (
	"errors"

	"github.com/meal-planner/backend/internal/config"
	"github.com/meal-planner/backend/internal/models"
	"github.com/meal-planner/backend/internal/testutil"
	"github.com/meal-planner/backend/internal/utils"
)

var errRepo = errors.New("repository failure")

const (
	testSecret   = "test-secret-key-for-unit-tests-32-chars"
	testPassword = "Password1!"
)

func testConfig() *config.Config {
	return &config.Config{
		JWTSecret:          testSecret,
		JWTExpirationHours: 1,
		BcryptCost:         4, // bcrypt.MinCost keeps tests fast
	}
}

// seedUser stores a user with testPassword and returns it.
func seedUser(repo *testutil.UserRepo, email string) *models.User {
	hash, err := utils.HashPassword(testPassword, 4)
	if err != nil {
		panic(err)
	}
	u := &models.User{Email: email, Name: "Test", PasswordHash: hash}
	if err := repo.Create(u); err != nil {
		panic(err)
	}
	return u
}
