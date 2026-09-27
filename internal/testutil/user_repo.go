// Package testutil provides test doubles shared by package tests.
package testutil

import (
	"fmt"
	"strings"

	"github.com/meal-planner/backend/internal/models"
)

// UserRepo is an in-memory repository.UserRepository for tests.
type UserRepo struct {
	Users  map[string]*models.User
	nextID int
	// Err, when set, is returned by every method.
	Err error
}

// NewUserRepo returns an empty in-memory user repository.
func NewUserRepo() *UserRepo {
	return &UserRepo{Users: map[string]*models.User{}}
}

// Create stores the user, assigning an ID if it has none.
func (r *UserRepo) Create(user *models.User) error {
	if r.Err != nil {
		return r.Err
	}
	if user.ID == "" {
		r.nextID++
		user.ID = fmt.Sprintf("user_%d", r.nextID)
	}
	r.Users[user.ID] = user
	return nil
}

// FindByEmail returns the user with the given email (case-insensitive), or nil.
func (r *UserRepo) FindByEmail(email string) (*models.User, error) {
	if r.Err != nil {
		return nil, r.Err
	}
	for _, u := range r.Users {
		if strings.EqualFold(u.Email, email) {
			return u, nil
		}
	}
	return nil, nil
}

// FindByID returns the user with the given ID, or nil.
func (r *UserRepo) FindByID(id string) (*models.User, error) {
	if r.Err != nil {
		return nil, r.Err
	}
	return r.Users[id], nil
}

// Update replaces the stored user.
func (r *UserRepo) Update(user *models.User) error {
	if r.Err != nil {
		return r.Err
	}
	r.Users[user.ID] = user
	return nil
}

// Delete removes the user with the given ID.
func (r *UserRepo) Delete(id string) error {
	if r.Err != nil {
		return r.Err
	}
	delete(r.Users, id)
	return nil
}
