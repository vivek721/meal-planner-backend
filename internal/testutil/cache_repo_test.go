package testutil

import (
	"errors"
	"testing"
	"time"

	"github.com/meal-planner/backend/internal/models"
)

func TestCacheRepoUpsertAndGet(t *testing.T) {
	r := NewCacheRepo()
	if e, err := r.Get("k"); e != nil || err != nil {
		t.Fatalf("missing key: %v, %v", e, err)
	}
	_ = r.Upsert(&models.CachedResponse{Key: "k", Payload: `1`})
	_ = r.Upsert(&models.CachedResponse{Key: "k", Payload: `2`})
	if e, _ := r.Get("k"); e == nil || e.Payload != `2` {
		t.Fatalf("upsert must replace: %+v", e)
	}
	r.Err = errors.New("db down")
	if _, err := r.Get("k"); err == nil {
		t.Fatal("Err must be returned")
	}
}

func TestCacheRepoDeleteExpiredBefore(t *testing.T) {
	r := NewCacheRepo()
	cutoff := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	_ = r.Upsert(&models.CachedResponse{Key: "old", ExpiresAt: cutoff.Add(-time.Hour)})
	_ = r.Upsert(&models.CachedResponse{Key: "fresh", ExpiresAt: cutoff.Add(time.Hour)})
	_ = r.Upsert(&models.CachedResponse{Key: "boundary", ExpiresAt: cutoff})

	n, err := r.DeleteExpiredBefore(cutoff)
	if err != nil || n != 1 {
		t.Fatalf("DeleteExpiredBefore = %d, %v; want 1, nil", n, err)
	}
	if _, ok := r.Entries["old"]; ok {
		t.Error("entry expired before cutoff must be deleted")
	}
	if _, ok := r.Entries["fresh"]; !ok {
		t.Error("entry expiring after cutoff must be kept")
	}
	if _, ok := r.Entries["boundary"]; !ok {
		t.Error("entry expiring exactly at cutoff must be kept (not before)")
	}

	r.Err = errors.New("db down")
	if _, err := r.DeleteExpiredBefore(cutoff); err == nil {
		t.Fatal("Err must be returned")
	}
}
