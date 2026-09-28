// Package mealdb is a small client for TheMealDB (https://www.themealdb.com)
// that turns its responses into clean Go types.
package mealdb

import (
	"context"
	"errors"
)

// ErrNotFound is returned by Lookup when TheMealDB has no meal with that id.
var ErrNotFound = errors.New("mealdb: meal not found")

// Ingredient is one ingredient line; Measure is TheMealDB's free text (e.g. "3/4 cup").
type Ingredient struct {
	Name    string `json:"name"`
	Measure string `json:"measure"`
}

// Meal is a full recipe. Area is exposed to API clients as "cuisine".
type Meal struct {
	ID           string       `json:"id"`
	Name         string       `json:"name"`
	Category     string       `json:"category"`
	Area         string       `json:"cuisine"`
	Thumbnail    string       `json:"thumbnail"`
	Ingredients  []Ingredient `json:"ingredients"`
	Instructions []string     `json:"instructions"`
	Tags         []string     `json:"tags"`
	YouTubeURL   string       `json:"youtubeUrl,omitempty"`
	SourceURL    string       `json:"sourceUrl,omitempty"`
}

// MealRef is the partial meal returned by TheMealDB's filter endpoint.
type MealRef struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Thumbnail string `json:"thumbnail"`
}

// Category is a meal category with its image and description.
type Category struct {
	Name        string `json:"name"`
	Thumbnail   string `json:"thumbnail"`
	Description string `json:"description"`
}

// FilterKind selects which TheMealDB filter to apply.
type FilterKind string

// Filter kinds, matching TheMealDB's filter.php query parameters.
const (
	FilterCategory   FilterKind = "c"
	FilterArea       FilterKind = "a"
	FilterIngredient FilterKind = "i"
)

// Client fetches recipes from TheMealDB.
type Client interface {
	Search(ctx context.Context, query string) ([]Meal, error)
	Filter(ctx context.Context, kind FilterKind, value string) ([]MealRef, error)
	Lookup(ctx context.Context, id string) (*Meal, error)
	Categories(ctx context.Context) ([]Category, error)
	Areas(ctx context.Context) ([]string, error)
}
