package repository

import (
	"errors"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/meal-planner/backend/internal/models"
)

// CacheRepository stores cached TheMealDB responses.
type CacheRepository interface {
	// Get returns the entry for key, or (nil, nil) if there is none.
	Get(key string) (*models.CachedResponse, error)
	// Upsert inserts the entry or replaces the existing one with the same key.
	Upsert(entry *models.CachedResponse) error
}

type cacheRepository struct {
	db *gorm.DB
}

// NewCacheRepository returns a GORM-backed CacheRepository.
func NewCacheRepository(db *gorm.DB) CacheRepository {
	return &cacheRepository{db: db}
}

func (r *cacheRepository) Get(key string) (*models.CachedResponse, error) {
	var entry models.CachedResponse
	err := r.db.Where("key = ?", key).First(&entry).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &entry, nil
}

func (r *cacheRepository) Upsert(entry *models.CachedResponse) error {
	return r.db.Clauses(clause.OnConflict{UpdateAll: true}).Create(entry).Error
}
