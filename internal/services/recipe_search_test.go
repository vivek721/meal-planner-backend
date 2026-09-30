package services

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/meal-planner/backend/internal/mealdb"
	"github.com/meal-planner/backend/internal/testutil"
)

func ref(id string) mealdb.MealRef {
	return mealdb.MealRef{ID: id, Name: "Meal " + id, Thumbnail: "t" + id}
}

func ids(p *RecipePage) []string {
	out := make([]string, 0, len(p.Recipes))
	for _, r := range p.Recipes {
		out = append(out, r.ID)
	}
	return out
}

func TestSearchRequiresCriteria(t *testing.T) {
	f := newRecipeFixture(t)
	for _, q := range []RecipeQuery{{}, {Q: "  ", Category: "\t"}} {
		if _, err := f.svc.Search(context.Background(), q); !errors.Is(err, ErrInvalidSearch) {
			t.Errorf("%+v: err = %v, want ErrInvalidSearch", q, err)
		}
	}
	if f.client.Calls != 0 {
		t.Errorf("invalid search must not call upstream")
	}
}

func TestSearchSingleFilterSetsKnownFields(t *testing.T) {
	f := newRecipeFixture(t)
	f.client.Filters["c=Seafood"] = []mealdb.MealRef{ref("1"), ref("2")}
	p, err := f.svc.Search(context.Background(), RecipeQuery{Category: " Seafood "})
	if err != nil || p.Total != 2 || p.Recipes[0] != (RecipeSummary{ID: "1", Name: "Meal 1", Thumbnail: "t1", Category: "Seafood"}) {
		t.Fatalf("Search = %+v, %v", p, err)
	}
}

func TestSearchIntersectsFiltersKeepingFirstOrder(t *testing.T) {
	f := newRecipeFixture(t)
	f.client.Filters["c=Vegetarian"] = []mealdb.MealRef{ref("3"), ref("1"), ref("2")}
	f.client.Filters["a=Italian"] = []mealdb.MealRef{ref("2"), ref("3"), ref("9")}
	f.client.Filters["i=garlic"] = []mealdb.MealRef{ref("3"), ref("2"), ref("7")}
	p, err := f.svc.Search(context.Background(), RecipeQuery{Category: "Vegetarian", Cuisine: "Italian", Ingredient: "garlic"})
	if err != nil || fmt.Sprint(ids(p)) != "[3 2]" || p.Recipes[0].Cuisine != "Italian" || p.Recipes[0].Category != "Vegetarian" {
		t.Fatalf("Search = %+v, %v", p, err)
	}
}

func TestSearchByNameNarrowedByFilters(t *testing.T) {
	f := newRecipeFixture(t)
	// the fake is keyed by the query exactly as the service passes it (trimmed, case kept)
	f.client.SearchResults["Chicken"] = []mealdb.Meal{
		{ID: "1", Name: "Chicken Tikka", Category: "Chicken", Area: "Indian"},
		{ID: "2", Name: "Chicken Parm", Category: "Chicken", Area: "Italian"},
		{ID: "3", Name: "Chicken Soup", Category: "Starter", Area: "Italian"},
	}
	f.client.Filters["i=garlic"] = []mealdb.MealRef{ref("2"), ref("3")}
	p, err := f.svc.Search(context.Background(), RecipeQuery{Q: "Chicken", Cuisine: "italian", Ingredient: "garlic"})
	if err != nil || fmt.Sprint(ids(p)) != "[2 3]" || p.Recipes[1] != (RecipeSummary{ID: "3", Name: "Chicken Soup", Category: "Starter", Cuisine: "Italian"}) {
		t.Fatalf("Search = %+v, %v", p, err)
	}
	if _, ok := f.cache.Entries["search:q=chicken"]; !ok {
		t.Error("name search must be cached under a lower-cased key")
	}
}

func TestSearchPaging(t *testing.T) {
	f := newRecipeFixture(t)
	refs := make([]mealdb.MealRef, 0, 60)
	for i := 1; i <= 60; i++ {
		refs = append(refs, ref(fmt.Sprint(i)))
	}
	f.client.Filters["c=Beef"] = refs
	ctx := context.Background()

	p, _ := f.svc.Search(ctx, RecipeQuery{Category: "Beef"})
	if len(p.Recipes) != 24 || p.Page != 1 || p.Total != 60 || p.TotalPages != 3 {
		t.Errorf("default paging: len=%d page=%d total=%d pages=%d", len(p.Recipes), p.Page, p.Total, p.TotalPages)
	}
	p, _ = f.svc.Search(ctx, RecipeQuery{Category: "Beef", Page: 2, Limit: 500})
	if len(p.Recipes) != 10 || p.Recipes[0].ID != "51" || p.TotalPages != 2 {
		t.Errorf("limit must cap at 50: len=%d first=%s pages=%d", len(p.Recipes), p.Recipes[0].ID, p.TotalPages)
	}
	p, _ = f.svc.Search(ctx, RecipeQuery{Category: "Beef", Page: 9})
	if p.Recipes == nil || len(p.Recipes) != 0 || p.Total != 60 {
		t.Errorf("page past the end must be an empty non-nil list: %+v", p)
	}
}

func TestSearchPagingHugePageDoesNotPanic(t *testing.T) {
	f := newRecipeFixture(t)
	refs := make([]mealdb.MealRef, 0, 60)
	for i := 1; i <= 60; i++ {
		refs = append(refs, ref(fmt.Sprint(i)))
	}
	f.client.Filters["c=Beef"] = refs
	ctx := context.Background()

	for _, page := range []int{math.MaxInt, 384307168202282327} {
		p, err := f.svc.Search(ctx, RecipeQuery{Category: "Beef", Page: page})
		if err != nil {
			t.Fatalf("page %d: unexpected error %v", page, err)
		}
		if p.Recipes == nil || len(p.Recipes) != 0 {
			t.Errorf("page %d: Recipes = %+v, want empty non-nil", page, p.Recipes)
		}
		if p.Total != 60 || p.TotalPages != 3 || p.Page != page {
			t.Errorf("page %d: Total=%d TotalPages=%d Page=%d, want Total=60 TotalPages=3 Page=%d",
				page, p.Total, p.TotalPages, p.Page, page)
		}
	}
}

func TestSearchRejectsOverlongParams(t *testing.T) {
	f := newRecipeFixture(t)
	long := strings.Repeat("a", 101)
	for _, q := range []RecipeQuery{
		{Q: long}, {Category: long}, {Cuisine: long}, {Ingredient: long},
	} {
		if _, err := f.svc.Search(context.Background(), q); !errors.Is(err, ErrSearchTooLong) {
			t.Errorf("%+v: err = %v, want ErrSearchTooLong", q, err)
		}
	}
	if f.client.Calls != 0 {
		t.Errorf("overlong search must not call upstream, calls = %d", f.client.Calls)
	}

	// Rune count, not byte count: 100 multi-byte runes must be accepted.
	f.client.Filters["c="+strings.Repeat("é", 100)] = []mealdb.MealRef{ref("1")}
	if _, err := f.svc.Search(context.Background(), RecipeQuery{Category: strings.Repeat("é", 100)}); err != nil {
		t.Errorf("100 multi-byte runes must be accepted: %v", err)
	}
}

func TestSearchNoMatchesIsEmptyNotNil(t *testing.T) {
	f := newRecipeFixture(t)
	p, err := f.svc.Search(context.Background(), RecipeQuery{Q: "zzz"})
	if err != nil || p.Recipes == nil || p.Total != 0 || p.TotalPages != 0 || p.Page != 1 {
		t.Fatalf("no matches: %+v, %v", p, err)
	}
}

func TestSearchUpstreamErrorPropagates(t *testing.T) {
	f := newRecipeFixture(t)
	f.client.Err = errors.New("down")
	if _, err := f.svc.Search(context.Background(), RecipeQuery{Category: "Beef"}); !errors.Is(err, ErrUpstreamUnavailable) {
		t.Fatalf("err = %v, want ErrUpstreamUnavailable", err)
	}
}

func TestSearchFilterSpellingsShareOneCacheEntry(t *testing.T) {
	f := newRecipeFixture(t)
	f.client.Filters["c=Beef"] = []mealdb.MealRef{ref("1"), ref("2")}
	f.client.Filters["i=Chicken_Breast"] = []mealdb.MealRef{ref("3")}
	for _, q := range []RecipeQuery{{Category: "Beef"}, {Category: "BEEF"}, {Category: " beef "}} {
		if p, err := f.svc.Search(context.Background(), q); err != nil || fmt.Sprint(ids(p)) != "[1 2]" {
			t.Fatalf("%+v: Search = %+v, %v", q, p, err)
		}
	}
	for _, q := range []RecipeQuery{{Ingredient: "Chicken_Breast"}, {Ingredient: "chicken  breast"}} {
		if p, err := f.svc.Search(context.Background(), q); err != nil || fmt.Sprint(ids(p)) != "[3]" {
			t.Fatalf("%+v: Search = %+v, %v", q, p, err)
		}
	}
	if f.client.FilterCalls != 2 {
		t.Errorf("FilterCalls = %d, want 2 (one per distinct filter, whatever the spelling)", f.client.FilterCalls)
	}
	for _, key := range []string{"filter:c=beef", "filter:i=chicken breast"} {
		if _, ok := f.cache.Entries[key]; !ok {
			t.Errorf("missing normalised cache key %q", key)
		}
	}
}

func TestSearchFilterReportsCanonicalLabels(t *testing.T) {
	f := newRecipeFixture(t)
	f.client.CategoryList = []mealdb.Category{{Name: "Beef"}, {Name: "Seafood"}}
	f.client.AreaList = []string{"Canadian", "Italian"}
	f.client.Filters["c=beef"] = []mealdb.MealRef{ref("1")}
	f.client.Filters["a=CANADIAN"] = []mealdb.MealRef{ref("1")}
	p, err := f.svc.Search(context.Background(), RecipeQuery{Category: "beef", Cuisine: "CANADIAN"})
	if err != nil || p.Total != 1 || p.Recipes[0].Category != "Beef" || p.Recipes[0].Cuisine != "Canadian" {
		t.Fatalf("Search = %+v, %v", p, err)
	}
}

func TestSearchFilterLabelFallsBackToTypedText(t *testing.T) {
	f := newRecipeFixture(t)
	f.client.CategoriesErr = errors.New("categories down")
	f.client.AreaList = []string{"Italian"}
	f.client.Filters["c=beef"] = []mealdb.MealRef{ref("1")}
	f.client.Filters["a=Martian"] = []mealdb.MealRef{ref("1")}
	p, err := f.svc.Search(context.Background(), RecipeQuery{Category: "beef", Cuisine: "Martian"})
	if err != nil || p.Total != 1 || p.Recipes[0].Category != "beef" || p.Recipes[0].Cuisine != "Martian" {
		t.Fatalf("a failed or non-matching label lookup must not fail the search: %+v, %v", p, err)
	}
}

// barrierClient holds every Search and Filter call until `want` of them are
// in flight at once, so a service that makes them one at a time fails.
type barrierClient struct {
	*testutil.MealDBClient
	want    int
	mu      sync.Mutex
	arrived int
	all     chan struct{}
}

func newBarrierClient(want int) *barrierClient {
	return &barrierClient{MealDBClient: testutil.NewMealDBClient(), want: want, all: make(chan struct{})}
}

func (b *barrierClient) wait(ctx context.Context) error {
	b.mu.Lock()
	b.arrived++
	if b.arrived == b.want {
		close(b.all)
	}
	b.mu.Unlock()
	select {
	case <-b.all:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(2 * time.Second):
		return errors.New("upstream calls were made one at a time")
	}
}

func (b *barrierClient) Search(ctx context.Context, query string) ([]mealdb.Meal, error) {
	if err := b.wait(ctx); err != nil {
		return nil, err
	}
	return b.MealDBClient.Search(ctx, query)
}

func (b *barrierClient) Filter(ctx context.Context, kind mealdb.FilterKind, value string) ([]mealdb.MealRef, error) {
	if err := b.wait(ctx); err != nil {
		return nil, err
	}
	return b.MealDBClient.Filter(ctx, kind, value)
}

func newServiceWith(client mealdb.Client) RecipeService {
	return NewRecipeService(client, testutil.NewCacheRepo(), RecipeCacheTTL{Detail: time.Hour, Search: time.Hour}, time.Now)
}

func TestSearchFetchesFiltersConcurrently(t *testing.T) {
	c := newBarrierClient(3)
	c.Filters["c=Vegetarian"] = []mealdb.MealRef{ref("1"), ref("2")}
	c.Filters["a=Italian"] = []mealdb.MealRef{ref("2")}
	c.Filters["i=garlic"] = []mealdb.MealRef{ref("2")}
	p, err := newServiceWith(c).Search(context.Background(), RecipeQuery{Category: "Vegetarian", Cuisine: "Italian", Ingredient: "garlic"})
	if err != nil || fmt.Sprint(ids(p)) != "[2]" {
		t.Fatalf("Search = %+v, %v", p, err)
	}
}

func TestSearchByNameFetchesIngredientFilterConcurrently(t *testing.T) {
	c := newBarrierClient(2)
	c.SearchResults["soup"] = []mealdb.Meal{{ID: "1", Name: "Soup"}, {ID: "2", Name: "Other soup"}}
	c.Filters["i=garlic"] = []mealdb.MealRef{ref("2")}
	p, err := newServiceWith(c).Search(context.Background(), RecipeQuery{Q: "soup", Ingredient: "garlic"})
	if err != nil || fmt.Sprint(ids(p)) != "[2]" {
		t.Fatalf("Search = %+v, %v", p, err)
	}
}

// failFastClient fails the cuisine filter at once and holds the others until
// their context is cancelled, recording how many were cancelled.
type failFastClient struct {
	*testutil.MealDBClient
	cancelled atomic.Int32
}

func (c *failFastClient) Filter(ctx context.Context, kind mealdb.FilterKind, _ string) ([]mealdb.MealRef, error) {
	if kind == mealdb.FilterArea {
		return nil, errors.New("area filter down")
	}
	select {
	case <-ctx.Done():
		c.cancelled.Add(1)
		return nil, ctx.Err()
	case <-time.After(2 * time.Second):
		return []mealdb.MealRef{}, nil
	}
}

func TestSearchFilterFailureCancelsTheOthers(t *testing.T) {
	c := &failFastClient{MealDBClient: testutil.NewMealDBClient()}
	start := time.Now()
	_, err := newServiceWith(c).Search(context.Background(), RecipeQuery{Category: "Beef", Cuisine: "Italian", Ingredient: "garlic"})
	if !errors.Is(err, ErrUpstreamUnavailable) {
		t.Fatalf("err = %v, want ErrUpstreamUnavailable", err)
	}
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Errorf("Search took %v; the other fetches should be cancelled, not awaited", elapsed)
	}
	if n := c.cancelled.Load(); n != 2 {
		t.Errorf("cancelled = %d, want 2", n)
	}
}
