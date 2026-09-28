package mealdb

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// maxBodyBytes caps how much of a TheMealDB response is read.
const maxBodyBytes = 5 << 20

// HTTPClient implements Client over TheMealDB's JSON API.
type HTTPClient struct {
	baseURL string
	http    *http.Client
}

// NewHTTPClient returns a client for baseURL (e.g. https://www.themealdb.com/api/json/v1/1).
func NewHTTPClient(baseURL string, timeout time.Duration) *HTTPClient {
	return &HTTPClient{baseURL: strings.TrimRight(baseURL, "/"), http: &http.Client{Timeout: timeout}}
}

// get fetches path?params and returns the body of a 200 response.
func (c *HTTPClient) get(ctx context.Context, path string, params url.Values) ([]byte, error) {
	endpoint := c.baseURL + "/" + path + "?" + params.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("mealdb: build request: %w", err)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("mealdb: %s: %w", path, err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("mealdb: %s returned status %d", path, resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBodyBytes))
	if err != nil {
		return nil, fmt.Errorf("mealdb: read %s: %w", path, err)
	}
	return body, nil
}

func (c *HTTPClient) meals(ctx context.Context, path string, params url.Values) ([]rawMeal, error) {
	body, err := c.get(ctx, path, params)
	if err != nil {
		return nil, err
	}
	return decodeMeals(body)
}

// Search returns full meals whose name matches query.
func (c *HTTPClient) Search(ctx context.Context, query string) ([]Meal, error) {
	raw, err := c.meals(ctx, "search.php", url.Values{"s": {query}})
	if err != nil {
		return nil, err
	}
	meals := make([]Meal, 0, len(raw))
	for _, r := range raw {
		meals = append(meals, toMeal(r))
	}
	return meals, nil
}

// Filter returns partial meals matching one category, area or main ingredient.
func (c *HTTPClient) Filter(ctx context.Context, kind FilterKind, value string) ([]MealRef, error) {
	raw, err := c.meals(ctx, "filter.php", url.Values{string(kind): {value}})
	if err != nil {
		return nil, err
	}
	refs := make([]MealRef, 0, len(raw))
	for _, r := range raw {
		refs = append(refs, MealRef{ID: r.get("idMeal"), Name: r.get("strMeal"), Thumbnail: r.get("strMealThumb")})
	}
	return refs, nil
}

// Lookup returns the meal with the given id, or ErrNotFound.
func (c *HTTPClient) Lookup(ctx context.Context, id string) (*Meal, error) {
	raw, err := c.meals(ctx, "lookup.php", url.Values{"i": {id}})
	if err != nil {
		return nil, err
	}
	if len(raw) == 0 {
		return nil, ErrNotFound
	}
	meal := toMeal(raw[0])
	return &meal, nil
}

// Categories returns all meal categories.
func (c *HTTPClient) Categories(ctx context.Context) ([]Category, error) {
	body, err := c.get(ctx, "categories.php", url.Values{})
	if err != nil {
		return nil, err
	}
	var env struct {
		Categories []rawMeal `json:"categories"`
	}
	if err := json.Unmarshal(body, &env); err != nil {
		return nil, fmt.Errorf("mealdb: decode categories: %w", err)
	}
	cats := make([]Category, 0, len(env.Categories))
	for _, r := range env.Categories {
		cats = append(cats, Category{
			Name: r.get("strCategory"), Thumbnail: r.get("strCategoryThumb"), Description: r.get("strCategoryDescription"),
		})
	}
	return cats, nil
}

// Areas returns all cuisine names, as TheMealDB lists them.
func (c *HTTPClient) Areas(ctx context.Context) ([]string, error) {
	raw, err := c.meals(ctx, "list.php", url.Values{"a": {"list"}})
	if err != nil {
		return nil, err
	}
	areas := make([]string, 0, len(raw))
	for _, r := range raw {
		if a := r.get("strArea"); a != "" {
			areas = append(areas, a)
		}
	}
	return areas, nil
}
