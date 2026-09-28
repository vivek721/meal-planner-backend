package services

import (
	"context"
	"strings"
	"unicode/utf8"

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
// category/cuisine and, if requested, by the main-ingredient filter.
func (s *recipeService) searchByName(ctx context.Context, q RecipeQuery) ([]RecipeSummary, error) {
	key := "search:q=" + strings.ToLower(q.Q)
	meals, err := cached(ctx, s, key, s.ttl.Search, func(ctx context.Context) ([]mealdb.Meal, error) {
		return s.client.Search(ctx, q.Q)
	})
	if err != nil {
		return nil, err
	}
	var allowed map[string]bool
	if q.Ingredient != "" {
		refs, err := s.filter(ctx, mealdb.FilterIngredient, q.Ingredient)
		if err != nil {
			return nil, err
		}
		allowed = idSet(refs)
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
		if allowed != nil && !allowed[m.ID] {
			continue
		}
		out = append(out, RecipeSummary{ID: m.ID, Name: m.Name, Thumbnail: m.Thumbnail, Category: m.Category, Cuisine: m.Area})
	}
	return out, nil
}

// searchByFilters intersects each requested filter list, keeping the order of the first.
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

	var base []mealdb.MealRef
	var keep []map[string]bool
	for i, r := range reqs {
		refs, err := s.filter(ctx, r.kind, r.value)
		if err != nil {
			return nil, err
		}
		if i == 0 {
			base = refs
		} else {
			keep = append(keep, idSet(refs))
		}
	}

	out := make([]RecipeSummary, 0, len(base))
next:
	for _, ref := range base {
		for _, set := range keep {
			if !set[ref.ID] {
				continue next
			}
		}
		out = append(out, RecipeSummary{ID: ref.ID, Name: ref.Name, Thumbnail: ref.Thumbnail, Category: q.Category, Cuisine: q.Cuisine})
	}
	return out, nil
}

func (s *recipeService) filter(ctx context.Context, kind mealdb.FilterKind, value string) ([]mealdb.MealRef, error) {
	key := "filter:" + string(kind) + "=" + value
	return cached(ctx, s, key, s.ttl.Search, func(ctx context.Context) ([]mealdb.MealRef, error) {
		return s.client.Filter(ctx, kind, value)
	})
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
