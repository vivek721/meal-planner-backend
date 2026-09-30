package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	"github.com/meal-planner/backend/internal/mealdb"
	"github.com/meal-planner/backend/internal/models"
	"github.com/meal-planner/backend/internal/repository"
)

// Recipe service errors that handlers map to HTTP statuses.
var (
	ErrUpstreamUnavailable = errors.New("recipes are temporarily unavailable")
	ErrRecipeNotFound      = errors.New("recipe not found")
	ErrInvalidSearch       = errors.New("provide at least one of q, category, cuisine or ingredient")
	ErrSearchTooLong       = errors.New("search parameters must be at most 100 characters")
)

// Paging limits for Search.
const (
	defaultRecipeLimit = 24
	maxRecipeLimit     = 50
)

// maxSearchParamLength is the maximum rune length accepted for any of q,
// category, cuisine or ingredient, keeping cache keys bounded.
const maxSearchParamLength = 100

// RecipeCacheTTL sets how long cached TheMealDB results stay fresh.
type RecipeCacheTTL struct {
	Detail time.Duration // lookups, categories, cuisines
	Search time.Duration // name searches and filters
}

// RecipeQuery selects recipes; at least one of Q, Category, Cuisine or Ingredient is required.
type RecipeQuery struct {
	Q, Category, Cuisine, Ingredient string
	Page, Limit                      int
}

// RecipeSummary is a recipe in a result list.
type RecipeSummary struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Thumbnail string `json:"thumbnail"`
	Category  string `json:"category,omitempty"`
	Cuisine   string `json:"cuisine,omitempty"`
}

// RecipePage is one page of search results.
type RecipePage struct {
	Recipes    []RecipeSummary `json:"recipes"`
	Total      int             `json:"total"`
	Page       int             `json:"page"`
	TotalPages int             `json:"totalPages"`
}

// RecipeService serves TheMealDB recipes through a read-through cache.
type RecipeService interface {
	Categories(ctx context.Context) ([]mealdb.Category, error)
	Cuisines(ctx context.Context) ([]string, error)
	Get(ctx context.Context, id string) (*mealdb.Meal, error)
	Search(ctx context.Context, q RecipeQuery) (*RecipePage, error)
}

// readCache is a read-through cache over the mealdb_cache table, shared by
// the recipe and nutrition services. name prefixes its log lines.
type readCache struct {
	name string
	repo repository.CacheRepository
	now  func() time.Time
}

func newReadCache(name string, repo repository.CacheRepository, now func() time.Time) *readCache {
	return &readCache{name: name, repo: repo, now: now}
}

type recipeService struct {
	client mealdb.Client
	rc     *readCache
	ttl    RecipeCacheTTL
}

// NewRecipeService creates a RecipeService. now is injectable for tests; pass time.Now.
func NewRecipeService(client mealdb.Client, cache repository.CacheRepository, ttl RecipeCacheTTL, now func() time.Time) RecipeService {
	return &recipeService{client: client, rc: newReadCache("recipe cache", cache, now), ttl: ttl}
}

// passThrough reports whether err must reach the caller unchanged: not
// wrapped as unavailable, and never answered with a stale entry.
func passThrough(err error) bool {
	return errors.Is(err, mealdb.ErrNotFound) || errors.Is(err, ErrRecipeNotFound)
}

// cached returns the value for key: a fresh cache entry if there is one,
// otherwise the upstream result (which is then stored). If the upstream fails,
// an expired entry is served instead; with no entry at all the error is
// ErrUpstreamUnavailable. Pass-through errors are returned unchanged.
func cached[T any](ctx context.Context, c *readCache, key string, ttl time.Duration, fetch func(context.Context) (T, error)) (T, error) {
	var zero T
	entry, err := c.repo.Get(key)
	if err != nil {
		log.Printf("%s: read %q: %v", c.name, key, err)
		entry = nil
	}
	var stale *T
	if entry != nil {
		var v T
		if jsonErr := json.Unmarshal([]byte(entry.Payload), &v); jsonErr == nil {
			if c.now().Before(entry.ExpiresAt) {
				return v, nil
			}
			stale = &v
		} else {
			log.Printf("%s: decode %q: %v", c.name, key, jsonErr)
		}
	}

	v, err := fetch(ctx)
	if err != nil {
		if passThrough(err) {
			return zero, err
		}
		if stale != nil {
			log.Printf("%s: serving stale %q after upstream error: %v", c.name, key, err)
			return *stale, nil
		}
		return zero, fmt.Errorf("%w: %w", ErrUpstreamUnavailable, err)
	}

	if payload, mErr := json.Marshal(v); mErr == nil {
		now := c.now()
		if upErr := c.repo.Upsert(&models.CachedResponse{
			Key: key, Payload: string(payload),
			FetchedAt: now, ExpiresAt: now.Add(ttl),
		}); upErr != nil {
			log.Printf("%s: write %q: %v", c.name, key, upErr)
		}
	}
	return v, nil
}

func (s *recipeService) Categories(ctx context.Context) ([]mealdb.Category, error) {
	return cached(ctx, s.rc, "categories", s.ttl.Detail, s.client.Categories)
}

func (s *recipeService) Cuisines(ctx context.Context) ([]string, error) {
	areas, err := cached(ctx, s.rc, "areas", s.ttl.Detail, s.client.Areas)
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(areas))
	for _, a := range areas {
		if a = strings.TrimSpace(a); a != "" && !strings.EqualFold(a, "Unknown") {
			out = append(out, a)
		}
	}
	sort.Strings(out)
	return out, nil
}

func (s *recipeService) Get(ctx context.Context, id string) (*mealdb.Meal, error) {
	meal, err := cached(ctx, s.rc, "lookup:"+id, s.ttl.Detail, func(ctx context.Context) (*mealdb.Meal, error) {
		return s.client.Lookup(ctx, id)
	})
	if errors.Is(err, mealdb.ErrNotFound) {
		return nil, ErrRecipeNotFound
	}
	if err != nil {
		return nil, err
	}
	return meal, nil
}

// Search is implemented in recipe_search.go.
