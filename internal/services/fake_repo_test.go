package services

import (
	"errors"
	"fmt"
	"strings"

	"github.com/meal-planner/backend/internal/config"
	"github.com/meal-planner/backend/internal/models"
	"github.com/meal-planner/backend/internal/utils"
)

// fakeUserRepo is an in-memory repository.UserRepository for tests.
type fakeUserRepo struct {
	users  map[string]*models.User
	nextID int
	// err, when set, is returned by every method.
	err error
}

func newFakeUserRepo() *fakeUserRepo {
	return &fakeUserRepo{users: map[string]*models.User{}}
}

func (r *fakeUserRepo) Create(user *models.User) error {
	if r.err != nil {
		return r.err
	}
	if user.ID == "" {
		r.nextID++
		user.ID = fmt.Sprintf("user_%d", r.nextID)
	}
	r.users[user.ID] = user
	return nil
}

func (r *fakeUserRepo) FindByEmail(email string) (*models.User, error) {
	if r.err != nil {
		return nil, r.err
	}
	for _, u := range r.users {
		if strings.EqualFold(u.Email, email) {
			return u, nil
		}
	}
	return nil, nil
}

func (r *fakeUserRepo) FindByID(id string) (*models.User, error) {
	if r.err != nil {
		return nil, r.err
	}
	return r.users[id], nil
}

func (r *fakeUserRepo) Update(user *models.User) error {
	if r.err != nil {
		return r.err
	}
	r.users[user.ID] = user
	return nil
}

func (r *fakeUserRepo) Delete(id string) error {
	if r.err != nil {
		return r.err
	}
	delete(r.users, id)
	return nil
}

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
func seedUser(repo *fakeUserRepo, email string) *models.User {
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
