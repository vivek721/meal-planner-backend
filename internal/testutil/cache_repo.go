package testutil

import "github.com/meal-planner/backend/internal/models"

// CacheRepo is an in-memory repository.CacheRepository for tests.
type CacheRepo struct {
	Entries map[string]*models.CachedResponse
	// Err, when set, is returned by every method.
	Err error
}

// NewCacheRepo returns an empty in-memory cache repository.
func NewCacheRepo() *CacheRepo {
	return &CacheRepo{Entries: map[string]*models.CachedResponse{}}
}

// Get returns a copy of the entry for key, or nil.
func (r *CacheRepo) Get(key string) (*models.CachedResponse, error) {
	if r.Err != nil {
		return nil, r.Err
	}
	e, ok := r.Entries[key]
	if !ok {
		return nil, nil
	}
	cp := *e
	return &cp, nil
}

// Upsert stores a copy of the entry.
func (r *CacheRepo) Upsert(entry *models.CachedResponse) error {
	if r.Err != nil {
		return r.Err
	}
	cp := *entry
	r.Entries[entry.Key] = &cp
	return nil
}
