package services

import (
	"context"
	"encoding/json"
	"math"
	"os"
	"testing"

	"github.com/meal-planner/backend/internal/mealdb"
	"github.com/meal-planner/backend/internal/usda"
)

// TestGoldenEstimate runs the whole pipeline over recorded fixtures and pins
// the resulting totals, so accidental changes to the maths are caught. The
// expectations were verified by hand from the fixture values (grams x
// per-100 g / 100, summed, rounded once); the 1% tolerance absorbs float
// noise without letting arithmetic drift through.
func TestGoldenEstimate(t *testing.T) {
	var meal mealdb.Meal
	mustLoad(t, "testdata/nutrition_golden_meal.json", &meal)
	var fx struct {
		Searches map[string][]usda.SearchFood `json:"searches"`
		Foods    []usda.Food                  `json:"foods"`
	}
	mustLoad(t, "testdata/nutrition_golden_foods.json", &fx)

	f := newNutritionFixture(t, `{"chicken breasts": {"itemGrams": 174}}`)
	f.recipes.client.Meals[meal.ID] = meal
	f.client.SearchResults = fx.Searches
	for _, food := range fx.Foods {
		f.client.FoodsByID[food.FDCID] = food
	}

	got, err := f.svc.Estimate(context.Background(), meal.ID)
	if err != nil {
		t.Fatalf("Estimate: %v", err)
	}

	within1pc := func(name string, got, want float64) {
		t.Helper()
		if want == 0 && got == 0 {
			return
		}
		if math.Abs(got-want) > 0.01*math.Abs(want) {
			t.Errorf("%s = %v, want about %v", name, got, want)
		}
	}
	within1pc("calories", float64(got.Totals.Calories), 1708)
	within1pc("protein", got.Totals.Protein, 110.4)
	within1pc("carbohydrate", got.Totals.Carbohydrate, 283.4)
	within1pc("fat", got.Totals.Fat, 15.9)
	within1pc("fiber", got.Totals.Fiber, 11.2)
	within1pc("sugars", got.Totals.Sugars, 124.6)
	within1pc("sodium", float64(got.Totals.Sodium), 10751)

	if got.Coverage.Counted != 8 || got.Coverage.Total != 10 {
		t.Errorf("coverage = %+v, want 8 of 10", got.Coverage)
	}
	if len(got.Incomplete) != 1 || got.Incomplete[0] != "sugars" {
		t.Errorf("incomplete = %v, want [sugars]", got.Incomplete)
	}
	byName := map[string]IngredientNutrition{}
	for _, ing := range got.Ingredients {
		byName[ing.Name] = ing
	}
	if l := byName["salt"]; l.Status != "notCounted" || l.Reason != "unmeasurable" {
		t.Errorf("salt = %+v", l)
	}
	if l := byName["stir-fry vegetables"]; l.Status != "notCounted" || l.Reason != "noMatch" {
		t.Errorf("stir-fry vegetables = %+v", l)
	}
}

func mustLoad(t *testing.T, path string, v any) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if err := json.Unmarshal(data, v); err != nil {
		t.Fatalf("decode %s: %v", path, err)
	}
}
