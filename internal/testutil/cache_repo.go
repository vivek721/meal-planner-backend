package testutil

import (
	"sync"
	"time"

	"github.com/meal-planner/backend/internal/models"
)

// CacheRepo is an in-memory repository.CacheRepository for tests. It is safe
// for concurrent use; set Err before the calls under test start.
type CacheRepo struct {
	Entries map[string]*models.CachedResponse
	// Err, when set, is returned by every method.
	Err error

	mu sync.Mutex
}

// NewCacheRepo returns an empty in-memory cache repository.
func NewCacheRepo() *CacheRepo {
	return &CacheRepo{Entries: map[string]*models.CachedResponse{}}
}

// Get returns a copy of the entry for key, or nil.
func (r *CacheRepo) Get(key string) (*models.CachedResponse, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
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
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.Err != nil {
		return r.Err
	}
	cp := *entry
	r.Entries[entry.Key] = &cp
	return nil
}

// DeleteExpiredBefore removes entries whose ExpiresAt is before cutoff and
// returns how many were removed.
func (r *CacheRepo) DeleteExpiredBefore(cutoff time.Time) (int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.Err != nil {
		return 0, r.Err
	}
	var n int64
	for k, e := range r.Entries {
		if e.ExpiresAt.Before(cutoff) {
			delete(r.Entries, k)
			n++
		}
	}
	return n, nil
}
