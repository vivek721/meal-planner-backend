package services

import (
	"bytes"
	"context"
	"errors"
	"log"
	"strings"
	"testing"
	"time"

	"github.com/meal-planner/backend/internal/models"
	"github.com/meal-planner/backend/internal/testutil"
)

func TestPurgeExpiredCacheDeletesOnlyRowsOlderThanRetention(t *testing.T) {
	repo := testutil.NewCacheRepo()
	now := time.Date(2026, 9, 28, 12, 0, 0, 0, time.UTC)
	retention := 30 * 24 * time.Hour
	cutoff := now.Add(-retention)

	// Expired well before the retention window: must be purged.
	_ = repo.Upsert(&models.CachedResponse{Key: "ancient", ExpiresAt: cutoff.Add(-time.Hour)})
	// Expired, but within the retention window: stale-on-error still needs this. Must be kept.
	_ = repo.Upsert(&models.CachedResponse{Key: "recently-expired", ExpiresAt: cutoff.Add(time.Hour)})
	// Still fresh: must be kept.
	_ = repo.Upsert(&models.CachedResponse{Key: "fresh", ExpiresAt: now.Add(time.Hour)})

	PurgeExpiredCache(repo, retention, now)

	if _, ok := repo.Entries["ancient"]; ok {
		t.Error("row expired before the retention cutoff must be purged")
	}
	if _, ok := repo.Entries["recently-expired"]; !ok {
		t.Error("row expired but within the retention window must be kept")
	}
	if _, ok := repo.Entries["fresh"]; !ok {
		t.Error("fresh row must be kept")
	}
}

func TestPurgeExpiredCacheLogsCount(t *testing.T) {
	repo := testutil.NewCacheRepo()
	now := time.Date(2026, 9, 28, 12, 0, 0, 0, time.UTC)
	retention := 30 * 24 * time.Hour
	_ = repo.Upsert(&models.CachedResponse{Key: "ancient", ExpiresAt: now.Add(-31 * 24 * time.Hour)})

	var buf bytes.Buffer
	orig := log.Writer()
	log.SetOutput(&buf)
	defer log.SetOutput(orig)

	PurgeExpiredCache(repo, retention, now)

	if got := buf.String(); !strings.Contains(got, "mealdb cache: purged 1 rows older than") {
		t.Errorf("log output = %q, want it to mention the purged count", got)
	}
}

func TestPurgeExpiredCacheRepoErrorIsLoggedNotFatal(t *testing.T) {
	repo := testutil.NewCacheRepo()
	repo.Err = errors.New("db down")

	var buf bytes.Buffer
	orig := log.Writer()
	log.SetOutput(&buf)
	defer log.SetOutput(orig)

	PurgeExpiredCache(repo, 30*24*time.Hour, time.Now()) // must not panic

	if got := buf.String(); !strings.Contains(got, "mealdb cache: purge failed") {
		t.Errorf("log output = %q, want it to mention the purge failure", got)
	}
}

// signalingRepo wraps a CacheRepo and signals on calls after every
// DeleteExpiredBefore call, so tests can observe loop iterations without
// racing on the underlying map.
type signalingRepo struct {
	*testutil.CacheRepo
	calls chan struct{}
}

func (r *signalingRepo) DeleteExpiredBefore(cutoff time.Time) (int64, error) {
	n, err := r.CacheRepo.DeleteExpiredBefore(cutoff)
	r.calls <- struct{}{}
	return n, err
}

func TestPurgeExpiredCacheLoopRunsImmediatelyAndStopsOnCancel(t *testing.T) {
	base := testutil.NewCacheRepo()
	now := time.Date(2026, 9, 28, 12, 0, 0, 0, time.UTC)
	_ = base.Upsert(&models.CachedResponse{Key: "ancient", ExpiresAt: now.Add(-31 * 24 * time.Hour)})
	repo := &signalingRepo{CacheRepo: base, calls: make(chan struct{}, 10)}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		PurgeExpiredCacheLoop(ctx, repo, 30*24*time.Hour, func() time.Time { return now }, time.Millisecond)
		close(done)
	}()

	select {
	case <-repo.calls:
		// Immediate run happened without waiting for a tick.
	case <-time.After(time.Second):
		t.Fatal("PurgeExpiredCacheLoop did not run immediately")
	}

	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("PurgeExpiredCacheLoop did not stop after context cancellation")
	}

	if _, ok := base.Entries["ancient"]; ok {
		t.Error("immediate run must purge the ancient row")
	}
}
