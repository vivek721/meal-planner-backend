package mealdb

import (
	"bytes"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// rawMeal is one TheMealDB meal object; every value is a string or null.
type rawMeal map[string]*string

// stepMarker matches bare step headings like "STEP 1", "step 2:" or "Step 3."
var stepMarker = regexp.MustCompile(`(?i)^step\s*\d+\s*[:.]?$`)

func (r rawMeal) get(key string) string {
	if v := r[key]; v != nil {
		return strings.TrimSpace(*v)
	}
	return ""
}

// toMeal converts a raw meal into a Meal with non-nil slices.
func toMeal(r rawMeal) Meal {
	m := Meal{
		ID:           r.get("idMeal"),
		Name:         r.get("strMeal"),
		Category:     r.get("strCategory"),
		Area:         r.get("strArea"),
		Thumbnail:    r.get("strMealThumb"),
		Ingredients:  []Ingredient{},
		Instructions: splitSteps(r.get("strInstructions")),
		Tags:         splitTags(r.get("strTags")),
		YouTubeURL:   r.get("strYoutube"),
		SourceURL:    r.get("strSource"),
	}
	for i := 1; i <= 20; i++ {
		name := r.get("strIngredient" + strconv.Itoa(i))
		if name == "" {
			continue
		}
		m.Ingredients = append(m.Ingredients, Ingredient{Name: name, Measure: r.get("strMeasure" + strconv.Itoa(i))})
	}
	return m
}

func splitSteps(text string) []string {
	steps := []string{}
	for _, line := range strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || stepMarker.MatchString(line) {
			continue
		}
		steps = append(steps, line)
	}
	return steps
}

func splitTags(text string) []string {
	tags := []string{}
	for _, tag := range strings.Split(text, ",") {
		if tag = strings.TrimSpace(tag); tag != "" {
			tags = append(tags, tag)
		}
	}
	return tags
}

// decodeMeals reads {"meals": [...]}. TheMealDB uses null or a string such as
// "Invalid ID" for "no results", which becomes an empty list.
func decodeMeals(body []byte) ([]rawMeal, error) {
	var env struct {
		Meals json.RawMessage `json:"meals"`
	}
	if err := json.Unmarshal(body, &env); err != nil {
		return nil, fmt.Errorf("mealdb: decode response: %w", err)
	}
	trimmed := bytes.TrimSpace(env.Meals)
	if len(trimmed) == 0 || trimmed[0] != '[' {
		return []rawMeal{}, nil
	}
	var meals []rawMeal
	if err := json.Unmarshal(trimmed, &meals); err != nil {
		return nil, fmt.Errorf("mealdb: decode meals: %w", err)
	}
	return meals, nil
}
