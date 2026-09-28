package services

import (
	"context"
	"log"
	"time"

	"github.com/meal-planner/backend/internal/repository"
)

// CachePurgeInterval is how often PurgeExpiredCacheLoop reruns PurgeExpiredCache.
const CachePurgeInterval = 24 * time.Hour

// PurgeExpiredCache deletes mealdb_cache rows whose ExpiresAt is older than
// retention (i.e. before now.Add(-retention)) and logs how many rows were
// removed. Rows that are expired but still within the retention window are
// kept, since the read-through cache serves them on upstream failure
// (stale-on-error). A repository error is logged and otherwise ignored: a
// failed purge is not critical enough to fail startup or crash the process.
func PurgeExpiredCache(repo repository.CacheRepository, retention time.Duration, now time.Time) {
	cutoff := now.Add(-retention)
	n, err := repo.DeleteExpiredBefore(cutoff)
	if err != nil {
		log.Printf("mealdb cache: purge failed: %v", err)
		return
	}
	log.Printf("mealdb cache: purged %d rows older than %s", n, cutoff)
}

// PurgeExpiredCacheLoop runs PurgeExpiredCache immediately and then again
// every interval, using now for the current time on each run, until ctx is
// cancelled.
func PurgeExpiredCacheLoop(ctx context.Context, repo repository.CacheRepository, retention time.Duration, now func() time.Time, interval time.Duration) {
	PurgeExpiredCache(repo, retention, now())

	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			PurgeExpiredCache(repo, retention, now())
		}
	}
}
