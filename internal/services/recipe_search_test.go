package services

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/meal-planner/backend/internal/mealdb"
)

func ref(id string) mealdb.MealRef {
	return mealdb.MealRef{ID: id, Name: "Meal " + id, Thumbnail: "t" + id}
}

func ids(p *RecipePage) []string {
	out := []string{}
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
