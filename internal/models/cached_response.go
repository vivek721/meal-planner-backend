package models

import "time"

// CachedResponse is a cached, normalised TheMealDB result.
type CachedResponse struct {
	Key       string    `gorm:"type:varchar(255);primaryKey"`
	Payload   string    `gorm:"type:jsonb;not null"`
	FetchedAt time.Time `gorm:"not null"`
	ExpiresAt time.Time `gorm:"not null;index"`
}

// TableName stores cached responses in mealdb_cache.
func (CachedResponse) TableName() string { return "mealdb_cache" }
