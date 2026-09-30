package handlers_test

import (
	"errors"
	"net/http"
	"testing"

	"github.com/meal-planner/backend/internal/mealdb"
	"github.com/meal-planner/backend/internal/usda"
)

func seedNutritionMeal(ts *testServer) {
	ts.mealdb.Meals["52772"] = mealdb.Meal{
		ID: "52772", Name: "Teriyaki",
		Ingredients:  []mealdb.Ingredient{{Name: "soy sauce", Measure: "1 tbs"}},
		Instructions: []string{}, Tags: []string{},
	}
	ts.usda.SearchResults["soy sauce"] = []usda.SearchFood{{FDCID: 174277, Description: "Soy sauce (shoyu)"}}
	ts.usda.FoodsByID[174277] = usda.Food{
		FDCID: 174277, Description: "Soy sauce (shoyu)",
		Per100g:  usda.Nutrients{usda.Calories: 53, usda.Protein: 8.14, usda.Carbohydrate: 4.93, usda.Fat: 0.57, usda.Fiber: 0.8, usda.Sugars: 0.4, usda.Sodium: 5493},
		Portions: []usda.Portion{{Amount: 1, Unit: "tbsp", GramWeight: 16}},
	}
}

func TestNutritionEndpoint(t *testing.T) {
	ts := newTestServer(t)
	_, token := ts.seedUser(t, "nutrition@example.com")
	seedNutritionMeal(ts)

	// 200 with the documented shape
	var body map[string]any
	if status := ts.getJSON(t, "/api/recipes/52772/nutrition", token, &body); status != http.StatusOK {
		t.Fatalf("status = %d, body %v", status, body)
	}
	if body["recipeId"] != "52772" || body["source"] != "USDA FoodData Central" {
		t.Errorf("body = %v", body)
	}
	if cov, ok := body["coverage"].(map[string]any); !ok || cov["counted"] != float64(1) || cov["total"] != float64(1) {
		t.Errorf("coverage = %v", body["coverage"])
	}

	// 400 non-numeric id
	if status := ts.getJSON(t, "/api/recipes/abc/nutrition", token, &body); status != http.StatusBadRequest {
		t.Errorf("bad id status = %d", status)
	}
	// 404 unknown meal
	if status := ts.getJSON(t, "/api/recipes/99999/nutrition", token, &body); status != http.StatusNotFound {
		t.Errorf("missing meal status = %d", status)
	}
	// 401 without a token
	if status := ts.getJSON(t, "/api/recipes/52772/nutrition", "", &body); status != http.StatusUnauthorized {
		t.Errorf("no-auth status = %d", status)
	}
}

func TestNutritionEndpoint503(t *testing.T) {
	ts := newTestServer(t)
	_, token := ts.seedUser(t, "nutrition503@example.com")
	ts.mealdb.Meals["52772"] = mealdb.Meal{
		ID: "52772", Name: "Teriyaki",
		Ingredients:  []mealdb.Ingredient{{Name: "soy sauce", Measure: "1 tbs"}},
		Instructions: []string{}, Tags: []string{},
	}
	ts.usda.Err = errors.New("fdc down")

	var body map[string]any
	if status := ts.getJSON(t, "/api/recipes/52772/nutrition", token, &body); status != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, body %v", status, body)
	}
	if body["error"] != "nutrition is temporarily unavailable, please try again shortly" {
		t.Errorf("error = %v", body["error"])
	}
}
