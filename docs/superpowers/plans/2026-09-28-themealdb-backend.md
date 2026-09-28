# TheMealDB Backend Integration Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Serve TheMealDB recipes from the Go API under `/api/recipes`, using a Postgres read-through cache that serves stale data when the upstream fails.

**Architecture:** A small `internal/mealdb` HTTP client normalises TheMealDB responses. A `services.RecipeService` wraps every client call in a read-through cache backed by a new `mealdb_cache` table: fresh hits are served, misses and expired entries are refetched, and an expired entry is served if the refetch fails. It also intersects filter lists and pages results. A `handlers.RecipeHandler` exposes four authenticated endpoints.

**Tech Stack:** Go 1.26, Gin 1.12, GORM 1.31 + Postgres (jsonb), `net/http`, `httptest`. No new dependencies.

**Spec:** `docs/superpowers/specs/2026-09-28-themealdb-integration-design.md` (read it first). This plan covers spec §2, §3, the backend half of §5, and §6 step 1. The frontend half gets its own plan after this PR merges.

**Branch:** `feat/themealdb-recipes` (already created, and it contains the spec commit).

## Global Constraints

- Module path: `github.com/meal-planner/backend`. Go `1.26.0`, as in `go.mod`. **Add no new dependencies.**
- TheMealDB base URL default: `https://www.themealdb.com/api/json/v1/1`. Timeout default is 5s. Detail TTL is 168h; search/filter TTL is 24h.
- Env vars: `MEALDB_BASE_URL`, `MEALDB_TIMEOUT_SECONDS`, `MEALDB_DETAIL_TTL_HOURS`, `MEALDB_SEARCH_TTL_HOURS`.
- All `/api/recipes*` routes require `middleware.AuthMiddleware(cfg)`.
- Error body shape: `{"error": "<message>"}`. Status codes: 400 for invalid input, 404 for an unknown recipe, 503 when the upstream fails with nothing cached, and 500 otherwise.
- JSON field names are camelCase and exactly as in the spec: `id`, `name`, `thumbnail`, `category`, `cuisine`, `ingredients[{name,measure}]`, `instructions[]`, `tags[]`, `youtubeUrl`, `sourceUrl`, and `{recipes,total,page,totalPages}`.
- Paging: `limit` defaults to 24, max 50 (larger values are capped, not rejected). `page` starts at 1.
- Every exported identifier needs a doc comment (the revive `exported` rule). Tests use the standard library only (no testify), like the existing tests.
- CI runs `golangci-lint` v2.14 with the repo's `.golangci.yml`, `go test -race`, and a **70% total coverage gate**.
- Every commit message ends with:
  ```
  Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>
  Claude-Session: https://claude.ai/code/session_014u1dJTmEC1ARhmephG3Zfk
  ```

## Review Focus

The inputs and conditions below are implied by the spec but easy to miss. Each one has a test in the task noted.
1. **`meals` is not an array.** TheMealDB can return `{"meals": null}` or a string such as `"Invalid ID"` instead of an array. This must be treated as "no results" (or not found for a lookup), never as a decode error or a 503. See Task 2.
2. **Corrupt or unreadable cache.** A cache row whose payload no longer decodes, or a cache `Get` that returns an error, must fall through to a refetch. It must not fail the request. See Task 4.
3. **Paging past the end and empty results.** `page` past the last page, or zero matches, must return `"recipes": []` (not `null`), with `total`/`totalPages` still correct. See Task 5.
4. **Non-numeric or odd recipe ids.** Examples: `/api/recipes/abc` and `/api/recipes/52772x`. These must give 400 without calling the upstream. See Task 6.
5. **`Unknown` cuisine and blank values.** TheMealDB's area list includes `"Unknown"`, and values can carry whitespace. `/cuisines` excludes `"Unknown"`, returns a sorted list, and trims values. Filter params that are only whitespace count as absent. See Tasks 4 and 5.

---

## File Structure

| File | Responsibility |
|---|---|
| `internal/config/config.go` (modify) | Four new MealDB settings |
| `internal/config/config_test.go` (modify) | Defaults and overrides for the new settings |
| `internal/mealdb/types.go` (create) | `Meal`, `Ingredient`, `MealRef`, `Category`, `FilterKind`, the `Client` interface, `ErrNotFound` |
| `internal/mealdb/normalize.go` (create) | Converts raw TheMealDB JSON into `Meal` (ingredients, steps, tags) |
| `internal/mealdb/client.go` (create) | `HTTPClient`, the `net/http` implementation of `Client` |
| `internal/mealdb/normalize_test.go`, `client_test.go` (create) | Unit tests; the client tests use an `httptest` fake server |
| `internal/models/cached_response.go` (create) | `CachedResponse` GORM model, table `mealdb_cache` |
| `internal/repository/cache_repository.go` (create) | `CacheRepository` interface and GORM implementation (upsert) |
| `internal/testutil/cache_repo.go` (create) | In-memory `CacheRepo` for tests |
| `internal/testutil/mealdb_client.go` (create) | Fake `mealdb.Client` with call counters |
| `internal/database/database.go` (modify) | AutoMigrate `CachedResponse` |
| `internal/services/recipe_service.go` (create) | `RecipeService`: read-through cache, filters, paging |
| `internal/services/recipe_service_test.go` (create) | Cache states, filters, paging |
| `internal/handlers/recipe_handler.go` (create) | HTTP handlers and error-to-status mapping |
| `internal/handlers/recipe_handler_test.go` (create) | Handler tests through the router |
| `internal/handlers/helpers_test.go`, `internal/router/router.go`, `internal/router/router_test.go` (modify) | Wire the recipe service into `router.New` |
| `.golangci.yml` (modify) | Let `wrapcheck` accept errors from the two new interfaces |
| `.env.example`, `.env.docker`, `docker-compose.yml`, `docker-compose.dev.yml`, `README.md` (modify) | Config and docs |

---

### Task 1: MealDB configuration

**Files:**
- Modify: `internal/config/config.go`
- Test: `internal/config/config_test.go`

**Interfaces:**
- Produces: `Config.MealDBBaseURL string`, `Config.MealDBTimeout time.Duration`, `Config.MealDBDetailTTL time.Duration`, `Config.MealDBSearchTTL time.Duration`.

- [ ] **Step 1: Write the failing test.** Append to `internal/config/config_test.go`:

```go
func TestLoadMealDBDefaults(t *testing.T) {
	for _, k := range []string{"MEALDB_BASE_URL", "MEALDB_TIMEOUT_SECONDS", "MEALDB_DETAIL_TTL_HOURS", "MEALDB_SEARCH_TTL_HOURS"} {
		t.Setenv(k, "")
	}
	cfg := Load()
	if cfg.MealDBBaseURL != "https://www.themealdb.com/api/json/v1/1" {
		t.Errorf("MealDBBaseURL = %q", cfg.MealDBBaseURL)
	}
	if cfg.MealDBTimeout != 5*time.Second {
		t.Errorf("MealDBTimeout = %v", cfg.MealDBTimeout)
	}
	if cfg.MealDBDetailTTL != 168*time.Hour {
		t.Errorf("MealDBDetailTTL = %v", cfg.MealDBDetailTTL)
	}
	if cfg.MealDBSearchTTL != 24*time.Hour {
		t.Errorf("MealDBSearchTTL = %v", cfg.MealDBSearchTTL)
	}
}

func TestLoadMealDBOverrides(t *testing.T) {
	t.Setenv("MEALDB_BASE_URL", "http://mealdb.test/api")
	t.Setenv("MEALDB_TIMEOUT_SECONDS", "2")
	t.Setenv("MEALDB_DETAIL_TTL_HOURS", "1")
	t.Setenv("MEALDB_SEARCH_TTL_HOURS", "3")
	cfg := Load()
	if cfg.MealDBBaseURL != "http://mealdb.test/api" || cfg.MealDBTimeout != 2*time.Second ||
		cfg.MealDBDetailTTL != time.Hour || cfg.MealDBSearchTTL != 3*time.Hour {
		t.Errorf("overrides not applied: %+v", cfg)
	}
}
```

If `config_test.go` does not import `time` yet, add it.

- [ ] **Step 2: Run the test to verify it fails.**
Run: `go test ./internal/config/ -run MealDB -v`
Expected: FAIL to compile with `cfg.MealDBBaseURL undefined`.

- [ ] **Step 3: Implement.** In `internal/config/config.go`, add these fields to `Config` after `RateLimitPerMin`:

```go
	// TheMealDB recipe source
	MealDBBaseURL   string
	MealDBTimeout   time.Duration
	MealDBDetailTTL time.Duration
	MealDBSearchTTL time.Duration
```

Then add these to the `Load()` literal after `RateLimitPerMin: ...`:

```go
		// TheMealDB
		MealDBBaseURL:   getEnv("MEALDB_BASE_URL", "https://www.themealdb.com/api/json/v1/1"),
		MealDBTimeout:   time.Duration(getEnvAsInt("MEALDB_TIMEOUT_SECONDS", 5)) * time.Second,
		MealDBDetailTTL: time.Duration(getEnvAsInt("MEALDB_DETAIL_TTL_HOURS", 168)) * time.Hour,
		MealDBSearchTTL: time.Duration(getEnvAsInt("MEALDB_SEARCH_TTL_HOURS", 24)) * time.Hour,
```

- [ ] **Step 4: Run the tests to verify they pass.**
Run: `go test ./internal/config/ -v`
Expected: PASS (both new tests and the existing ones).

- [ ] **Step 5: Update config files and commit.** Append to `.env.example` and `.env.docker`:

```
# TheMealDB recipe source (public key 1 is for development/educational use)
MEALDB_BASE_URL=https://www.themealdb.com/api/json/v1/1
MEALDB_TIMEOUT_SECONDS=5
MEALDB_DETAIL_TTL_HOURS=168
MEALDB_SEARCH_TTL_HOURS=24
```

In `docker-compose.yml`, and in `docker-compose.dev.yml` if its backend service has an `environment:` block, add these under the backend `environment:` after `RATE_LIMIT_PER_MIN`:

```yaml
      # TheMealDB
      MEALDB_BASE_URL: ${MEALDB_BASE_URL:-https://www.themealdb.com/api/json/v1/1}
      MEALDB_TIMEOUT_SECONDS: ${MEALDB_TIMEOUT_SECONDS:-5}
      MEALDB_DETAIL_TTL_HOURS: ${MEALDB_DETAIL_TTL_HOURS:-168}
      MEALDB_SEARCH_TTL_HOURS: ${MEALDB_SEARCH_TTL_HOURS:-24}
```

Then commit:

```bash
git add internal/config .env.example .env.docker docker-compose.yml docker-compose.dev.yml
git commit -m "feat(config): add TheMealDB settings" -m "Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_014u1dJTmEC1ARhmephG3Zfk"
```

---

### Task 2: `mealdb` package — types, normalisation, HTTP client

**Files:**
- Create: `internal/mealdb/types.go`, `internal/mealdb/normalize.go`, `internal/mealdb/client.go`
- Test: `internal/mealdb/normalize_test.go`, `internal/mealdb/client_test.go`
- Modify: `.golangci.yml`

**Interfaces:**
- Produces (used by Tasks 3–6):

```go
package mealdb
var ErrNotFound error
type Ingredient struct { Name, Measure string }                     // json: name, measure
type Meal struct { ID, Name, Category, Area, Thumbnail string;       // json: id,name,category,cuisine,thumbnail
                   Ingredients []Ingredient; Instructions, Tags []string; // json: ingredients,instructions,tags
                   YouTubeURL, SourceURL string }                     // json: youtubeUrl,sourceUrl (omitempty)
type MealRef struct { ID, Name, Thumbnail string }                   // json: id,name,thumbnail
type Category struct { Name, Thumbnail, Description string }         // json: name,thumbnail,description
type FilterKind string // FilterCategory "c", FilterArea "a", FilterIngredient "i"
type Client interface {
	Search(ctx context.Context, query string) ([]Meal, error)
	Filter(ctx context.Context, kind FilterKind, value string) ([]MealRef, error)
	Lookup(ctx context.Context, id string) (*Meal, error) // ErrNotFound if absent
	Categories(ctx context.Context) ([]Category, error)
	Areas(ctx context.Context) ([]string, error)
}
func NewHTTPClient(baseURL string, timeout time.Duration) *HTTPClient
```

*(The spec's `FilterByCategory`/`FilterByArea`/`FilterByIngredient` are merged into a single `Filter(kind, value)`. It's the same behaviour with less code.)*

- [ ] **Step 1: Create the types.** Write `internal/mealdb/types.go`:

```go
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
```

- [ ] **Step 2: Write the failing normalisation tests.** Write `internal/mealdb/normalize_test.go`:

```go
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
		"strTags": s("Meat, Casserole,,"), "strYoutube": s(""), "strSource": nil,
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
```

- [ ] **Step 3: Run the tests to verify they fail.**
Run: `go test ./internal/mealdb/ -v`
Expected: FAIL to compile with `undefined: rawMeal`, `toMeal`, `decodeMeals`.

- [ ] **Step 4: Implement the normalisation.** Write `internal/mealdb/normalize.go`:

```go
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
```

- [ ] **Step 5: Run the normalisation tests.**
Run: `go test ./internal/mealdb/ -v`
Expected: PASS.

- [ ] **Step 6: Write the failing client tests.** Write `internal/mealdb/client_test.go`:

```go
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
```

- [ ] **Step 7: Run the client tests to verify they fail.**
Run: `go test ./internal/mealdb/ -run 'Search|Filter|Client' -v`
Expected: FAIL to compile with `undefined: HTTPClient`, `NewHTTPClient`.

- [ ] **Step 8: Implement the client.** Write `internal/mealdb/client.go`:

```go
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
```

- [ ] **Step 9: Run all mealdb tests.**
Run: `go test ./internal/mealdb/ -race -v`
Expected: PASS for all 6 tests.

- [ ] **Step 10: Allow unwrapped errors from the new interfaces, then commit.** In `.golangci.yml` under `wrapcheck.ignore-interface-regexps`, change:

```yaml
      ignore-interface-regexps:
        - repository.UserRepository
```

to:

```yaml
      ignore-interface-regexps:
        - repository.UserRepository
        - repository.CacheRepository
        - mealdb.Client
```

Then commit:

```bash
git add internal/mealdb .golangci.yml
git commit -m "feat(mealdb): add TheMealDB client with response normalisation" -m "Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_014u1dJTmEC1ARhmephG3Zfk"
```

---

### Task 3: Cache model, repository and test doubles

**Files:**
- Create: `internal/models/cached_response.go`, `internal/repository/cache_repository.go`, `internal/testutil/cache_repo.go`, `internal/testutil/mealdb_client.go`
- Modify: `internal/database/database.go`
- Test: `internal/testutil/cache_repo_test.go`

**Interfaces:**
- Consumes: `mealdb.Client`, `mealdb.Meal`, `mealdb.MealRef`, `mealdb.Category`, `mealdb.FilterKind`, `mealdb.ErrNotFound` (Task 2).
- Produces:

```go
type models.CachedResponse struct { Key, Payload string; FetchedAt, ExpiresAt time.Time } // table mealdb_cache
type repository.CacheRepository interface {
	Get(key string) (*models.CachedResponse, error) // (nil, nil) when missing
	Upsert(entry *models.CachedResponse) error
}
func repository.NewCacheRepository(db *gorm.DB) repository.CacheRepository
type testutil.CacheRepo struct { Entries map[string]*models.CachedResponse; Err error }
func testutil.NewCacheRepo() *testutil.CacheRepo
type testutil.MealDBClient struct {
	Meals map[string]mealdb.Meal; SearchResults map[string][]mealdb.Meal
	Filters map[string][]mealdb.MealRef // key: string(kind)+"="+value, e.g. "c=Seafood"
	CategoryList []mealdb.Category; AreaList []string
	Err error; Calls int
}
func testutil.NewMealDBClient() *testutil.MealDBClient
```

- [ ] **Step 1: Write the failing test.** Write `internal/testutil/cache_repo_test.go`:

```go
package testutil

import (
	"errors"
	"testing"

	"github.com/meal-planner/backend/internal/models"
)

func TestCacheRepoUpsertAndGet(t *testing.T) {
	r := NewCacheRepo()
	if e, err := r.Get("k"); e != nil || err != nil {
		t.Fatalf("missing key: %v, %v", e, err)
	}
	_ = r.Upsert(&models.CachedResponse{Key: "k", Payload: `1`})
	_ = r.Upsert(&models.CachedResponse{Key: "k", Payload: `2`})
	if e, _ := r.Get("k"); e == nil || e.Payload != `2` {
		t.Fatalf("upsert must replace: %+v", e)
	}
	r.Err = errors.New("db down")
	if _, err := r.Get("k"); err == nil {
		t.Fatal("Err must be returned")
	}
}
```

- [ ] **Step 2: Run the test to verify it fails.**
Run: `go test ./internal/testutil/ -v`
Expected: FAIL to compile with `undefined: NewCacheRepo`, `models.CachedResponse`.

- [ ] **Step 3: Create the model.** Write `internal/models/cached_response.go`:

```go
package models

import "time"

// CachedResponse is a cached, normalised TheMealDB result.
type CachedResponse struct {
	Key       string    `gorm:"type:varchar(255);primaryKey"`
	Payload   string    `gorm:"type:jsonb;not null"`
	FetchedAt time.Time `gorm:"not null"`
	ExpiresAt time.Time `gorm:"not null;index"`
}

// TableName stores cached responses in mealdb_cache.
func (CachedResponse) TableName() string { return "mealdb_cache" }
```

- [ ] **Step 4: Create the repository.** Write `internal/repository/cache_repository.go`:

```go
package repository

import (
	"errors"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/meal-planner/backend/internal/models"
)

// CacheRepository stores cached TheMealDB responses.
type CacheRepository interface {
	// Get returns the entry for key, or (nil, nil) if there is none.
	Get(key string) (*models.CachedResponse, error)
	// Upsert inserts the entry or replaces the existing one with the same key.
	Upsert(entry *models.CachedResponse) error
}

type cacheRepository struct {
	db *gorm.DB
}

// NewCacheRepository returns a GORM-backed CacheRepository.
func NewCacheRepository(db *gorm.DB) CacheRepository {
	return &cacheRepository{db: db}
}

func (r *cacheRepository) Get(key string) (*models.CachedResponse, error) {
	var entry models.CachedResponse
	err := r.db.Where("key = ?", key).First(&entry).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &entry, nil
}

func (r *cacheRepository) Upsert(entry *models.CachedResponse) error {
	return r.db.Clauses(clause.OnConflict{UpdateAll: true}).Create(entry).Error
}
```

(`user_repository.go` returns raw GORM errors the same way, and wrapcheck ignores this package.)

- [ ] **Step 5: Create the in-memory cache.** Write `internal/testutil/cache_repo.go`:

```go
package testutil

import "github.com/meal-planner/backend/internal/models"

// CacheRepo is an in-memory repository.CacheRepository for tests.
type CacheRepo struct {
	Entries map[string]*models.CachedResponse
	// Err, when set, is returned by every method.
	Err error
}

// NewCacheRepo returns an empty in-memory cache repository.
func NewCacheRepo() *CacheRepo {
	return &CacheRepo{Entries: map[string]*models.CachedResponse{}}
}

// Get returns a copy of the entry for key, or nil.
func (r *CacheRepo) Get(key string) (*models.CachedResponse, error) {
	if r.Err != nil {
		return nil, r.Err
	}
	e, ok := r.Entries[key]
	if !ok {
		return nil, nil
	}
	cp := *e
	return &cp, nil
}

// Upsert stores a copy of the entry.
func (r *CacheRepo) Upsert(entry *models.CachedResponse) error {
	if r.Err != nil {
		return r.Err
	}
	cp := *entry
	r.Entries[entry.Key] = &cp
	return nil
}
```

- [ ] **Step 6: Create the fake TheMealDB client.** Write `internal/testutil/mealdb_client.go`:

```go
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
```

- [ ] **Step 7: Register the migration.** In `internal/database/database.go`, change:

```go
		&models.User{},
		// Add other models here as they are created
```

to:

```go
		&models.User{},
		&models.CachedResponse{},
		// Add other models here as they are created
```

- [ ] **Step 8: Run the tests, then build and vet.**
Run: `go test ./internal/testutil/ -v && go build ./... && go vet ./...`
Expected: PASS, and the build and vet are clean.

- [ ] **Step 9: Commit.**

```bash
git add internal/models/cached_response.go internal/repository/cache_repository.go internal/testutil internal/database/database.go
git commit -m "feat(cache): add mealdb_cache table, repository and test doubles" -m "Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_014u1dJTmEC1ARhmephG3Zfk"
```

---

### Task 4: RecipeService — read-through cache, categories, cuisines, lookup

**Files:**
- Create: `internal/services/recipe_service.go`
- Test: `internal/services/recipe_service_test.go`

**Interfaces:**
- Consumes: `mealdb.Client`, `mealdb.ErrNotFound`, `repository.CacheRepository`, `models.CachedResponse`, `testutil.NewMealDBClient`, `testutil.NewCacheRepo`.
- Produces (Task 5 adds `Search`; Task 6 consumes everything):

```go
var ErrUpstreamUnavailable, ErrRecipeNotFound, ErrInvalidSearch error
type RecipeCacheTTL struct { Detail, Search time.Duration }
type RecipeQuery struct { Q, Category, Cuisine, Ingredient string; Page, Limit int }
type RecipeSummary struct { ID, Name, Thumbnail, Category, Cuisine string } // json: id,name,thumbnail,category?,cuisine?
type RecipePage struct { Recipes []RecipeSummary; Total, Page, TotalPages int } // json: recipes,total,page,totalPages
type RecipeService interface {
	Categories(ctx context.Context) ([]mealdb.Category, error)
	Cuisines(ctx context.Context) ([]string, error)
	Get(ctx context.Context, id string) (*mealdb.Meal, error)
	Search(ctx context.Context, q RecipeQuery) (*RecipePage, error) // implemented in Task 5
}
func NewRecipeService(client mealdb.Client, cache repository.CacheRepository, ttl RecipeCacheTTL, now func() time.Time) RecipeService
```

- [ ] **Step 1: Write the failing tests.** Write `internal/services/recipe_service_test.go`:

```go
package services

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/meal-planner/backend/internal/mealdb"
	"github.com/meal-planner/backend/internal/models"
	"github.com/meal-planner/backend/internal/testutil"
)

type recipeFixture struct {
	svc    RecipeService
	client *testutil.MealDBClient
	cache  *testutil.CacheRepo
	now    time.Time
}

func newRecipeFixture(t *testing.T) *recipeFixture {
	t.Helper()
	f := &recipeFixture{
		client: testutil.NewMealDBClient(),
		cache:  testutil.NewCacheRepo(),
		now:    time.Date(2026, 9, 28, 12, 0, 0, 0, time.UTC),
	}
	f.svc = NewRecipeService(f.client, f.cache, RecipeCacheTTL{Detail: 7 * 24 * time.Hour, Search: 24 * time.Hour},
		func() time.Time { return f.now })
	return f
}

var teriyaki = mealdb.Meal{ID: "52772", Name: "Teriyaki", Category: "Chicken", Area: "Japanese",
	Ingredients: []mealdb.Ingredient{}, Instructions: []string{"Cook."}, Tags: []string{}}

func TestGetCachesAndReusesFreshEntry(t *testing.T) {
	f := newRecipeFixture(t)
	f.client.Meals["52772"] = teriyaki
	ctx := context.Background()

	for i := 0; i < 2; i++ {
		got, err := f.svc.Get(ctx, "52772")
		if err != nil || !reflect.DeepEqual(*got, teriyaki) {
			t.Fatalf("call %d: %+v, %v", i, got, err)
		}
	}
	if f.client.Calls != 1 {
		t.Errorf("fresh cache hit must not call upstream; calls = %d", f.client.Calls)
	}
	entry := f.cache.Entries["lookup:52772"]
	if entry == nil || !entry.ExpiresAt.Equal(f.now.Add(7*24*time.Hour)) {
		t.Errorf("cache entry/TTL wrong: %+v", entry)
	}
}

func TestGetRefetchesExpiredEntry(t *testing.T) {
	f := newRecipeFixture(t)
	f.client.Meals["52772"] = teriyaki
	ctx := context.Background()
	_, _ = f.svc.Get(ctx, "52772")

	f.now = f.now.Add(8 * 24 * time.Hour)
	updated := teriyaki
	updated.Name = "Teriyaki v2"
	f.client.Meals["52772"] = updated
	got, err := f.svc.Get(ctx, "52772")
	if err != nil || got.Name != "Teriyaki v2" || f.client.Calls != 2 {
		t.Fatalf("expired entry must be refetched: %+v, %v, calls=%d", got, err, f.client.Calls)
	}
}

func TestGetServesStaleOnUpstreamError(t *testing.T) {
	f := newRecipeFixture(t)
	f.client.Meals["52772"] = teriyaki
	ctx := context.Background()
	_, _ = f.svc.Get(ctx, "52772")

	f.now = f.now.Add(30 * 24 * time.Hour)
	f.client.Err = errors.New("upstream down")
	got, err := f.svc.Get(ctx, "52772")
	if err != nil || got.Name != "Teriyaki" {
		t.Fatalf("stale entry must be served when upstream fails: %+v, %v", got, err)
	}
}

func TestGetUpstreamErrorWithNothingCached(t *testing.T) {
	f := newRecipeFixture(t)
	f.client.Err = errors.New("upstream down")
	if _, err := f.svc.Get(context.Background(), "52772"); !errors.Is(err, ErrUpstreamUnavailable) {
		t.Fatalf("err = %v, want ErrUpstreamUnavailable", err)
	}
}

func TestGetNotFound(t *testing.T) {
	f := newRecipeFixture(t)
	if _, err := f.svc.Get(context.Background(), "99999"); !errors.Is(err, ErrRecipeNotFound) {
		t.Fatalf("err = %v, want ErrRecipeNotFound", err)
	}
}

func TestCorruptOrUnreadableCacheFallsThroughToFetch(t *testing.T) {
	f := newRecipeFixture(t)
	f.client.Meals["52772"] = teriyaki
	f.cache.Entries["lookup:52772"] = &models.CachedResponse{Key: "lookup:52772", Payload: `{not json`,
		FetchedAt: f.now, ExpiresAt: f.now.Add(time.Hour)}
	if got, err := f.svc.Get(context.Background(), "52772"); err != nil || got.Name != "Teriyaki" {
		t.Fatalf("corrupt fresh entry must be refetched: %+v, %v", got, err)
	}

	f.cache.Err = errors.New("db down")
	if got, err := f.svc.Get(context.Background(), "52772"); err != nil || got.Name != "Teriyaki" {
		t.Fatalf("cache read error must fall through to upstream: %+v, %v", got, err)
	}
}

func TestCuisinesSortedTrimmedWithoutUnknown(t *testing.T) {
	f := newRecipeFixture(t)
	f.client.AreaList = []string{"Italian", " American ", "Unknown", "British"}
	got, err := f.svc.Cuisines(context.Background())
	if err != nil || !reflect.DeepEqual(got, []string{"American", "British", "Italian"}) {
		t.Fatalf("Cuisines = %v, %v", got, err)
	}
}

func TestCategoriesCached(t *testing.T) {
	f := newRecipeFixture(t)
	f.client.CategoryList = []mealdb.Category{{Name: "Beef"}}
	ctx := context.Background()
	_, _ = f.svc.Categories(ctx)
	got, err := f.svc.Categories(ctx)
	if err != nil || len(got) != 1 || f.client.Calls != 1 {
		t.Fatalf("Categories = %v, %v, calls=%d", got, err, f.client.Calls)
	}
}
```

- [ ] **Step 2: Run the tests to verify they fail.**
Run: `go test ./internal/services/ -run 'Get|Corrupt|Cuisines|Categories' -v`
Expected: FAIL to compile with `undefined: RecipeService`, `NewRecipeService`.

- [ ] **Step 3: Implement.** Write `internal/services/recipe_service.go`:

```go
package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	"github.com/meal-planner/backend/internal/mealdb"
	"github.com/meal-planner/backend/internal/models"
	"github.com/meal-planner/backend/internal/repository"
)

// Recipe service errors that handlers map to HTTP statuses.
var (
	ErrUpstreamUnavailable = errors.New("recipes are temporarily unavailable")
	ErrRecipeNotFound      = errors.New("recipe not found")
	ErrInvalidSearch       = errors.New("provide at least one of q, category, cuisine or ingredient")
)

// Paging limits for Search.
const (
	defaultRecipeLimit = 24
	maxRecipeLimit     = 50
)

// RecipeCacheTTL sets how long cached TheMealDB results stay fresh.
type RecipeCacheTTL struct {
	Detail time.Duration // lookups, categories, cuisines
	Search time.Duration // name searches and filters
}

// RecipeQuery selects recipes; at least one of Q, Category, Cuisine or Ingredient is required.
type RecipeQuery struct {
	Q, Category, Cuisine, Ingredient string
	Page, Limit                      int
}

// RecipeSummary is a recipe in a result list.
type RecipeSummary struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Thumbnail string `json:"thumbnail"`
	Category  string `json:"category,omitempty"`
	Cuisine   string `json:"cuisine,omitempty"`
}

// RecipePage is one page of search results.
type RecipePage struct {
	Recipes    []RecipeSummary `json:"recipes"`
	Total      int             `json:"total"`
	Page       int             `json:"page"`
	TotalPages int             `json:"totalPages"`
}

// RecipeService serves TheMealDB recipes through a read-through cache.
type RecipeService interface {
	Categories(ctx context.Context) ([]mealdb.Category, error)
	Cuisines(ctx context.Context) ([]string, error)
	Get(ctx context.Context, id string) (*mealdb.Meal, error)
	Search(ctx context.Context, q RecipeQuery) (*RecipePage, error)
}

type recipeService struct {
	client mealdb.Client
	cache  repository.CacheRepository
	ttl    RecipeCacheTTL
	now    func() time.Time
}

// NewRecipeService creates a RecipeService. now is injectable for tests; pass time.Now.
func NewRecipeService(client mealdb.Client, cache repository.CacheRepository, ttl RecipeCacheTTL, now func() time.Time) RecipeService {
	return &recipeService{client: client, cache: cache, ttl: ttl, now: now}
}

// cached returns the value for key: a fresh cache entry if there is one,
// otherwise the upstream result (which is then stored). If the upstream fails,
// an expired entry is served instead; with no entry at all the error is
// ErrUpstreamUnavailable. mealdb.ErrNotFound is passed through unchanged.
func cached[T any](ctx context.Context, s *recipeService, key string, ttl time.Duration,
	fetch func(context.Context) (T, error)) (T, error) {
	var zero T
	entry, err := s.cache.Get(key)
	if err != nil {
		log.Printf("recipe cache: read %q: %v", key, err)
		entry = nil
	}
	var stale *T
	if entry != nil {
		var v T
		if jsonErr := json.Unmarshal([]byte(entry.Payload), &v); jsonErr == nil {
			if s.now().Before(entry.ExpiresAt) {
				return v, nil
			}
			stale = &v
		}
	}

	v, err := fetch(ctx)
	if err != nil {
		if errors.Is(err, mealdb.ErrNotFound) {
			return zero, err
		}
		if stale != nil {
			log.Printf("recipe cache: serving stale %q after upstream error: %v", key, err)
			return *stale, nil
		}
		return zero, fmt.Errorf("%w: %w", ErrUpstreamUnavailable, err)
	}

	if payload, mErr := json.Marshal(v); mErr == nil {
		now := s.now()
		if upErr := s.cache.Upsert(&models.CachedResponse{Key: key, Payload: string(payload),
			FetchedAt: now, ExpiresAt: now.Add(ttl)}); upErr != nil {
			log.Printf("recipe cache: write %q: %v", key, upErr)
		}
	}
	return v, nil
}

func (s *recipeService) Categories(ctx context.Context) ([]mealdb.Category, error) {
	return cached(ctx, s, "categories", s.ttl.Detail, s.client.Categories)
}

func (s *recipeService) Cuisines(ctx context.Context) ([]string, error) {
	areas, err := cached(ctx, s, "areas", s.ttl.Detail, s.client.Areas)
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(areas))
	for _, a := range areas {
		if a = strings.TrimSpace(a); a != "" && !strings.EqualFold(a, "Unknown") {
			out = append(out, a)
		}
	}
	sort.Strings(out)
	return out, nil
}

func (s *recipeService) Get(ctx context.Context, id string) (*mealdb.Meal, error) {
	meal, err := cached(ctx, s, "lookup:"+id, s.ttl.Detail, func(ctx context.Context) (*mealdb.Meal, error) {
		return s.client.Lookup(ctx, id)
	})
	if errors.Is(err, mealdb.ErrNotFound) {
		return nil, ErrRecipeNotFound
	}
	if err != nil {
		return nil, err
	}
	return meal, nil
}

// Search is implemented in recipe_search.go.
```

In the same package, create `internal/services/recipe_search.go` with a temporary stub so the package compiles (Task 5 replaces it):

```go
package services

import "context"

func (s *recipeService) Search(_ context.Context, _ RecipeQuery) (*RecipePage, error) {
	return nil, ErrInvalidSearch
}
```

- [ ] **Step 4: Run the tests to verify they pass.**
Run: `go test ./internal/services/ -race -v`
Expected: PASS for the 8 new tests and the existing service tests.

- [ ] **Step 5: Commit.**

```bash
git add internal/services/recipe_service.go internal/services/recipe_search.go internal/services/recipe_service_test.go
git commit -m "feat(recipes): add RecipeService with stale-on-error read-through cache" -m "Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_014u1dJTmEC1ARhmephG3Zfk"
```

---

### Task 5: RecipeService.Search — combined filters and paging

**Files:**
- Modify: `internal/services/recipe_search.go` (replace the stub)
- Test: `internal/services/recipe_search_test.go`

**Interfaces:**
- Consumes: `cached`, `recipeService`, `RecipeQuery`, `RecipeSummary`, `RecipePage`, `ErrInvalidSearch`, `defaultRecipeLimit`, `maxRecipeLimit` (Task 4); `mealdb.FilterCategory/FilterArea/FilterIngredient` (Task 2).
- Produces: `(*recipeService).Search`, which follows the `RecipeService.Search` contract.

**Cache keys:** `search:q=<lower-cased trimmed q>` and `filter:<kind>=<trimmed value>`. The filter value keeps its case: TheMealDB filter values may be case-sensitive, and lower-casing the key could cache an empty result under a key that a correctly-cased request later reads. *(This refines the spec §2.2 "lower-cased" note, which applies to search only.)*

- [ ] **Step 1: Write the failing tests.** Write `internal/services/recipe_search_test.go`:

```go
package services

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/meal-planner/backend/internal/mealdb"
)

func ref(id string) mealdb.MealRef { return mealdb.MealRef{ID: id, Name: "Meal " + id, Thumbnail: "t" + id} }

func ids(p *RecipePage) []string {
	out := []string{}
	for _, r := range p.Recipes {
		out = append(out, r.ID)
	}
	return out
}

func TestSearchRequiresCriteria(t *testing.T) {
	f := newRecipeFixture(t)
	for _, q := range []RecipeQuery{{}, {Q: "  ", Category: "\t"}} {
		if _, err := f.svc.Search(context.Background(), q); !errors.Is(err, ErrInvalidSearch) {
			t.Errorf("%+v: err = %v, want ErrInvalidSearch", q, err)
		}
	}
	if f.client.Calls != 0 {
		t.Errorf("invalid search must not call upstream")
	}
}

func TestSearchSingleFilterSetsKnownFields(t *testing.T) {
	f := newRecipeFixture(t)
	f.client.Filters["c=Seafood"] = []mealdb.MealRef{ref("1"), ref("2")}
	p, err := f.svc.Search(context.Background(), RecipeQuery{Category: " Seafood "})
	if err != nil || p.Total != 2 || p.Recipes[0] != (RecipeSummary{ID: "1", Name: "Meal 1", Thumbnail: "t1", Category: "Seafood"}) {
		t.Fatalf("Search = %+v, %v", p, err)
	}
}

func TestSearchIntersectsFiltersKeepingFirstOrder(t *testing.T) {
	f := newRecipeFixture(t)
	f.client.Filters["c=Vegetarian"] = []mealdb.MealRef{ref("3"), ref("1"), ref("2")}
	f.client.Filters["a=Italian"] = []mealdb.MealRef{ref("2"), ref("3"), ref("9")}
	f.client.Filters["i=garlic"] = []mealdb.MealRef{ref("3"), ref("2"), ref("7")}
	p, err := f.svc.Search(context.Background(), RecipeQuery{Category: "Vegetarian", Cuisine: "Italian", Ingredient: "garlic"})
	if err != nil || fmt.Sprint(ids(p)) != "[3 2]" || p.Recipes[0].Cuisine != "Italian" || p.Recipes[0].Category != "Vegetarian" {
		t.Fatalf("Search = %+v, %v", p, err)
	}
}

func TestSearchByNameNarrowedByFilters(t *testing.T) {
	f := newRecipeFixture(t)
	// the fake is keyed by the query exactly as the service passes it (trimmed, case kept)
	f.client.SearchResults["Chicken"] = []mealdb.Meal{
		{ID: "1", Name: "Chicken Tikka", Category: "Chicken", Area: "Indian"},
		{ID: "2", Name: "Chicken Parm", Category: "Chicken", Area: "Italian"},
		{ID: "3", Name: "Chicken Soup", Category: "Starter", Area: "Italian"},
	}
	f.client.Filters["i=garlic"] = []mealdb.MealRef{ref("2"), ref("3")}
	p, err := f.svc.Search(context.Background(), RecipeQuery{Q: "Chicken", Cuisine: "italian", Ingredient: "garlic"})
	if err != nil || fmt.Sprint(ids(p)) != "[2 3]" || p.Recipes[1] != (RecipeSummary{ID: "3", Name: "Chicken Soup", Category: "Starter", Cuisine: "Italian"}) {
		t.Fatalf("Search = %+v, %v", p, err)
	}
	if _, ok := f.cache.Entries["search:q=chicken"]; !ok {
		t.Error("name search must be cached under a lower-cased key")
	}
}

func TestSearchPaging(t *testing.T) {
	f := newRecipeFixture(t)
	refs := make([]mealdb.MealRef, 0, 60)
	for i := 1; i <= 60; i++ {
		refs = append(refs, ref(fmt.Sprint(i)))
	}
	f.client.Filters["c=Beef"] = refs
	ctx := context.Background()

	p, _ := f.svc.Search(ctx, RecipeQuery{Category: "Beef"})
	if len(p.Recipes) != 24 || p.Page != 1 || p.Total != 60 || p.TotalPages != 3 {
		t.Errorf("default paging: len=%d page=%d total=%d pages=%d", len(p.Recipes), p.Page, p.Total, p.TotalPages)
	}
	p, _ = f.svc.Search(ctx, RecipeQuery{Category: "Beef", Page: 2, Limit: 500})
	if len(p.Recipes) != 10 || p.Recipes[0].ID != "51" || p.TotalPages != 2 {
		t.Errorf("limit must cap at 50: len=%d first=%s pages=%d", len(p.Recipes), p.Recipes[0].ID, p.TotalPages)
	}
	p, _ = f.svc.Search(ctx, RecipeQuery{Category: "Beef", Page: 9})
	if p.Recipes == nil || len(p.Recipes) != 0 || p.Total != 60 {
		t.Errorf("page past the end must be an empty non-nil list: %+v", p)
	}
}

func TestSearchNoMatchesIsEmptyNotNil(t *testing.T) {
	f := newRecipeFixture(t)
	p, err := f.svc.Search(context.Background(), RecipeQuery{Q: "zzz"})
	if err != nil || p.Recipes == nil || p.Total != 0 || p.TotalPages != 0 || p.Page != 1 {
		t.Fatalf("no matches: %+v, %v", p, err)
	}
}

func TestSearchUpstreamErrorPropagates(t *testing.T) {
	f := newRecipeFixture(t)
	f.client.Err = errors.New("down")
	if _, err := f.svc.Search(context.Background(), RecipeQuery{Category: "Beef"}); !errors.Is(err, ErrUpstreamUnavailable) {
		t.Fatalf("err = %v, want ErrUpstreamUnavailable", err)
	}
}
```

- [ ] **Step 2: Run the tests to verify they fail.**
Run: `go test ./internal/services/ -run Search -v`
Expected: FAIL (the stub returns `ErrInvalidSearch` for everything).

- [ ] **Step 3: Implement.** Replace `internal/services/recipe_search.go` with:

```go
package services

import (
	"context"
	"strings"

	"github.com/meal-planner/backend/internal/mealdb"
)

func (s *recipeService) Search(ctx context.Context, q RecipeQuery) (*RecipePage, error) {
	q.Q = strings.TrimSpace(q.Q)
	q.Category = strings.TrimSpace(q.Category)
	q.Cuisine = strings.TrimSpace(q.Cuisine)
	q.Ingredient = strings.TrimSpace(q.Ingredient)
	if q.Q == "" && q.Category == "" && q.Cuisine == "" && q.Ingredient == "" {
		return nil, ErrInvalidSearch
	}

	var (
		results []RecipeSummary
		err     error
	)
	if q.Q != "" {
		results, err = s.searchByName(ctx, q)
	} else {
		results, err = s.searchByFilters(ctx, q)
	}
	if err != nil {
		return nil, err
	}
	return paginate(results, q.Page, q.Limit), nil
}

// searchByName runs TheMealDB's name search, then narrows the full meals by
// category/cuisine and, if requested, by the main-ingredient filter.
func (s *recipeService) searchByName(ctx context.Context, q RecipeQuery) ([]RecipeSummary, error) {
	key := "search:q=" + strings.ToLower(q.Q)
	meals, err := cached(ctx, s, key, s.ttl.Search, func(ctx context.Context) ([]mealdb.Meal, error) {
		return s.client.Search(ctx, q.Q)
	})
	if err != nil {
		return nil, err
	}
	var allowed map[string]bool
	if q.Ingredient != "" {
		refs, err := s.filter(ctx, mealdb.FilterIngredient, q.Ingredient)
		if err != nil {
			return nil, err
		}
		allowed = idSet(refs)
	}

	out := make([]RecipeSummary, 0, len(meals))
	for i := range meals {
		m := &meals[i]
		if q.Category != "" && !strings.EqualFold(m.Category, q.Category) {
			continue
		}
		if q.Cuisine != "" && !strings.EqualFold(m.Area, q.Cuisine) {
			continue
		}
		if allowed != nil && !allowed[m.ID] {
			continue
		}
		out = append(out, RecipeSummary{ID: m.ID, Name: m.Name, Thumbnail: m.Thumbnail, Category: m.Category, Cuisine: m.Area})
	}
	return out, nil
}

// searchByFilters intersects each requested filter list, keeping the order of the first.
func (s *recipeService) searchByFilters(ctx context.Context, q RecipeQuery) ([]RecipeSummary, error) {
	type filterReq struct {
		kind  mealdb.FilterKind
		value string
	}
	var reqs []filterReq
	for _, r := range []filterReq{
		{mealdb.FilterCategory, q.Category}, {mealdb.FilterArea, q.Cuisine}, {mealdb.FilterIngredient, q.Ingredient},
	} {
		if r.value != "" {
			reqs = append(reqs, r)
		}
	}

	var base []mealdb.MealRef
	var keep []map[string]bool
	for i, r := range reqs {
		refs, err := s.filter(ctx, r.kind, r.value)
		if err != nil {
			return nil, err
		}
		if i == 0 {
			base = refs
		} else {
			keep = append(keep, idSet(refs))
		}
	}

	out := make([]RecipeSummary, 0, len(base))
next:
	for _, ref := range base {
		for _, set := range keep {
			if !set[ref.ID] {
				continue next
			}
		}
		out = append(out, RecipeSummary{ID: ref.ID, Name: ref.Name, Thumbnail: ref.Thumbnail, Category: q.Category, Cuisine: q.Cuisine})
	}
	return out, nil
}

func (s *recipeService) filter(ctx context.Context, kind mealdb.FilterKind, value string) ([]mealdb.MealRef, error) {
	key := "filter:" + string(kind) + "=" + value
	return cached(ctx, s, key, s.ttl.Search, func(ctx context.Context) ([]mealdb.MealRef, error) {
		return s.client.Filter(ctx, kind, value)
	})
}

func idSet(refs []mealdb.MealRef) map[string]bool {
	set := make(map[string]bool, len(refs))
	for _, r := range refs {
		set[r.ID] = true
	}
	return set
}

func paginate(all []RecipeSummary, page, limit int) *RecipePage {
	if limit <= 0 {
		limit = defaultRecipeLimit
	}
	if limit > maxRecipeLimit {
		limit = maxRecipeLimit
	}
	if page < 1 {
		page = 1
	}
	total := len(all)
	totalPages := (total + limit - 1) / limit
	start := min((page-1)*limit, total)
	end := min(start+limit, total)
	recipes := make([]RecipeSummary, 0, end-start)
	recipes = append(recipes, all[start:end]...)
	return &RecipePage{Recipes: recipes, Total: total, Page: page, TotalPages: totalPages}
}
```

- [ ] **Step 4: Run the tests to verify they pass.**
Run: `go test ./internal/services/ -race -v`
Expected: PASS for all service tests.

- [ ] **Step 5: Commit.**

```bash
git add internal/services/recipe_search.go internal/services/recipe_search_test.go
git commit -m "feat(recipes): combine filters and page recipe search results" -m "Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_014u1dJTmEC1ARhmephG3Zfk"
```

---

### Task 6: RecipeHandler, routes and wiring

**Files:**
- Create: `internal/handlers/recipe_handler.go`
- Test: `internal/handlers/recipe_handler_test.go`
- Modify: `internal/handlers/helpers_test.go`, `internal/router/router.go`, `internal/router/router_test.go`

**Interfaces:**
- Consumes: `services.RecipeService`, `services.RecipeQuery`, `services.NewRecipeService`, `services.RecipeCacheTTL`, `services.ErrRecipeNotFound`, `services.ErrUpstreamUnavailable`, `services.ErrInvalidSearch` (Tasks 4 and 5); `mealdb.NewHTTPClient` (Task 2); `repository.NewCacheRepository` and the `testutil` doubles (Task 3); `Config.MealDB*` (Task 1).
- Produces: `func handlers.NewRecipeHandler(recipes services.RecipeService) *handlers.RecipeHandler`. **The signature changes** to `func router.New(userRepo repository.UserRepository, recipes services.RecipeService, cfg *config.Config) *gin.Engine`.

- [ ] **Step 1: Update the test server.** In `internal/handlers/helpers_test.go`:
  - Add the imports `"github.com/meal-planner/backend/internal/services"` and `"net/http"`. Keep `time`.
  - Add a `mealdb *testutil.MealDBClient` field to `testServer`.
  - Replace the last two lines of `newTestServer` with:

```go
	repo := testutil.NewUserRepo()
	client := testutil.NewMealDBClient()
	recipes := services.NewRecipeService(client, testutil.NewCacheRepo(),
		services.RecipeCacheTTL{Detail: time.Hour, Search: time.Hour}, time.Now)
	return &testServer{engine: router.New(repo, recipes, cfg), repo: repo, mealdb: client}
```

Also add this helper at the end of the file, which the recipe tests use to decode array bodies:

```go
// getJSON sends an authenticated GET and decodes the body into out.
func (s *testServer) getJSON(t *testing.T, path, token string, out any) int {
	t.Helper()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, path, http.NoBody)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	s.engine.ServeHTTP(w, req)
	if out != nil {
		if err := json.Unmarshal(w.Body.Bytes(), out); err != nil {
			t.Fatalf("decode %s: %v (%s)", path, err, w.Body.String())
		}
	}
	return w.Code
}
```

- [ ] **Step 2: Write the failing handler tests.** Write `internal/handlers/recipe_handler_test.go`:

```go
package handlers_test

import (
	"errors"
	"net/http"
	"testing"

	"github.com/meal-planner/backend/internal/mealdb"
)

func TestRecipesRequireAuth(t *testing.T) {
	s := newTestServer(t)
	for _, path := range []string{"/api/recipes?q=x", "/api/recipes/52772", "/api/recipes/categories", "/api/recipes/cuisines"} {
		if code, _ := s.do(t, http.MethodGet, path, nil, ""); code != http.StatusUnauthorized {
			t.Errorf("%s without token: %d, want 401", path, code)
		}
	}
}

func TestGetRecipe(t *testing.T) {
	s := newTestServer(t)
	_, token := s.seedUser(t, "r@example.com")
	s.mealdb.Meals["52772"] = mealdb.Meal{ID: "52772", Name: "Teriyaki", Area: "Japanese",
		Ingredients: []mealdb.Ingredient{{Name: "soy sauce", Measure: "1 cup"}}, Instructions: []string{"Cook."},
		Tags: []string{}, YouTubeURL: "https://youtube.test/x"}

	var got map[string]any
	if code := s.getJSON(t, "/api/recipes/52772", token, &got); code != http.StatusOK {
		t.Fatalf("status %d", code)
	}
	if got["cuisine"] != "Japanese" || got["youtubeUrl"] != "https://youtube.test/x" || got["sourceUrl"] != nil {
		t.Errorf("unexpected body %v", got)
	}
	ing := got["ingredients"].([]any)[0].(map[string]any)
	if ing["name"] != "soy sauce" || ing["measure"] != "1 cup" {
		t.Errorf("ingredient %v", ing)
	}
}

func TestGetRecipeErrors(t *testing.T) {
	s := newTestServer(t)
	_, token := s.seedUser(t, "e@example.com")

	for _, id := range []string{"abc", "52772x", "-1"} {
		if code, resp := s.do(t, http.MethodGet, "/api/recipes/"+id, nil, token); code != http.StatusBadRequest || resp["error"] == nil {
			t.Errorf("id %q: %d %v, want 400 with error", id, code, resp)
		}
	}
	if s.mealdb.Calls != 0 {
		t.Errorf("invalid ids must not reach upstream (calls=%d)", s.mealdb.Calls)
	}
	if code, _ := s.do(t, http.MethodGet, "/api/recipes/99999", nil, token); code != http.StatusNotFound {
		t.Errorf("unknown id: %d, want 404", code)
	}
	s.mealdb.Err = errors.New("down")
	if code, resp := s.do(t, http.MethodGet, "/api/recipes/11111", nil, token); code != http.StatusServiceUnavailable || resp["error"] == nil {
		t.Errorf("upstream down: %d %v, want 503", code, resp)
	}
}

func TestSearchRecipes(t *testing.T) {
	s := newTestServer(t)
	_, token := s.seedUser(t, "s@example.com")
	s.mealdb.Filters["c=Seafood"] = []mealdb.MealRef{{ID: "1", Name: "Fish pie", Thumbnail: "t1"}}

	var page struct {
		Recipes []map[string]any `json:"recipes"`
		Total   int              `json:"total"`
		Page    int              `json:"page"`
		Pages   int              `json:"totalPages"`
	}
	if code := s.getJSON(t, "/api/recipes?category=Seafood&page=1&limit=10", token, &page); code != http.StatusOK {
		t.Fatalf("status %d", code)
	}
	if page.Total != 1 || page.Page != 1 || page.Pages != 1 || page.Recipes[0]["category"] != "Seafood" {
		t.Errorf("page %+v", page)
	}

	for _, path := range []string{"/api/recipes", "/api/recipes?category=Seafood&page=0", "/api/recipes?category=Seafood&limit=abc"} {
		if code, _ := s.do(t, http.MethodGet, path, nil, token); code != http.StatusBadRequest {
			t.Errorf("%s: %d, want 400", path, code)
		}
	}
}

func TestCategoriesAndCuisines(t *testing.T) {
	s := newTestServer(t)
	_, token := s.seedUser(t, "c@example.com")
	s.mealdb.CategoryList = []mealdb.Category{{Name: "Beef", Thumbnail: "b.png", Description: "Beef."}}
	s.mealdb.AreaList = []string{"Italian", "Unknown", "American"}

	var cats []map[string]any
	if code := s.getJSON(t, "/api/recipes/categories", token, &cats); code != http.StatusOK || cats[0]["name"] != "Beef" {
		t.Errorf("categories %d %v", code, cats)
	}
	var cuisines []string
	if code := s.getJSON(t, "/api/recipes/cuisines", token, &cuisines); code != http.StatusOK || len(cuisines) != 2 || cuisines[0] != "American" {
		t.Errorf("cuisines %d %v", code, cuisines)
	}
}
```

- [ ] **Step 3: Run the tests to verify they fail.**
Run: `go test ./internal/handlers/ -run 'Recipe|Categories' -v`
Expected: FAIL to compile with `too many arguments in call to router.New`.

- [ ] **Step 4: Implement the handler.** Write `internal/handlers/recipe_handler.go`:

```go
package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/meal-planner/backend/internal/services"
)

const (
	msgInvalidRecipeID   = "invalid recipe id"
	msgInvalidPaging     = "page and limit must be positive integers"
	msgRecipesDown       = "recipes are temporarily unavailable, please try again shortly"
	msgRecipeNotFound    = "recipe not found"
	msgInternalServerErr = "internal server error"
)

// RecipeHandler serves TheMealDB recipes.
type RecipeHandler struct {
	recipes services.RecipeService
}

// NewRecipeHandler creates a RecipeHandler.
func NewRecipeHandler(recipes services.RecipeService) *RecipeHandler {
	return &RecipeHandler{recipes: recipes}
}

// Categories returns all recipe categories.
func (h *RecipeHandler) Categories(c *gin.Context) {
	cats, err := h.recipes.Categories(c.Request.Context())
	if err != nil {
		recipeError(c, err)
		return
	}
	c.JSON(http.StatusOK, cats)
}

// Cuisines returns all cuisine names, sorted.
func (h *RecipeHandler) Cuisines(c *gin.Context) {
	cuisines, err := h.recipes.Cuisines(c.Request.Context())
	if err != nil {
		recipeError(c, err)
		return
	}
	c.JSON(http.StatusOK, cuisines)
}

// Get returns one recipe by its numeric TheMealDB id.
func (h *RecipeHandler) Get(c *gin.Context) {
	id := c.Param("id")
	if !isPositiveInt(id) {
		c.JSON(http.StatusBadRequest, gin.H{"error": msgInvalidRecipeID})
		return
	}
	meal, err := h.recipes.Get(c.Request.Context(), id)
	if err != nil {
		recipeError(c, err)
		return
	}
	c.JSON(http.StatusOK, meal)
}

// Search returns a page of recipes matching q/category/cuisine/ingredient.
func (h *RecipeHandler) Search(c *gin.Context) {
	page, okPage := optionalPositiveInt(c.Query("page"))
	limit, okLimit := optionalPositiveInt(c.Query("limit"))
	if !okPage || !okLimit {
		c.JSON(http.StatusBadRequest, gin.H{"error": msgInvalidPaging})
		return
	}
	result, err := h.recipes.Search(c.Request.Context(), services.RecipeQuery{
		Q: c.Query("q"), Category: c.Query("category"), Cuisine: c.Query("cuisine"), Ingredient: c.Query("ingredient"),
		Page: page, Limit: limit,
	})
	if err != nil {
		recipeError(c, err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func recipeError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, services.ErrInvalidSearch):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	case errors.Is(err, services.ErrRecipeNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": msgRecipeNotFound})
	case errors.Is(err, services.ErrUpstreamUnavailable):
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": msgRecipesDown})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": msgInternalServerErr})
	}
}

func isPositiveInt(s string) bool {
	n, err := strconv.Atoi(s)
	return err == nil && n > 0 && strconv.Itoa(n) == s
}

// optionalPositiveInt parses an optional query value: "" gives (0, true).
func optionalPositiveInt(s string) (int, bool) {
	if s == "" {
		return 0, true
	}
	if !isPositiveInt(s) {
		return 0, false
	}
	n, _ := strconv.Atoi(s)
	return n, true
}
```

- [ ] **Step 5: Wire up the router.** In `internal/router/router.go`:
  - Add the imports `"time"` and `"github.com/meal-planner/backend/internal/mealdb"`.
  - Replace `Setup` and the `New` signature:

```go
// Setup initializes and configures the router backed by the given database.
func Setup(db *gorm.DB, cfg *config.Config) *gin.Engine {
	recipes := services.NewRecipeService(
		mealdb.NewHTTPClient(cfg.MealDBBaseURL, cfg.MealDBTimeout),
		repository.NewCacheRepository(db),
		services.RecipeCacheTTL{Detail: cfg.MealDBDetailTTL, Search: cfg.MealDBSearchTTL},
		time.Now,
	)
	return New(repository.NewUserRepository(db), recipes, cfg)
}

// New builds the router on top of the given user repository and recipe service.
func New(userRepo repository.UserRepository, recipes services.RecipeService, cfg *config.Config) *gin.Engine {
```

  - In the `/api` info map, after the `"auth": gin.H{...},` entry, add:

```go
				"recipes": gin.H{
					"search":     "GET /api/recipes?q=&category=&cuisine=&ingredient=&page=&limit= (protected)",
					"detail":     "GET /api/recipes/:id (protected)",
					"categories": "GET /api/recipes/categories (protected)",
					"cuisines":   "GET /api/recipes/cuisines (protected)",
				},
```

  - After the `userHandler := ...` line, add `recipeHandler := handlers.NewRecipeHandler(recipes)`.
  - After the closing `}` of the `auth` group block (still inside the `api` block), add:

```go
		// Recipe routes (protected)
		recipeRoutes := api.Group("/recipes")
		recipeRoutes.Use(middleware.AuthMiddleware(cfg))
		{
			recipeRoutes.GET("", recipeHandler.Search)
			recipeRoutes.GET("/categories", recipeHandler.Categories)
			recipeRoutes.GET("/cuisines", recipeHandler.Cuisines)
			recipeRoutes.GET("/:id", recipeHandler.Get)
		}
```

- [ ] **Step 6: Update the router tests.** In `internal/router/router_test.go`, add the imports `"time"` and `"github.com/meal-planner/backend/internal/services"`, then add this helper:

```go
func testRecipes() services.RecipeService {
	return services.NewRecipeService(testutil.NewMealDBClient(), testutil.NewCacheRepo(),
		services.RecipeCacheTTL{Detail: time.Hour, Search: time.Hour}, time.Now)
}
```

Then change both `New(testutil.NewUserRepo(), testConfig(...))` calls to `New(testutil.NewUserRepo(), testRecipes(), testConfig(...))`.

- [ ] **Step 7: Run the whole suite with coverage.**
Run: `go build ./... && go vet ./... && go test ./... -race -coverprofile=coverage.out && go tool cover -func=coverage.out | tail -1`
Expected: every package passes, and the total is **≥ 70%**. If the total is under 70%, stop and report which package is low; don't lower the gate.

- [ ] **Step 8: Run lint.**
Run: `go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.14.0 run ./...`
Expected: `0 issues.` Fix any findings in the new code. The most likely one is `goconst`/`dupl` in tests; address it by extracting a helper, not by adding a `//nolint`.

- [ ] **Step 9: Commit.**

```bash
git add internal/handlers internal/router
git commit -m "feat(recipes): add /api/recipes endpoints backed by TheMealDB" -m "Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_014u1dJTmEC1ARhmephG3Zfk"
```

---

### Task 7: README, manual verification against real TheMealDB, PR

**Files:**
- Modify: `README.md`

- [ ] **Step 1: Document the feature.** In `README.md`:
  - Add the four recipe endpoints to the endpoint table: path, `Yes` for auth, and a purpose.
  - Add the four `MEALDB_*` variables, with their defaults, to the environment-variable table.
  - Add a short "Recipes (TheMealDB)" section covering four things. First, recipes come from TheMealDB through a Postgres read-through cache (`mealdb_cache`). Second, the TTLs. Third, stale entries are served if TheMealDB fails. Fourth, attribution: "Recipe data and images from TheMealDB (themealdb.com). The public API key `1` is for development and educational use; a public production deployment should follow TheMealDB's supporter terms."
  - Remove any roadmap line that says a recipe service is unbuilt.

- [ ] **Step 2: Test the endpoints against real TheMealDB.** Run:

```bash
docker compose up -d --build
# register + login to get a token
curl -s -X POST localhost:3001/api/auth/register -H 'Content-Type: application/json' -d '{"email":"mealdb-check@example.test","password":"Str0ngPass!word","name":"Check"}' >/dev/null
TOKEN=$(curl -s -X POST localhost:3001/api/auth/login -H 'Content-Type: application/json' -d '{"email":"mealdb-check@example.test","password":"Str0ngPass!word"}' | sed -E 's/.*"token":"([^"]+)".*/\1/')
curl -s localhost:3001/api/recipes/categories -H "Authorization: Bearer $TOKEN" | head -c 300; echo
curl -s localhost:3001/api/recipes/cuisines -H "Authorization: Bearer $TOKEN" | head -c 300; echo
curl -s "localhost:3001/api/recipes?category=Seafood&cuisine=British" -H "Authorization: Bearer $TOKEN" | head -c 400; echo
curl -s "localhost:3001/api/recipes?q=chicken&limit=3" -H "Authorization: Bearer $TOKEN" | head -c 400; echo
curl -s localhost:3001/api/recipes/52772 -H "Authorization: Bearer $TOKEN" | head -c 500; echo
docker exec meal-planner-db psql -U postgres -d meal_planner -c "select key, expires_at from mealdb_cache order by key;"
```

Expected:
- The categories and cuisines lists come back non-empty.
- The Seafood + British results are non-empty, and each has `"category":"Seafood","cuisine":"British"`.
- The chicken search returns 3 results with a `total` above 3.
- Recipe 52772 is "Teriyaki Chicken Casserole", with ingredients and steps.
- The `mealdb_cache` query shows `categories`, `areas`, `filter:c=Seafood`, `filter:a=British`, `search:q=chicken` and `lookup:52772`.

If the auth response nests the token differently, adjust the `sed` expression and check the result with `echo $TOKEN`.

- [ ] **Step 3: Check stale-on-error.** Point TheMealDB at an unreachable host while the cache is warm, then force the entries to be expired:

```bash
docker exec meal-planner-db psql -U postgres -d meal_planner -c "update mealdb_cache set expires_at = now() - interval '1 hour';"
MEALDB_BASE_URL=http://127.0.0.1:9/api docker compose up -d backend
curl -s -o /dev/null -w '%{http_code}\n' localhost:3001/api/recipes/52772 -H "Authorization: Bearer $TOKEN"   # cached, expired, upstream down
curl -s -o /dev/null -w '%{http_code}\n' localhost:3001/api/recipes/52773 -H "Authorization: Bearer $TOKEN"   # never cached, upstream down
docker compose logs backend | grep -m1 "serving stale"
docker compose up -d backend   # restore the real base URL
```

Expected: `200` for 52772 (served stale), `503` for 52773, and a `serving stale "lookup:52772"` log line.

- [ ] **Step 4: Commit the docs, push, and open the PR.**

```bash
git add README.md
git commit -m "docs: document TheMealDB recipe endpoints and cache" -m "Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_014u1dJTmEC1ARhmephG3Zfk"
git push -u origin feat/themealdb-recipes
gh pr create --base main --title "feat: TheMealDB recipes behind /api/recipes with a Postgres read-through cache" --body-file <(cat <<'EOF'
Implements part 1 (backend) of `docs/superpowers/specs/2026-09-28-themealdb-integration-design.md`.

- `internal/mealdb`: TheMealDB client with response normalisation (ingredients, steps, tags; `meals: null`/string handled)
- `mealdb_cache` table + `CacheRepository`; `RecipeService` read-through cache with stale-on-error and 503 when nothing is cached
- `GET /api/recipes` (q/category/cuisine/ingredient, intersected, paged), `/api/recipes/:id`, `/categories`, `/cuisines` — all authenticated
- Config: `MEALDB_BASE_URL`, `MEALDB_TIMEOUT_SECONDS`, `MEALDB_DETAIL_TTL_HOURS`, `MEALDB_SEARCH_TTL_HOURS`

Verified: unit/service/handler tests, coverage gate, golangci-lint, and a manual run against real TheMealDB including stale-on-error.

🤖 Generated with [Claude Code](https://claude.com/claude-code)

https://claude.ai/code/session_014u1dJTmEC1ARhmephG3Zfk
EOF
)
```

Expected: the PR URL is printed. Wait for CI: Lint, Test (1.26/1.27), Security Scan and Build must pass.
