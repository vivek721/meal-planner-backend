package mealdb

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// fakeMealDB serves canned bodies keyed by "path?query".
func fakeMealDB(t *testing.T, routes map[string]string) *HTTPClient {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, ok := routes[r.URL.Path+"?"+r.URL.RawQuery]
		if !ok {
			http.Error(w, "unexpected "+r.URL.String(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return NewHTTPClient(srv.URL, time.Second)
}

const oneMeal = `{"meals":[{"idMeal":"52772","strMeal":"Teriyaki","strCategory":"Chicken","strArea":"Japanese",
"strMealThumb":"https://img/x.jpg","strInstructions":"Cook.","strTags":null,"strIngredient1":"soy sauce","strMeasure1":"1 cup"}]}`

func TestSearchAndLookup(t *testing.T) {
	c := fakeMealDB(t, map[string]string{
		"/search.php?s=teri+yaki": oneMeal,
		"/lookup.php?i=52772":     oneMeal,
		"/lookup.php?i=1":         `{"meals":null}`,
	})
	ctx := context.Background()

	meals, err := c.Search(ctx, "teri yaki")
	if err != nil || len(meals) != 1 || meals[0].Area != "Japanese" {
		t.Fatalf("Search: %+v, %v", meals, err)
	}
	meal, err := c.Lookup(ctx, "52772")
	if err != nil || meal.Name != "Teriyaki" || meal.Ingredients[0].Measure != "1 cup" {
		t.Fatalf("Lookup: %+v, %v", meal, err)
	}
	if _, err := c.Lookup(ctx, "1"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Lookup missing: err = %v, want ErrNotFound", err)
	}
}

func TestFilterCategoriesAreas(t *testing.T) {
	c := fakeMealDB(t, map[string]string{
		"/filter.php?c=Seafood": `{"meals":[{"strMeal":"Fish pie","strMealThumb":"https://img/f.jpg","idMeal":"52802"}]}`,
		"/filter.php?i=nothing": `{"meals":null}`,
		"/categories.php?":      `{"categories":[{"idCategory":"1","strCategory":"Beef","strCategoryThumb":"https://img/b.png","strCategoryDescription":" Beef is meat. "}]}`,
		"/list.php?a=list":      `{"meals":[{"strArea":"Italian"},{"strArea":"American"}]}`,
	})
	ctx := context.Background()

	refs, err := c.Filter(ctx, FilterCategory, "Seafood")
	if err != nil || len(refs) != 1 || refs[0] != (MealRef{ID: "52802", Name: "Fish pie", Thumbnail: "https://img/f.jpg"}) {
		t.Fatalf("Filter: %+v, %v", refs, err)
	}
	if refs, err := c.Filter(ctx, FilterIngredient, "nothing"); err != nil || len(refs) != 0 || refs == nil {
		t.Fatalf("Filter empty: %#v, %v (want non-nil empty)", refs, err)
	}
	cats, err := c.Categories(ctx)
	if err != nil || len(cats) != 1 || cats[0] != (Category{Name: "Beef", Thumbnail: "https://img/b.png", Description: "Beef is meat."}) {
		t.Fatalf("Categories: %+v, %v", cats, err)
	}
	areas, err := c.Areas(ctx)
	if err != nil || len(areas) != 2 || areas[0] != "Italian" {
		t.Fatalf("Areas: %v, %v", areas, err)
	}
}

func TestClientErrors(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/search.php":
			http.Error(w, "boom", http.StatusBadGateway)
		case "/lookup.php":
			_, _ = w.Write([]byte(`{"meals":[`))
		case "/categories.php":
			time.Sleep(200 * time.Millisecond)
		}
	}))
	t.Cleanup(srv.Close)
	c := NewHTTPClient(srv.URL, 50*time.Millisecond)
	ctx := context.Background()

	if _, err := c.Search(ctx, "x"); err == nil {
		t.Error("non-2xx status must return an error")
	}
	if _, err := c.Lookup(ctx, "1"); err == nil || errors.Is(err, ErrNotFound) {
		t.Errorf("malformed JSON must be a non-NotFound error, got %v", err)
	}
	if _, err := c.Categories(ctx); err == nil {
		t.Error("timeout must return an error")
	}
}
