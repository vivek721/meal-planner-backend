package mealdb

import (
	"reflect"
	"testing"
)

func s(v string) *string { return &v }

func TestToMeal(t *testing.T) {
	raw := rawMeal{
		"idMeal": s("52772"), "strMeal": s(" Teriyaki Chicken Casserole "), "strCategory": s("Chicken"),
		"strArea": s("Japanese"), "strMealThumb": s("https://img/x.jpg"),
		"strInstructions": s("STEP 1\r\nPreheat oven.\r\n\r\nstep 2:\nMix sauce.\n  Bake 35 min.  "),
		"strTags":         s("Meat, Casserole,,"), "strYoutube": s(""), "strSource": nil,
		"strIngredient1": s("soy sauce"), "strMeasure1": s("3/4 cup"),
		"strIngredient2": s("  "), "strMeasure2": s("1 tbsp"),
		"strIngredient3": s("garlic"), "strMeasure3": nil,
	}
	got := toMeal(raw)
	want := Meal{
		ID: "52772", Name: "Teriyaki Chicken Casserole", Category: "Chicken", Area: "Japanese",
		Thumbnail:    "https://img/x.jpg",
		Ingredients:  []Ingredient{{Name: "soy sauce", Measure: "3/4 cup"}, {Name: "garlic", Measure: ""}},
		Instructions: []string{"Preheat oven.", "Mix sauce.", "Bake 35 min."},
		Tags:         []string{"Meat", "Casserole"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("toMeal:\n got %+v\nwant %+v", got, want)
	}
}

func TestToMealEmptyListsAreNotNil(t *testing.T) {
	got := toMeal(rawMeal{"idMeal": s("1"), "strMeal": s("X")})
	if got.Ingredients == nil || got.Instructions == nil || got.Tags == nil {
		t.Errorf("empty slices must be non-nil so JSON renders []: %+v", got)
	}
}

func TestDecodeMealsHandlesNonArrays(t *testing.T) {
	for _, body := range []string{`{"meals":null}`, `{"meals":"Invalid ID"}`, `{}`} {
		meals, err := decodeMeals([]byte(body))
		if err != nil || len(meals) != 0 {
			t.Errorf("%s: got %v, %v; want empty, nil", body, meals, err)
		}
	}
	if _, err := decodeMeals([]byte(`{"meals":[`)); err == nil {
		t.Error("malformed JSON must return an error")
	}
}
