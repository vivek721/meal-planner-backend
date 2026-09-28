package services

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/meal-planner/backend/internal/mealdb"
	"github.com/meal-planner/backend/internal/models"
	"github.com/meal-planner/backend/internal/testutil"
)

type recipeFixture struct {
	svc    RecipeService
	client *testutil.MealDBClient
	cache  *testutil.CacheRepo
	now    time.Time
}

func newRecipeFixture(t *testing.T) *recipeFixture {
	t.Helper()
	f := &recipeFixture{
		client: testutil.NewMealDBClient(),
		cache:  testutil.NewCacheRepo(),
		now:    time.Date(2026, 9, 28, 12, 0, 0, 0, time.UTC),
	}
	f.svc = NewRecipeService(f.client, f.cache, RecipeCacheTTL{Detail: 7 * 24 * time.Hour, Search: 24 * time.Hour},
		func() time.Time { return f.now })
	return f
}

var teriyaki = mealdb.Meal{
	ID: "52772", Name: "Teriyaki", Category: "Chicken", Area: "Japanese",
	Ingredients: []mealdb.Ingredient{}, Instructions: []string{"Cook."}, Tags: []string{},
}

func TestGetCachesAndReusesFreshEntry(t *testing.T) {
	f := newRecipeFixture(t)
	f.client.Meals["52772"] = teriyaki
	ctx := context.Background()

	for i := 0; i < 2; i++ {
		got, err := f.svc.Get(ctx, "52772")
		if err != nil || !reflect.DeepEqual(*got, teriyaki) {
			t.Fatalf("call %d: %+v, %v", i, got, err)
		}
	}
	if f.client.Calls != 1 {
		t.Errorf("fresh cache hit must not call upstream; calls = %d", f.client.Calls)
	}
	entry := f.cache.Entries["lookup:52772"]
	if entry == nil || !entry.ExpiresAt.Equal(f.now.Add(7*24*time.Hour)) {
		t.Errorf("cache entry/TTL wrong: %+v", entry)
	}
}

func TestGetRefetchesExpiredEntry(t *testing.T) {
	f := newRecipeFixture(t)
	f.client.Meals["52772"] = teriyaki
	ctx := context.Background()
	_, _ = f.svc.Get(ctx, "52772")

	f.now = f.now.Add(8 * 24 * time.Hour)
	updated := teriyaki
	updated.Name = "Teriyaki v2"
	f.client.Meals["52772"] = updated
	got, err := f.svc.Get(ctx, "52772")
	if err != nil || got.Name != "Teriyaki v2" || f.client.Calls != 2 {
		t.Fatalf("expired entry must be refetched: %+v, %v, calls=%d", got, err, f.client.Calls)
	}
}

func TestGetServesStaleOnUpstreamError(t *testing.T) {
	f := newRecipeFixture(t)
	f.client.Meals["52772"] = teriyaki
	ctx := context.Background()
	_, _ = f.svc.Get(ctx, "52772")

	f.now = f.now.Add(30 * 24 * time.Hour)
	f.client.Err = errors.New("upstream down")
	got, err := f.svc.Get(ctx, "52772")
	if err != nil || got.Name != "Teriyaki" {
		t.Fatalf("stale entry must be served when upstream fails: %+v, %v", got, err)
	}
}

func TestGetUpstreamErrorWithNothingCached(t *testing.T) {
	f := newRecipeFixture(t)
	f.client.Err = errors.New("upstream down")
	if _, err := f.svc.Get(context.Background(), "52772"); !errors.Is(err, ErrUpstreamUnavailable) {
		t.Fatalf("err = %v, want ErrUpstreamUnavailable", err)
	}
}

func TestGetNotFound(t *testing.T) {
	f := newRecipeFixture(t)
	if _, err := f.svc.Get(context.Background(), "99999"); !errors.Is(err, ErrRecipeNotFound) {
		t.Fatalf("err = %v, want ErrRecipeNotFound", err)
	}
}

func TestCorruptOrUnreadableCacheFallsThroughToFetch(t *testing.T) {
	f := newRecipeFixture(t)
	f.client.Meals["52772"] = teriyaki
	f.cache.Entries["lookup:52772"] = &models.CachedResponse{
		Key: "lookup:52772", Payload: `{not json`,
		FetchedAt: f.now, ExpiresAt: f.now.Add(time.Hour),
	}
	if got, err := f.svc.Get(context.Background(), "52772"); err != nil || got.Name != "Teriyaki" {
		t.Fatalf("corrupt fresh entry must be refetched: %+v, %v", got, err)
	}

	f.cache.Err = errors.New("db down")
	if got, err := f.svc.Get(context.Background(), "52772"); err != nil || got.Name != "Teriyaki" {
		t.Fatalf("cache read error must fall through to upstream: %+v, %v", got, err)
	}
}

func TestCuisinesSortedTrimmedWithoutUnknown(t *testing.T) {
	f := newRecipeFixture(t)
	f.client.AreaList = []string{"Italian", " American ", "Unknown", "British"}
	got, err := f.svc.Cuisines(context.Background())
	if err != nil || !reflect.DeepEqual(got, []string{"American", "British", "Italian"}) {
		t.Fatalf("Cuisines = %v, %v", got, err)
	}
}

func TestCategoriesCached(t *testing.T) {
	f := newRecipeFixture(t)
	f.client.CategoryList = []mealdb.Category{{Name: "Beef"}}
	ctx := context.Background()
	_, _ = f.svc.Categories(ctx)
	got, err := f.svc.Categories(ctx)
	if err != nil || len(got) != 1 || f.client.Calls != 1 {
		t.Fatalf("Categories = %v, %v, calls=%d", got, err, f.client.Calls)
	}
}
