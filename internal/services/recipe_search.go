package services

import "context"

func (s *recipeService) Search(_ context.Context, _ RecipeQuery) (*RecipePage, error) {
	return nil, ErrInvalidSearch
}
