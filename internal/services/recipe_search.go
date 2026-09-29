package services

import (
	"context"
	"strings"
	"unicode/utf8"

	"golang.org/x/sync/errgroup"

	"github.com/meal-planner/backend/internal/mealdb"
)

func (s *recipeService) Search(ctx context.Context, q RecipeQuery) (*RecipePage, error) {
	q.Q = strings.TrimSpace(q.Q)
	q.Category = strings.TrimSpace(q.Category)
	q.Cuisine = strings.TrimSpace(q.Cuisine)
	q.Ingredient = strings.TrimSpace(q.Ingredient)
	if q.Q == "" && q.Category == "" && q.Cuisine == "" && q.Ingredient == "" {
		return nil, ErrInvalidSearch
	}
	for _, v := range [...]string{q.Q, q.Category, q.Cuisine, q.Ingredient} {
		if utf8.RuneCountInString(v) > maxSearchParamLength {
			return nil, ErrSearchTooLong
		}
	}

	var (
		results []RecipeSummary
		err     error
	)
	if q.Q != "" {
		results, err = s.searchByName(ctx, q)
	} else {
		results, err = s.searchByFilters(ctx, q)
	}
	if err != nil {
		return nil, err
	}
	return paginate(results, q.Page, q.Limit), nil
}

// searchByName runs TheMealDB's name search, then narrows the full meals by
// category/cuisine and, if requested, by the main-ingredient filter. The name
// search and the ingredient filter are fetched concurrently.
func (s *recipeService) searchByName(ctx context.Context, q RecipeQuery) ([]RecipeSummary, error) {
	var (
		meals   []mealdb.Meal
		allowed map[string]bool
	)
	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		var err error
		meals, err = cached(gctx, s, "search:q="+strings.ToLower(q.Q), s.ttl.Search, func(ctx context.Context) ([]mealdb.Meal, error) {
			return s.client.Search(ctx, q.Q)
		})
		return err
	})
	if q.Ingredient != "" {
		g.Go(func() error {
			refs, err := s.filter(gctx, mealdb.FilterIngredient, q.Ingredient)
			allowed = idSet(refs)
			return err
		})
	}
	if err := g.Wait(); err != nil {
		return nil, err //nolint:wrapcheck // already an ErrUpstreamUnavailable from cached
	}

	out := make([]RecipeSummary, 0, len(meals))
	for i := range meals {
		m := &meals[i]
		if q.Category != "" && !strings.EqualFold(m.Category, q.Category) {
			continue
		}
		if q.Cuisine != "" && !strings.EqualFold(m.Area, q.Cuisine) {
			continue
		}
		if q.Ingredient != "" && !allowed[m.ID] {
			continue
		}
		out = append(out, RecipeSummary{ID: m.ID, Name: m.Name, Thumbnail: m.Thumbnail, Category: m.Category, Cuisine: m.Area})
	}
	return out, nil
}

// searchByFilters fetches each requested filter list concurrently and
// intersects them, keeping the order of the first. Results are labeled with
// TheMealDB's own spelling of the category and cuisine ("Beef", not "beef").
func (s *recipeService) searchByFilters(ctx context.Context, q RecipeQuery) ([]RecipeSummary, error) {
	type filterReq struct {
		kind  mealdb.FilterKind
		value string
	}
	var reqs []filterReq
	for _, r := range []filterReq{
		{mealdb.FilterCategory, q.Category}, {mealdb.FilterArea, q.Cuisine}, {mealdb.FilterIngredient, q.Ingredient},
	} {
		if r.value != "" {
			reqs = append(reqs, r)
		}
	}

	lists := make([][]mealdb.MealRef, len(reqs))
	category, cuisine := q.Category, q.Cuisine
	g, gctx := errgroup.WithContext(ctx)
	for i, r := range reqs {
		g.Go(func() error {
			var err error
			lists[i], err = s.filter(gctx, r.kind, r.value)
			return err
		})
	}
	if category != "" {
		g.Go(func() error {
			category = s.canonicalCategory(gctx, category)
			return nil
		})
	}
	if cuisine != "" {
		g.Go(func() error {
			cuisine = s.canonicalCuisine(gctx, cuisine)
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return nil, err //nolint:wrapcheck // already an ErrUpstreamUnavailable from cached
	}

	keep := make([]map[string]bool, 0, len(lists)-1)
	for _, refs := range lists[1:] {
		keep = append(keep, idSet(refs))
	}
	out := make([]RecipeSummary, 0, len(lists[0]))
next:
	for _, ref := range lists[0] {
		for _, set := range keep {
			if !set[ref.ID] {
				continue next
			}
		}
		out = append(out, RecipeSummary{ID: ref.ID, Name: ref.Name, Thumbnail: ref.Thumbnail, Category: category, Cuisine: cuisine})
	}
	return out, nil
}

// filter returns one TheMealDB filter list. TheMealDB matches filter values
// ignoring case and treating "_" as a space, so the cache key does too:
// "Beef", "beef" and "BEEF" share one entry and one upstream call.
func (s *recipeService) filter(ctx context.Context, kind mealdb.FilterKind, value string) ([]mealdb.MealRef, error) {
	key := "filter:" + string(kind) + "=" + normalizeFilterValue(value)
	return cached(ctx, s, key, s.ttl.Search, func(ctx context.Context) ([]mealdb.MealRef, error) {
		return s.client.Filter(ctx, kind, value)
	})
}

// normalizeFilterValue lower-cases v, treats "_" as a space and collapses runs
// of spaces, matching how TheMealDB compares filter values.
func normalizeFilterValue(v string) string {
	return strings.Join(strings.Fields(strings.ReplaceAll(strings.ToLower(v), "_", " ")), " ")
}

// canonicalCategory returns TheMealDB's spelling of a category name, or name
// unchanged when the list is unavailable or has no match. The label is
// cosmetic, so a failed lookup never fails the search.
func (s *recipeService) canonicalCategory(ctx context.Context, name string) string {
	cats, err := s.Categories(ctx)
	if err != nil {
		return name
	}
	names := make([]string, len(cats))
	for i, c := range cats {
		names[i] = c.Name
	}
	return canonicalName(names, name)
}

// canonicalCuisine is canonicalCategory for cuisines (TheMealDB's "areas").
func (s *recipeService) canonicalCuisine(ctx context.Context, name string) string {
	areas, err := cached(ctx, s, "areas", s.ttl.Detail, s.client.Areas)
	if err != nil {
		return name
	}
	return canonicalName(areas, name)
}

func canonicalName(names []string, name string) string {
	want := normalizeFilterValue(name)
	for _, n := range names {
		if normalizeFilterValue(n) == want {
			return n
		}
	}
	return name
}

func idSet(refs []mealdb.MealRef) map[string]bool {
	set := make(map[string]bool, len(refs))
	for _, r := range refs {
		set[r.ID] = true
	}
	return set
}

func paginate(all []RecipeSummary, page, limit int) *RecipePage {
	if limit <= 0 {
		limit = defaultRecipeLimit
	}
	if limit > maxRecipeLimit {
		limit = maxRecipeLimit
	}
	if page < 1 {
		page = 1
	}
	total := len(all)
	totalPages := (total + limit - 1) / limit
	if page > totalPages {
		return &RecipePage{Recipes: []RecipeSummary{}, Total: total, Page: page, TotalPages: totalPages}
	}
	// page <= totalPages here, so (page-1)*limit < total: no overflow risk
	// even for pathological page values, since we never multiply those.
	start := (page - 1) * limit
	end := min(start+limit, total)
	recipes := make([]RecipeSummary, 0, end-start)
	recipes = append(recipes, all[start:end]...)
	return &RecipePage{Recipes: recipes, Total: total, Page: page, TotalPages: totalPages}
}
