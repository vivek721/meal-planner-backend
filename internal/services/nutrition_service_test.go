package services

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/meal-planner/backend/internal/mealdb"
	"github.com/meal-planner/backend/internal/nutrition/overrides"
	"github.com/meal-planner/backend/internal/testutil"
	"github.com/meal-planner/backend/internal/usda"
)

type nutritionFixture struct {
	svc     NutritionService
	recipes *recipeFixture
	client  *testutil.USDAClient
	cache   *testutil.CacheRepo
	ov      *overrides.Set
	now     time.Time
}

func newNutritionFixture(t *testing.T, overridesJSON string) *nutritionFixture {
	t.Helper()
	ov, err := overrides.Parse([]byte(overridesJSON))
	if err != nil {
		t.Fatalf("overrides: %v", err)
	}
	f := &nutritionFixture{
		recipes: newRecipeFixture(t),
		client:  testutil.NewUSDAClient(),
		cache:   testutil.NewCacheRepo(),
		ov:      ov,
		now:     time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC),
	}
	f.svc = NewNutritionService(f.recipes.svc, f.client, f.cache,
		f.ov, NutritionTTL{Match: 720 * time.Hour, Food: 2160 * time.Hour, Result: 168 * time.Hour},
		func() time.Time { return f.now })
	return f
}

// seedMeal registers a meal with the recipe fixture's fake TheMealDB client.
func (f *nutritionFixture) seedMeal(id string, ingredients ...mealdb.Ingredient) {
	f.recipes.client.Meals[id] = mealdb.Meal{
		ID: id, Name: "Test Meal", Ingredients: ingredients,
		Instructions: []string{}, Tags: []string{},
	}
}

var soySauceFood = usda.Food{
	FDCID: 174277, Description: "Soy sauce made from soy and wheat (shoyu)",
	Per100g:  usda.Nutrients{usda.Calories: 53, usda.Protein: 8.14, usda.Carbohydrate: 4.93, usda.Fat: 0.57, usda.Fiber: 0.8, usda.Sugars: 0.4, usda.Sodium: 5493},
	Portions: []usda.Portion{{Amount: 1, Unit: "tbsp", GramWeight: 16}},
}

func (f *nutritionFixture) seedSoySauce() {
	f.client.SearchResults["soy sauce"] = []usda.SearchFood{{FDCID: 174277, Description: soySauceFood.Description, DataType: "SR Legacy"}}
	f.client.FoodsByID[174277] = soySauceFood
}

func TestEstimateCountsAndTotals(t *testing.T) {
	f := newNutritionFixture(t, `{}`)
	f.seedSoySauce()
	f.seedMeal("52772",
		mealdb.Ingredient{Name: "soy sauce", Measure: "3/4 cup"},
		mealdb.Ingredient{Name: "salt", Measure: "pinch"},
	)

	got, err := f.svc.Estimate(context.Background(), "52772")
	if err != nil {
		t.Fatalf("Estimate: %v", err)
	}
	if got.RecipeID != "52772" || got.Source != "USDA FoodData Central" {
		t.Errorf("header: %+v", got)
	}
	if !reflect.DeepEqual(got.Coverage, NutritionCoverage{Counted: 1, Total: 2}) {
		t.Errorf("coverage = %+v", got.Coverage)
	}
	// 3/4 cup = 177.441 ml; grams = 177.441 * 16 / 14.7868 = 192.0
	soy := got.Ingredients[0]
	if soy.Status != "counted" || soy.Grams < 191.9 || soy.Grams > 192.1 || soy.Food == nil || soy.Food.FDCID != 174277 {
		t.Errorf("soy line = %+v", soy)
	}
	// calories = 192.0/100*53 = 101.8 -> 102
	if soy.Calories != 102 {
		t.Errorf("soy calories = %d, want 102", soy.Calories)
	}
	salt := got.Ingredients[1]
	if salt.Status != "notCounted" || salt.Reason != "unmeasurable" || salt.Food != nil {
		t.Errorf("salt line = %+v", salt)
	}
	// sodium = 191.9994/100 * 5493 = 10546.5 -> 10547
	if got.Totals.Calories != 102 || got.Totals.Sodium != 10547 {
		t.Errorf("totals = %+v", got.Totals)
	}
	if len(got.Incomplete) != 0 {
		t.Errorf("incomplete = %v, want empty", got.Incomplete)
	}
	// An unmeasurable ingredient must never hit FDC.
	if f.client.SearchCalls != 1 {
		t.Errorf("SearchCalls = %d, want 1 (salt skipped)", f.client.SearchCalls)
	}
}

func TestEstimateReasons(t *testing.T) {
	f := newNutritionFixture(t, `{}`)
	// noMatch: search returns nothing. noPortion: food has no usable portion.
	f.client.SearchResults["dragon fruit"] = nil
	f.client.SearchResults["saffron"] = []usda.SearchFood{{FDCID: 5, Description: "Spices, saffron"}}
	f.client.FoodsByID[5] = usda.Food{FDCID: 5, Description: "Spices, saffron", Per100g: usda.Nutrients{usda.Calories: 310}}
	f.seedMeal("1",
		mealdb.Ingredient{Name: "dragon fruit", Measure: "2"},
		mealdb.Ingredient{Name: "saffron", Measure: "1 whole"},
	)
	got, err := f.svc.Estimate(context.Background(), "1")
	if err != nil {
		t.Fatalf("Estimate: %v", err)
	}
	if got.Ingredients[0].Reason != "noMatch" || got.Ingredients[1].Reason != "noPortion" {
		t.Errorf("reasons = %+v", got.Ingredients)
	}
	if got.Coverage.Counted != 0 {
		t.Errorf("coverage = %+v", got.Coverage)
	}
}

func TestEstimateOverridePinsFoodAndSkipsSearch(t *testing.T) {
	f := newNutritionFixture(t, `{"chicken breasts": {"fdcId": 171077, "itemGrams": 174}}`)
	f.client.FoodsByID[171077] = usda.Food{
		FDCID: 171077, Description: "Chicken, broilers or fryers, breast, meat only, raw",
		Per100g: usda.Nutrients{usda.Calories: 120, usda.Protein: 22.5, usda.Carbohydrate: 0, usda.Fat: 2.6, usda.Fiber: 0, usda.Sugars: 0, usda.Sodium: 45},
	}
	f.seedMeal("2", mealdb.Ingredient{Name: "Chicken Breasts", Measure: "2"})
	got, err := f.svc.Estimate(context.Background(), "2")
	if err != nil {
		t.Fatalf("Estimate: %v", err)
	}
	line := got.Ingredients[0]
	if line.Status != "counted" || line.Grams != 348 || line.Food.FDCID != 171077 {
		t.Errorf("line = %+v", line)
	}
	if f.client.SearchCalls != 0 {
		t.Errorf("SearchCalls = %d, want 0 (override pinned)", f.client.SearchCalls)
	}
	// 348/100*120 = 417.6 -> 418
	if got.Totals.Calories != 418 {
		t.Errorf("calories = %d", got.Totals.Calories)
	}
}

func TestEstimateAllNutrientsAbsent(t *testing.T) {
	f := newNutritionFixture(t, `{}`)
	f.client.SearchResults["mystery"] = []usda.SearchFood{{FDCID: 9, Description: "Mystery, raw"}}
	f.client.FoodsByID[9] = usda.Food{FDCID: 9, Description: "Mystery, raw", Per100g: usda.Nutrients{}}
	f.seedMeal("3", mealdb.Ingredient{Name: "mystery", Measure: "100g"})
	got, err := f.svc.Estimate(context.Background(), "3")
	if err != nil {
		t.Fatalf("Estimate: %v", err)
	}
	if got.Ingredients[0].Status != "counted" {
		t.Errorf("still counted by weight: %+v", got.Ingredients[0])
	}
	want := []string{"calories", "carbohydrate", "fat", "fiber", "protein", "sodium", "sugars"}
	if !reflect.DeepEqual(got.Incomplete, want) {
		t.Errorf("incomplete = %v, want all seven sorted", got.Incomplete)
	}
}

func TestEstimateZeroIngredients(t *testing.T) {
	f := newNutritionFixture(t, `{}`)
	f.seedMeal("4")
	got, err := f.svc.Estimate(context.Background(), "4")
	if err != nil {
		t.Fatalf("Estimate: %v", err)
	}
	if got.Coverage.Counted != 0 || got.Coverage.Total != 0 || len(got.Ingredients) != 0 {
		t.Errorf("zero-ingredient recipe: %+v", got)
	}
	if f.client.SearchCalls != 0 || f.client.FoodsCalls != 0 {
		t.Error("zero-ingredient recipe must make no FDC calls")
	}
}

func TestEstimateCountsDuplicateIngredientLines(t *testing.T) {
	f := newNutritionFixture(t, `{}`)
	f.seedSoySauce()
	f.seedMeal("5",
		mealdb.Ingredient{Name: "soy sauce", Measure: "1 tbs"},
		mealdb.Ingredient{Name: "Soy Sauce", Measure: "1 tbs"},
	)
	got, err := f.svc.Estimate(context.Background(), "5")
	if err != nil {
		t.Fatalf("Estimate: %v", err)
	}
	if got.Coverage.Counted != 2 {
		t.Errorf("both duplicate lines must count: %+v", got.Coverage)
	}
	if f.client.SearchCalls != 1 {
		t.Errorf("SearchCalls = %d, want 1 (deduped by normalised name)", f.client.SearchCalls)
	}
	// Each line: 16 g -> 16/100*53 = 8.48 -> 8 kcal shown per line; the
	// total is rounded once, from unrounded sums: 16.96 -> 17.
	if got.Totals.Calories != 17 {
		t.Errorf("calories = %d, want 17", got.Totals.Calories)
	}
}

func TestEstimateNotFoundPassesThrough(t *testing.T) {
	f := newNutritionFixture(t, `{}`)
	if _, err := f.svc.Estimate(context.Background(), "999"); !errors.Is(err, ErrRecipeNotFound) {
		t.Fatalf("err = %v, want ErrRecipeNotFound", err)
	}
}

func TestEstimateUpstreamFailure(t *testing.T) {
	f := newNutritionFixture(t, `{}`)
	f.seedMeal("6", mealdb.Ingredient{Name: "soy sauce", Measure: "1 tbs"})
	f.client.Err = errors.New("fdc down")
	if _, err := f.svc.Estimate(context.Background(), "6"); !errors.Is(err, ErrUpstreamUnavailable) {
		t.Fatalf("err = %v, want ErrUpstreamUnavailable", err)
	}
}
