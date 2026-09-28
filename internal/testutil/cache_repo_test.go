package testutil

import (
	"errors"
	"testing"

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
