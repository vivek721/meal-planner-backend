package handlers_test

import (
	"errors"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/meal-planner/backend/internal/mealdb"
)

func TestRecipesRequireAuth(t *testing.T) {
	s := newTestServer(t)
	for _, path := range []string{"/api/recipes?q=x", "/api/recipes/52772", "/api/recipes/categories", "/api/recipes/cuisines"} {
		if code, _ := s.do(t, http.MethodGet, path, nil, ""); code != http.StatusUnauthorized {
			t.Errorf("%s without token: %d, want 401", path, code)
		}
	}
}

func TestGetRecipe(t *testing.T) {
	s := newTestServer(t)
	_, token := s.seedUser(t, "r@example.com")
	s.mealdb.Meals["52772"] = mealdb.Meal{
		ID: "52772", Name: "Teriyaki", Area: "Japanese",
		Ingredients: []mealdb.Ingredient{{Name: "soy sauce", Measure: "1 cup"}}, Instructions: []string{"Cook."},
		Tags: []string{}, YouTubeURL: "https://youtube.test/x",
	}

	var got map[string]any
	if code := s.getJSON(t, "/api/recipes/52772", token, &got); code != http.StatusOK {
		t.Fatalf("status %d", code)
	}
	if got["cuisine"] != "Japanese" || got["youtubeUrl"] != "https://youtube.test/x" || got["sourceUrl"] != nil {
		t.Errorf("unexpected body %v", got)
	}
	ing := got["ingredients"].([]any)[0].(map[string]any)
	if ing["name"] != "soy sauce" || ing["measure"] != "1 cup" {
		t.Errorf("ingredient %v", ing)
	}
}

func TestGetRecipeErrors(t *testing.T) {
	s := newTestServer(t)
	_, token := s.seedUser(t, "e@example.com")

	for _, id := range []string{"abc", "52772x", "-1"} {
		if code, resp := s.do(t, http.MethodGet, "/api/recipes/"+id, nil, token); code != http.StatusBadRequest || resp["error"] == nil {
			t.Errorf("id %q: %d %v, want 400 with error", id, code, resp)
		}
	}
	if s.mealdb.Calls != 0 {
		t.Errorf("invalid ids must not reach upstream (calls=%d)", s.mealdb.Calls)
	}
	if code, _ := s.do(t, http.MethodGet, "/api/recipes/99999", nil, token); code != http.StatusNotFound {
		t.Errorf("unknown id: %d, want 404", code)
	}
	s.mealdb.Err = errors.New("down")
	if code, resp := s.do(t, http.MethodGet, "/api/recipes/11111", nil, token); code != http.StatusServiceUnavailable || resp["error"] == nil {
		t.Errorf("upstream down: %d %v, want 503", code, resp)
	}
}

func TestSearchRecipes(t *testing.T) {
	s := newTestServer(t)
	_, token := s.seedUser(t, "s@example.com")
	s.mealdb.Filters["c=Seafood"] = []mealdb.MealRef{{ID: "1", Name: "Fish pie", Thumbnail: "t1"}}

	var page struct {
		Recipes []map[string]any `json:"recipes"`
		Total   int              `json:"total"`
		Page    int              `json:"page"`
		Pages   int              `json:"totalPages"`
	}
	if code := s.getJSON(t, "/api/recipes?category=Seafood&page=1&limit=10", token, &page); code != http.StatusOK {
		t.Fatalf("status %d", code)
	}
	if page.Total != 1 || page.Page != 1 || page.Pages != 1 || page.Recipes[0]["category"] != "Seafood" {
		t.Errorf("page %+v", page)
	}

	for _, path := range []string{"/api/recipes", "/api/recipes?category=Seafood&page=0", "/api/recipes?category=Seafood&limit=abc"} {
		if code, _ := s.do(t, http.MethodGet, path, nil, token); code != http.StatusBadRequest {
			t.Errorf("%s: %d, want 400", path, code)
		}
	}
}

func TestSearchRecipesRejectsOverlongParams(t *testing.T) {
	s := newTestServer(t)
	_, token := s.seedUser(t, "long@example.com")
	long := strings.Repeat("a", 101)

	path := "/api/recipes?q=" + url.QueryEscape(long)
	code, resp := s.do(t, http.MethodGet, path, nil, token)
	if code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", code)
	}
	if resp["error"] != "search parameters must be at most 100 characters" {
		t.Errorf("error = %v", resp["error"])
	}
	if s.mealdb.Calls != 0 {
		t.Errorf("overlong search must not reach upstream (calls=%d)", s.mealdb.Calls)
	}
}

func TestCategoriesAndCuisines(t *testing.T) {
	s := newTestServer(t)
	_, token := s.seedUser(t, "c@example.com")
	s.mealdb.CategoryList = []mealdb.Category{{Name: "Beef", Thumbnail: "b.png", Description: "Beef."}}
	s.mealdb.AreaList = []string{"Italian", "Unknown", "American"}

	var cats []map[string]any
	if code := s.getJSON(t, "/api/recipes/categories", token, &cats); code != http.StatusOK || cats[0]["name"] != "Beef" {
		t.Errorf("categories %d %v", code, cats)
	}
	var cuisines []string
	if code := s.getJSON(t, "/api/recipes/cuisines", token, &cuisines); code != http.StatusOK || len(cuisines) != 2 || cuisines[0] != "American" {
		t.Errorf("cuisines %d %v", code, cuisines)
	}
}
