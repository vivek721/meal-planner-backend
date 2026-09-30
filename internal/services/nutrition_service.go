package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"sort"
	"strconv"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/meal-planner/backend/internal/mealdb"
	"github.com/meal-planner/backend/internal/models"
	"github.com/meal-planner/backend/internal/nutrition/measure"
	"github.com/meal-planner/backend/internal/nutrition/overrides"
	"github.com/meal-planner/backend/internal/repository"
	"github.com/meal-planner/backend/internal/usda"
)

// nutritionSource labels every estimate with its data source.
const nutritionSource = "USDA FoodData Central"

// Ingredient statuses and not-counted reasons in the API contract.
const (
	statusCounted      = "counted"
	statusNotCounted   = "notCounted"
	reasonUnmeasurable = "unmeasurable"
	reasonNoMatch      = "noMatch"
	reasonNoPortion    = "noPortion"
)

// NutritionTTL sets how long each cached layer stays fresh.
type NutritionTTL struct {
	Match  time.Duration // ingredient-name -> FDC id decisions
	Food   time.Duration // per-food nutrients and portions
	Result time.Duration // finished per-recipe estimates
}

// NutritionTotals sums every counted ingredient. Calories are kcal and
// Sodium mg (integers); the rest are grams to one decimal place.
type NutritionTotals struct {
	Calories     int     `json:"calories"`
	Protein      float64 `json:"protein"`
	Carbohydrate float64 `json:"carbohydrate"`
	Fat          float64 `json:"fat"`
	Fiber        float64 `json:"fiber"`
	Sugars       float64 `json:"sugars"`
	Sodium       int     `json:"sodium"`
}

// NutritionFoodRef points a counted ingredient at its FDC food.
type NutritionFoodRef struct {
	FDCID       int    `json:"fdcId"`
	Description string `json:"description"`
}

// IngredientNutrition traces one ingredient line: counted with its grams,
// food and calories, or notCounted with a reason.
type IngredientNutrition struct {
	Name     string            `json:"name"`
	Measure  string            `json:"measure"`
	Status   string            `json:"status"`
	Grams    float64           `json:"grams,omitempty"`
	Food     *NutritionFoodRef `json:"food,omitempty"`
	Calories int               `json:"calories,omitempty"`
	Reason   string            `json:"reason,omitempty"`
}

// NutritionCoverage says how many ingredient lines were counted.
type NutritionCoverage struct {
	Counted int `json:"counted"`
	Total   int `json:"total"`
}

// RecipeNutrition is the whole-recipe estimate returned by the API.
type RecipeNutrition struct {
	RecipeID    string                `json:"recipeId"`
	Source      string                `json:"source"`
	Totals      NutritionTotals       `json:"totals"`
	Incomplete  []string              `json:"incomplete,omitempty"`
	Coverage    NutritionCoverage     `json:"coverage"`
	Ingredients []IngredientNutrition `json:"ingredients"`
}

// NutritionService estimates a recipe's nutrition from USDA data.
type NutritionService interface {
	Estimate(ctx context.Context, mealID string) (*RecipeNutrition, error)
}

type nutritionService struct {
	recipes RecipeService
	client  usda.Client
	rc      *readCache
	ov      *overrides.Set
	ttl     NutritionTTL
}

// NewNutritionService creates a NutritionService sharing the mealdb_cache
// table. now is injectable for tests; pass time.Now.
func NewNutritionService(recipes RecipeService, client usda.Client, cache repository.CacheRepository, ov *overrides.Set, ttl NutritionTTL, now func() time.Time) NutritionService {
	return &nutritionService{
		recipes: recipes, client: client,
		rc: newReadCache("nutrition cache", cache, now), ov: ov, ttl: ttl,
	}
}

func (s *nutritionService) Estimate(ctx context.Context, mealID string) (*RecipeNutrition, error) {
	key := "nutrition:" + s.ov.Version() + ":" + mealID
	v, err := cached(ctx, s.rc, key, s.ttl.Result, func(ctx context.Context) (RecipeNutrition, error) {
		out, err := s.estimate(ctx, mealID)
		if err != nil {
			return RecipeNutrition{}, err
		}
		return *out, nil
	})
	if err != nil {
		return nil, err
	}
	return &v, nil
}

// estimate builds a fresh estimate: load the meal, match measurable
// ingredients to FDC foods, fetch those foods, then assemble the totals.
func (s *nutritionService) estimate(ctx context.Context, mealID string) (*RecipeNutrition, error) {
	meal, err := s.recipes.Get(ctx, mealID)
	if err != nil {
		return nil, err
	}
	matches, err := s.matchAll(ctx, meal.Ingredients)
	if err != nil {
		return nil, err
	}
	foods, err := s.fetchFoods(ctx, matchedIDs(matches))
	if err != nil {
		return nil, err
	}
	out := assemble(meal, matches, foods, s.ov)
	return &out, nil
}

// maxConcurrentSearches caps in-flight FDC search calls per estimate.
const maxConcurrentSearches = 4

// matchResult is a cached matching decision; FDCID 0 records "no match" so
// hopeless ingredients do not re-hit FDC for every recipe that uses them.
type matchResult struct {
	FDCID int `json:"fdcId"`
}

// matchAll resolves each distinct measurable ingredient name to an FDC id
// (0 = no match). Overrides with a pinned fdcId skip the search entirely;
// searched decisions are cached under the matcher version.
func (s *nutritionService) matchAll(ctx context.Context, ingredients []mealdb.Ingredient) (map[string]int, error) {
	names := distinctMeasurableNames(ingredients)
	results := make([]int, len(names))
	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(maxConcurrentSearches)
	for i, name := range names {
		g.Go(func() error {
			if o, ok := s.ov.Get(name); ok && o.FDCID > 0 {
				results[i] = o.FDCID
				return nil
			}
			key := "usda:match:" + s.ov.Version() + ":" + name
			m, err := cached(gctx, s.rc, key, s.ttl.Match, func(ctx context.Context) (matchResult, error) {
				found, err := s.client.Search(ctx, name)
				if err != nil {
					return matchResult{}, err
				}
				return matchResult{FDCID: pickMatch(name, found)}, nil
			})
			if err != nil {
				return err //nolint:wrapcheck // already an ErrUpstreamUnavailable from cached
			}
			results[i] = m.FDCID
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return nil, err //nolint:wrapcheck // already wrapped
	}
	out := make(map[string]int, len(names))
	for i, name := range names {
		out[name] = results[i]
	}
	return out, nil
}

// distinctMeasurableNames returns the normalised names of ingredients whose
// measures parse to something countable, first occurrence order, deduped.
func distinctMeasurableNames(ingredients []mealdb.Ingredient) []string {
	seen := map[string]bool{}
	var names []string
	for _, ing := range ingredients {
		if measure.Parse(ing.Measure).Kind == measure.Unmeasurable {
			continue
		}
		n := overrides.Normalize(ing.Name)
		if n == "" || seen[n] {
			continue
		}
		seen[n] = true
		names = append(names, n)
	}
	return names
}

func matchedIDs(matches map[string]int) []int {
	seen := map[int]bool{}
	var ids []int
	for _, id := range matches {
		if id > 0 && !seen[id] {
			seen[id] = true
			ids = append(ids, id)
		}
	}
	sort.Ints(ids) // deterministic batches and cache order
	return ids
}

func foodKey(id int) string { return "usda:food:" + strconv.Itoa(id) }

// fetchFoods returns the foods for ids: fresh cache entries first, then one
// batched Foods call for the rest. On an upstream error, expired entries
// stand in where they exist; an id with nothing at all fails the estimate.
// An id FDC no longer returns is simply absent from the result, and the
// ingredient is reported noMatch.
func (s *nutritionService) fetchFoods(ctx context.Context, ids []int) (map[int]usda.Food, error) {
	out := make(map[int]usda.Food, len(ids))
	stale := map[int]usda.Food{}
	var missing []int
	for _, id := range ids {
		entry, err := s.rc.repo.Get(foodKey(id))
		if err != nil {
			log.Printf("nutrition cache: read %q: %v", foodKey(id), err)
			entry = nil
		}
		if entry != nil {
			var f usda.Food
			if jsonErr := json.Unmarshal([]byte(entry.Payload), &f); jsonErr == nil {
				if s.rc.now().Before(entry.ExpiresAt) {
					out[id] = f
					continue
				}
				stale[id] = f
			}
		}
		missing = append(missing, id)
	}
	if len(missing) == 0 {
		return out, nil
	}

	fetched, err := s.client.Foods(ctx, missing)
	if err != nil {
		for _, id := range missing {
			f, ok := stale[id]
			if !ok {
				return nil, wrapUnavailable(err)
			}
			out[id] = f
		}
		log.Printf("nutrition cache: serving stale foods after upstream error: %v", err)
		return out, nil
	}
	now := s.rc.now()
	for _, f := range fetched {
		out[f.FDCID] = f
		if payload, mErr := json.Marshal(f); mErr == nil {
			if upErr := s.rc.repo.Upsert(&models.CachedResponse{
				Key: foodKey(f.FDCID), Payload: string(payload),
				FetchedAt: now, ExpiresAt: now.Add(s.ttl.Food),
			}); upErr != nil {
				log.Printf("nutrition cache: write %q: %v", foodKey(f.FDCID), upErr)
			}
		}
	}
	return out, nil
}

// wrapUnavailable marks an FDC failure the way handlers expect.
func wrapUnavailable(err error) error {
	return fmt.Errorf("%w: %w", ErrUpstreamUnavailable, err)
}

// assemble walks the ingredient lines in recipe order, converting and
// summing every counted one. A nutrient some counted ingredient lacks makes
// the total incomplete, never zero-padded. Totals are rounded once, here.
func assemble(meal *mealdb.Meal, matches map[string]int, foods map[int]usda.Food, ov *overrides.Set) RecipeNutrition {
	res := RecipeNutrition{
		RecipeID: meal.ID, Source: nutritionSource,
		Coverage:    NutritionCoverage{Total: len(meal.Ingredients)},
		Ingredients: make([]IngredientNutrition, 0, len(meal.Ingredients)),
	}
	sums := usda.Nutrients{}
	incomplete := map[usda.Nutrient]bool{}

	for _, ing := range meal.Ingredients {
		line := IngredientNutrition{Name: ing.Name, Measure: ing.Measure, Status: statusNotCounted}
		norm := overrides.Normalize(ing.Name)
		amt := measure.Parse(ing.Measure)
		o, _ := ov.Get(norm)

		switch food, ok := foods[matches[norm]]; {
		case amt.Kind == measure.Unmeasurable:
			line.Reason = reasonUnmeasurable
		case !ok:
			line.Reason = reasonNoMatch
		default:
			grams, convertible := gramsFor(amt, norm, o, food.Portions)
			if !convertible {
				line.Reason = reasonNoPortion
				break
			}
			line.Status = statusCounted
			line.Grams = round1(grams)
			line.Food = &NutritionFoodRef{FDCID: food.FDCID, Description: food.Description}
			if kcal, has := food.Per100g[usda.Calories]; has {
				line.Calories = int(math.Round(grams / 100 * kcal))
			}
			res.Coverage.Counted++
			for _, n := range usda.All {
				v, has := food.Per100g[n]
				if !has {
					incomplete[n] = true
					continue
				}
				sums[n] += grams / 100 * v
			}
		}
		res.Ingredients = append(res.Ingredients, line)
	}

	res.Totals = NutritionTotals{
		Calories:     int(math.Round(sums[usda.Calories])),
		Protein:      round1(sums[usda.Protein]),
		Carbohydrate: round1(sums[usda.Carbohydrate]),
		Fat:          round1(sums[usda.Fat]),
		Fiber:        round1(sums[usda.Fiber]),
		Sugars:       round1(sums[usda.Sugars]),
		Sodium:       int(math.Round(sums[usda.Sodium])),
	}
	for n := range incomplete {
		res.Incomplete = append(res.Incomplete, string(n))
	}
	sort.Strings(res.Incomplete)
	return res
}
