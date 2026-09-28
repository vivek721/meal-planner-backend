package testutil

import (
	"context"

	"github.com/meal-planner/backend/internal/mealdb"
)

// MealDBClient is a fake mealdb.Client backed by maps.
type MealDBClient struct {
	Meals         map[string]mealdb.Meal
	SearchResults map[string][]mealdb.Meal
	// Filters is keyed by kind and value, e.g. "c=Seafood".
	Filters      map[string][]mealdb.MealRef
	CategoryList []mealdb.Category
	AreaList     []string
	// Err, when set, is returned by every method.
	Err error
	// Calls counts every method call.
	Calls int
}

// NewMealDBClient returns an empty fake client.
func NewMealDBClient() *MealDBClient {
	return &MealDBClient{
		Meals:         map[string]mealdb.Meal{},
		SearchResults: map[string][]mealdb.Meal{},
		Filters:       map[string][]mealdb.MealRef{},
	}
}

// Search returns SearchResults[query] (an empty list when absent).
func (c *MealDBClient) Search(_ context.Context, query string) ([]mealdb.Meal, error) {
	c.Calls++
	if c.Err != nil {
		return nil, c.Err
	}
	if m, ok := c.SearchResults[query]; ok {
		return m, nil
	}
	return []mealdb.Meal{}, nil
}

// Filter returns Filters[kind=value] (an empty list when absent).
func (c *MealDBClient) Filter(_ context.Context, kind mealdb.FilterKind, value string) ([]mealdb.MealRef, error) {
	c.Calls++
	if c.Err != nil {
		return nil, c.Err
	}
	if refs, ok := c.Filters[string(kind)+"="+value]; ok {
		return refs, nil
	}
	return []mealdb.MealRef{}, nil
}

// Lookup returns Meals[id], or mealdb.ErrNotFound.
func (c *MealDBClient) Lookup(_ context.Context, id string) (*mealdb.Meal, error) {
	c.Calls++
	if c.Err != nil {
		return nil, c.Err
	}
	m, ok := c.Meals[id]
	if !ok {
		return nil, mealdb.ErrNotFound
	}
	return &m, nil
}

// Categories returns CategoryList.
func (c *MealDBClient) Categories(_ context.Context) ([]mealdb.Category, error) {
	c.Calls++
	if c.Err != nil {
		return nil, c.Err
	}
	return c.CategoryList, nil
}

// Areas returns AreaList.
func (c *MealDBClient) Areas(_ context.Context) ([]string, error) {
	c.Calls++
	if c.Err != nil {
		return nil, c.Err
	}
	return c.AreaList, nil
}
