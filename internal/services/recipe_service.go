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

type recipeService struct {
	client mealdb.Client
	cache  repository.CacheRepository
	ttl    RecipeCacheTTL
	now    func() time.Time
}

// NewRecipeService creates a RecipeService. now is injectable for tests; pass time.Now.
func NewRecipeService(client mealdb.Client, cache repository.CacheRepository, ttl RecipeCacheTTL, now func() time.Time) RecipeService {
	return &recipeService{client: client, cache: cache, ttl: ttl, now: now}
}

// cached returns the value for key: a fresh cache entry if there is one,
// otherwise the upstream result (which is then stored). If the upstream fails,
// an expired entry is served instead; with no entry at all the error is
// ErrUpstreamUnavailable. mealdb.ErrNotFound is passed through unchanged.
func cached[T any](ctx context.Context, s *recipeService, key string, ttl time.Duration, fetch func(context.Context) (T, error)) (T, error) {
	var zero T
	entry, err := s.cache.Get(key)
	if err != nil {
		log.Printf("recipe cache: read %q: %v", key, err)
		entry = nil
	}
	var stale *T
	if entry != nil {
		var v T
		if jsonErr := json.Unmarshal([]byte(entry.Payload), &v); jsonErr == nil {
			if s.now().Before(entry.ExpiresAt) {
				return v, nil
			}
			stale = &v
		} else {
			log.Printf("recipe cache: decode %q: %v", key, jsonErr)
		}
	}

	v, err := fetch(ctx)
	if err != nil {
		if errors.Is(err, mealdb.ErrNotFound) {
			return zero, err
		}
		if stale != nil {
			log.Printf("recipe cache: serving stale %q after upstream error: %v", key, err)
			return *stale, nil
		}
		return zero, fmt.Errorf("%w: %w", ErrUpstreamUnavailable, err)
	}

	if payload, mErr := json.Marshal(v); mErr == nil {
		now := s.now()
		if upErr := s.cache.Upsert(&models.CachedResponse{
			Key: key, Payload: string(payload),
			FetchedAt: now, ExpiresAt: now.Add(ttl),
		}); upErr != nil {
			log.Printf("recipe cache: write %q: %v", key, upErr)
		}
	}
	return v, nil
}

func (s *recipeService) Categories(ctx context.Context) ([]mealdb.Category, error) {
	return cached(ctx, s, "categories", s.ttl.Detail, s.client.Categories)
}

func (s *recipeService) Cuisines(ctx context.Context) ([]string, error) {
	areas, err := cached(ctx, s, "areas", s.ttl.Detail, s.client.Areas)
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
	meal, err := cached(ctx, s, "lookup:"+id, s.ttl.Detail, func(ctx context.Context) (*mealdb.Meal, error) {
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
