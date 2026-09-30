# USDA Nutrition (Backend) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Serve `GET /api/recipes/:id/nutrition` — whole-recipe nutrition estimated from TheMealDB ingredient lists via USDA FoodData Central, with per-ingredient traceability and honest coverage reporting.

**Architecture:** New units follow the existing client → service → handler → router pattern: an `internal/usda` FDC client, a pure measure parser in `internal/nutrition/measure`, an embedded overrides file in `internal/nutrition/overrides`, and a `NutritionService` in `internal/services` that reuses Part 1's `mealdb_cache` read-through cache (generalised out of `recipeService`). The handler maps service errors to 400/404/503 exactly like the recipe handler.

**Tech Stack:** Go 1.26, Gin, GORM (existing `mealdb_cache` table — no new tables), `golang.org/x/sync/errgroup`, `go:embed`, stdlib `net/http` + `httptest`.

**Spec:** `docs/superpowers/specs/2026-09-29-usda-nutrition-design.md` (approved). The frontend (§4 of the spec) is a separate plan in the frontend repo, written after this plan ships the endpoint.

## Global Constraints

- CI never calls FDC or TheMealDB: every test uses fakes, `httptest` servers or committed fixtures.
- The API key is sent only as the `api_key` query parameter and **never appears in logs or error strings**.
- JSON responses use camelCase; every error body is `{"error": "<message>"}`.
- Units in the response: `calories` kcal (integer), `sodium` mg (integer), all other nutrients grams to one decimal place.
- `reason` values are exactly `unmeasurable`, `noMatch`, `noPortion`; `status` values are exactly `counted`, `notCounted`.
- A nutrient FDC does not report is absent, never zero; unconvertible amounts are never guessed.
- A partial (failed-midway) estimate is never cached.
- Env defaults: `USDA_API_KEY=DEMO_KEY`, `USDA_BASE_URL=https://api.nal.usda.gov/fdc/v1`, `USDA_TIMEOUT_SECONDS=5`, `USDA_MATCH_TTL_HOURS=720`, `USDA_FOOD_TTL_HOURS=2160`, `NUTRITION_TTL_HOURS=168`.
- Part 1's tests keep passing **unchanged** through the cache refactor (Task 5).
- Coverage gate is 70% (`make ci-test`); run `make fmt`, `make vet` and `make lint` before each commit.
- Commit messages follow the repo's conventional style (`feat(nutrition): …`, `refactor: …`, `docs: …`) and end with `Co-Authored-By: Claude Fable 5 <noreply@anthropic.com>`.

## Review Focus

Spec-implied behaviors most likely to bite a user, each pinned to a test in the owning task:

1. **An FDC portion with `gramWeight: 0` or `amount: 0`** must be skipped, never divided by — otherwise totals become `Inf`/`NaN`. → Task 7 (`TestGramsForSkipsZeroWeightPortions`).
2. **A recipe listing the same ingredient twice** (TheMealDB does this) must count both lines in the totals while searching FDC once. → Task 8 (`TestEstimateCountsDuplicateIngredientLines`) and Task 9 (call-count assertion in `TestMatchCacheSharedAcrossCaseVariants`).
3. **A recipe with zero ingredients** must return 200 with `coverage {0, 0}` and make no FDC calls, not crash or 500. → Task 8 (`TestEstimateZeroIngredients`).
4. **A matched food reporting none of the seven nutrients** must still be `counted`, with every nutrient marked incomplete — not silently contribute zeros. → Task 8 (`TestEstimateAllNutrientsAbsent`).
5. **Ingredient-name case/spacing variants** (`Chicken Breasts` vs `chicken  breasts`) must share one match cache entry and one search call, or cache keys and rate limits blow up. → Task 9 (`TestMatchCacheSharedAcrossCaseVariants`).

---

### Task 1: Measure parser (`internal/nutrition/measure`)

**Files:**
- Create: `internal/nutrition/measure/measure.go`
- Test: `internal/nutrition/measure/measure_test.go`

**Interfaces:**
- Consumes: nothing (pure, no I/O).
- Produces:
  - `type Kind int` with constants `Unmeasurable`, `Mass`, `Volume`, `Count`.
  - `type Amount struct { Kind Kind; Value float64 }` — `Value` is grams for `Mass`, millilitres for `Volume`, item count for `Count`, 0 for `Unmeasurable`.
  - `func Parse(measure string) Amount`.

- [ ] **Step 1: Write the failing table test**

```go
package measure

import (
	"math"
	"testing"
)

func TestParse(t *testing.T) {
	cases := []struct {
		in   string
		kind Kind
		val  float64
	}{
		// mass, including attached units and package sizes
		{"1kg", Mass, 1000},
		{"200g", Mass, 200},
		{"1 (12 oz.)", Mass, 340.194},
		{"2 (16-oz.)", Mass, 0}, // see step 3: parens without a parseable "qty unit" fall through
		{"3 oz", Mass, 85.0485},
		{"1 lb", Mass, 453.592},
		{"2 lbs", Mass, 907.184},
		{"8 ounces", Mass, 226.796},
		// volume
		{"200ml", Volume, 200},
		{"1 l", Volume, 1000},
		{"3/4 cup", Volume, 177.441},
		{"¼ cup", Volume, 59.147},
		{"1 1/2 cups", Volume, 354.882},
		{"2 tbs", Volume, 29.5736},
		{"4 Tablespoons", Volume, 59.1472},
		{"1/2 teaspoon", Volume, 2.46446},
		{"2 fl oz", Volume, 59.147},
		{"1 pint", Volume, 473.176},
		// counts
		{"2", Count, 2},
		{"2 chopped", Count, 2},
		{"2 free-range", Count, 2},
		{"5 thinly sliced", Count, 5},
		{"1 whole", Count, 1},
		{"2 chicken breasts", Count, 2},
		{"Juice of 1", Count, 1},
		{"Zest of 1", Count, 1},
		{"2-3", Count, 2.5},
		{"1¼", Count, 1.25},
		{"1.2", Count, 1.2},
		// unmeasurable
		{"", Unmeasurable, 0},
		{"pinch", Unmeasurable, 0},
		{"dash", Unmeasurable, 0},
		{"to taste", Unmeasurable, 0},
		{"to serve", Unmeasurable, 0},
		{"to garnish", Unmeasurable, 0},
		{"for frying", Unmeasurable, 0},
		{"drizzle", Unmeasurable, 0},
		{"handful", Unmeasurable, 0},
	}
	for _, c := range cases {
		got := Parse(c.in)
		if got.Kind != c.kind || math.Abs(got.Value-c.val) > 0.01 {
			t.Errorf("Parse(%q) = {%v %v}, want {%v %v}", c.in, got.Kind, got.Value, c.kind, c.val)
		}
	}
}
```

Note on `{"2 (16-oz.)", Mass, 0}`: before running, decide the expected value: `16-oz` parses as a range token failing float parse on `16` vs `oz.` — the inner text `16-oz.` has no space, so after the digit-letter split normalization (step 3) it becomes `16- oz.` → quantity parse of `16-` fails → falls through parens handling → leading `2` → `Count 2`. **Change this row to `{"2 (16-oz.)", Count, 2}`** — a conservative, honest fallback (never guessed as mass).

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./internal/nutrition/measure/ -run TestParse -v`
Expected: FAIL — package does not compile / `Parse` undefined.

- [ ] **Step 3: Implement the parser**

```go
// Package measure parses TheMealDB's free-text ingredient measures ("3/4 cup",
// "1kg", "2 chicken breasts") into typed amounts. It is pure: no I/O.
package measure

import (
	"regexp"
	"strconv"
	"strings"
)

// Kind classifies a parsed measure.
type Kind int

// Kinds of amount. Unmeasurable covers "pinch", "to taste", empty text and
// anything else with no leading quantity.
const (
	Unmeasurable Kind = iota
	Mass              // Value is grams
	Volume            // Value is millilitres
	Count             // Value is a number of items
)

// Amount is a parsed measure. Value's unit depends on Kind.
type Amount struct {
	Kind  Kind
	Value float64
}

var massUnits = map[string]float64{
	"g": 1, "gram": 1, "grams": 1, "kg": 1000,
	"oz": 28.3495, "ounce": 28.3495, "ounces": 28.3495,
	"lb": 453.592, "lbs": 453.592, "pound": 453.592, "pounds": 453.592,
}

var volumeUnits = map[string]float64{
	"ml": 1, "l": 1000, "litre": 1000, "litres": 1000, "liter": 1000, "liters": 1000,
	"tsp": 4.92892, "teaspoon": 4.92892, "teaspoons": 4.92892,
	"tbs": 14.7868, "tbsp": 14.7868, "tablespoon": 14.7868, "tablespoons": 14.7868,
	"cup": 236.588, "cups": 236.588, "pint": 473.176, "pints": 473.176,
}

const flOzMl = 29.5735

var unicodeFractions = strings.NewReplacer(
	"¼", " 1/4", "½", " 1/2", "¾", " 3/4", "⅓", " 1/3", "⅔", " 2/3", "⅛", " 1/8",
)

// numUnitRe splits an attached unit off a number: "1kg" -> "1 kg".
var numUnitRe = regexp.MustCompile(`(\d)([a-z])`)

var parenRe = regexp.MustCompile(`\(([^)]*)\)`)

// Parse turns a free-text measure into an Amount. It never guesses: text it
// cannot read confidently is Unmeasurable (or a bare Count).
func Parse(measure string) Amount {
	s := strings.ToLower(strings.TrimSpace(measure))
	s = strings.TrimSpace(unicodeFractions.Replace(s))
	s = numUnitRe.ReplaceAllString(s, "$1 $2")
	if s == "" {
		return Amount{Kind: Unmeasurable}
	}
	if a, ok := parsePackage(s); ok {
		return a
	}
	if a, ok := parseOfCount(s); ok {
		return a
	}
	qty, rest, ok := parseQuantity(s)
	if !ok {
		return Amount{Kind: Unmeasurable}
	}
	if g, ok := unitGrams(rest); ok {
		return Amount{Kind: Mass, Value: qty * g}
	}
	if ml, ok := unitMl(rest); ok {
		return Amount{Kind: Volume, Value: qty * ml}
	}
	return Amount{Kind: Count, Value: qty}
}

// parsePackage handles a parenthesised package size: "1 (12 oz.)" is 12 oz
// per package times the leading count. Anything unreadable falls through.
func parsePackage(s string) (Amount, bool) {
	m := parenRe.FindStringSubmatchIndex(s)
	if m == nil {
		return Amount{}, false
	}
	qty, rest, ok := parseQuantity(strings.TrimSpace(s[m[2]:m[3]]))
	if !ok {
		return Amount{}, false
	}
	mult := 1.0
	if lead := strings.TrimSpace(s[:m[0]]); lead != "" {
		if q, _, ok := parseQuantity(lead); ok && q > 0 {
			mult = q
		}
	}
	if g, ok := unitGrams(rest); ok {
		return Amount{Kind: Mass, Value: mult * qty * g}, true
	}
	if ml, ok := unitMl(rest); ok {
		return Amount{Kind: Volume, Value: mult * qty * ml}, true
	}
	return Amount{}, false
}

// parseOfCount handles "juice of 1" and "zest of 1" as item counts.
func parseOfCount(s string) (Amount, bool) {
	for _, p := range []string{"juice of ", "zest of "} {
		if strings.HasPrefix(s, p) {
			if q, _, ok := parseQuantity(strings.TrimPrefix(s, p)); ok {
				return Amount{Kind: Count, Value: q}, true
			}
		}
	}
	return Amount{}, false
}

// parseQuantity reads a leading quantity (integer, decimal, fraction, mixed
// number or range) and returns it with the remaining text.
func parseQuantity(s string) (float64, string, bool) {
	fields := strings.Fields(s)
	if len(fields) == 0 {
		return 0, "", false
	}
	q, ok := parseNumber(fields[0])
	if !ok {
		return 0, "", false
	}
	rest := fields[1:]
	if len(rest) > 0 { // mixed number: "1 1/2"
		if f, isFrac := parseFraction(rest[0]); isFrac {
			q += f
			rest = rest[1:]
		}
	}
	return q, strings.Join(rest, " "), true
}

func parseNumber(tok string) (float64, bool) {
	if lo, hi, ok := splitRange(tok); ok {
		return (lo + hi) / 2, true
	}
	if f, ok := parseFraction(tok); ok {
		return f, true
	}
	f, err := strconv.ParseFloat(tok, 64)
	if err != nil || f < 0 {
		return 0, false
	}
	return f, true
}

func parseFraction(tok string) (float64, bool) {
	n, d, ok := strings.Cut(tok, "/")
	if !ok {
		return 0, false
	}
	nf, e1 := strconv.ParseFloat(n, 64)
	df, e2 := strconv.ParseFloat(d, 64)
	if e1 != nil || e2 != nil || df == 0 {
		return 0, false
	}
	return nf / df, true
}

// splitRange reads "2-3" as its midpoint.
func splitRange(tok string) (lo, hi float64, ok bool) {
	l, h, found := strings.Cut(tok, "-")
	if !found {
		return 0, 0, false
	}
	lf, e1 := strconv.ParseFloat(l, 64)
	hf, e2 := strconv.ParseFloat(h, 64)
	if e1 != nil || e2 != nil {
		return 0, 0, false
	}
	return lf, hf, true
}

func firstWord(s string) string {
	f := strings.Fields(s)
	if len(f) == 0 {
		return ""
	}
	return strings.TrimSuffix(f[0], ".")
}

func unitGrams(rest string) (float64, bool) {
	g, ok := massUnits[firstWord(rest)]
	return g, ok
}

func unitMl(rest string) (float64, bool) {
	f := strings.Fields(rest)
	if len(f) >= 2 && strings.TrimSuffix(f[0], ".") == "fl" && strings.TrimSuffix(f[1], ".") == "oz" {
		return flOzMl, true
	}
	ml, ok := volumeUnits[firstWord(rest)]
	return ml, ok
}
```

- [ ] **Step 4: Run the test to verify it passes**

Run: `go test ./internal/nutrition/measure/ -v`
Expected: PASS. If any row disagrees, fix the parser, not the expectation — except where a row's expectation was itself miscomputed (recompute by hand: e.g. `3/4 cup` = 0.75 × 236.588 = 177.441).

- [ ] **Step 5: Extend the table with the five sample meals' real measures**

Fetch each sample meal and read its `strMeasure1..20` values:

```bash
for id in 52772 52874 52959 52940 52795; do
  curl -s "https://www.themealdb.com/api/json/v1/1/lookup.php?i=$id" | python -c "import json,sys; m=json.load(sys.stdin)['meals'][0]; print(*[repr(m[f'strMeasure{i}']) for i in range(1,21) if (m[f'strMeasure{i}'] or '').strip()], sep='\n')"
done
```

Add every distinct measure printed to the test table with its correct expected `Kind`/`Value` (computed by hand from the unit tables above). This is a one-time fetch by the implementer; the committed test then runs offline. Run `go test ./internal/nutrition/measure/ -v` again; fix the parser for any measure it misreads (typical additions: more unit spellings). Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/nutrition/measure/
git commit -m "feat(nutrition): free-text measure parser"
```

---

### Task 2: USDA client — types and Search (`internal/usda`)

**Files:**
- Create: `internal/usda/types.go`, `internal/usda/client.go`
- Test: `internal/usda/client_test.go`

**Interfaces:**
- Consumes: nothing new.
- Produces (used by Tasks 3, 7, 8, 9):
  - `type Nutrient string` with constants `Calories`, `Protein`, `Carbohydrate`, `Fat`, `Fiber`, `Sugars`, `Sodium` whose values are the JSON names `"calories"`, `"protein"`, `"carbohydrate"`, `"fat"`, `"fiber"`, `"sugars"`, `"sodium"`.
  - `type Nutrients map[Nutrient]float64` — per 100 g; sodium in mg; an absent key means FDC did not report it.
  - `type Portion struct { Amount float64; Unit string; Modifier string; GramWeight float64 }` (JSON tags `amount`, `unit`, `modifier`, `gramWeight`).
  - `type SearchFood struct { FDCID int; Description string; DataType string }`.
  - `type Food struct { FDCID int; Description string; DataType string; Per100g Nutrients; Portions []Portion }` (JSON tags `fdcId`, `description`, `dataType`, `per100g`, `portions` — cached as JSON, so tags matter).
  - `type Client interface { Search(ctx context.Context, query string) ([]SearchFood, error); Foods(ctx context.Context, ids []int) ([]Food, error) }`.
  - `func NewHTTPClient(baseURL, apiKey string, timeout time.Duration) *HTTPClient`.

- [ ] **Step 1: Write the failing tests**

```go
package usda

import (
	"context"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"
)

func newTestClient(handler http.HandlerFunc) (*HTTPClient, *httptest.Server) {
	srv := httptest.NewServer(handler)
	return NewHTTPClient(srv.URL, "test-key", 2*time.Second), srv
}

func TestSearchSendsFiltersAndKey(t *testing.T) {
	var gotPath, gotQuery string
	c, srv := newTestClient(func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotQuery = r.URL.Path, r.URL.RawQuery
		_, _ = w.Write([]byte(`{"foods":[
			{"fdcId":174277,"description":"Soy sauce made from soy and wheat (shoyu)","dataType":"SR Legacy"},
			{"fdcId":999,"description":"Something branded","dataType":"SR Legacy"}]}`))
	})
	defer srv.Close()

	got, err := c.Search(context.Background(), "soy sauce")
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	want := []SearchFood{
		{FDCID: 174277, Description: "Soy sauce made from soy and wheat (shoyu)", DataType: "SR Legacy"},
		{FDCID: 999, Description: "Something branded", DataType: "SR Legacy"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Search = %+v, want %+v", got, want)
	}
	if gotPath != "/foods/search" {
		t.Errorf("path = %q", gotPath)
	}
	for _, part := range []string{"api_key=test-key", "dataType=Foundation%2CSR+Legacy", "pageSize=10", "query=soy+sauce"} {
		if !strings.Contains(gotQuery, part) {
			t.Errorf("query %q missing %q", gotQuery, part)
		}
	}
}

func TestSearchErrorsHideTheKey(t *testing.T) {
	c, srv := newTestClient(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	})
	defer srv.Close()

	_, err := c.Search(context.Background(), "salt")
	if err == nil {
		t.Fatal("want error on 429")
	}
	if !strings.Contains(err.Error(), "429") {
		t.Errorf("err = %v, want the status code", err)
	}
	if strings.Contains(err.Error(), "test-key") {
		t.Errorf("err = %v, must not contain the API key", err)
	}
}

func TestSearchMalformedJSON(t *testing.T) {
	c, srv := newTestClient(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{not json`))
	})
	defer srv.Close()
	if _, err := c.Search(context.Background(), "salt"); err == nil {
		t.Fatal("want error on malformed JSON")
	}
}

func TestSearchTimeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
	}))
	defer srv.Close()
	c := NewHTTPClient(srv.URL, "test-key", 20*time.Millisecond)
	if _, err := c.Search(context.Background(), "salt"); err == nil {
		t.Fatal("want error on timeout")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/usda/ -v`
Expected: FAIL — package does not exist.

- [ ] **Step 3: Implement types and Search**

`internal/usda/types.go`:

```go
// Package usda is a small client for USDA FoodData Central
// (https://fdc.nal.usda.gov/) that turns its responses into clean Go types.
package usda

import "context"

// Nutrient names a tracked nutrient; the string value is its JSON name in
// API responses.
type Nutrient string

// The seven tracked nutrients. Sodium is in mg; the rest are grams, except
// Calories (kcal). All are per 100 g of food.
const (
	Calories     Nutrient = "calories"
	Protein      Nutrient = "protein"
	Carbohydrate Nutrient = "carbohydrate"
	Fat          Nutrient = "fat"
	Fiber        Nutrient = "fiber"
	Sugars       Nutrient = "sugars"
	Sodium       Nutrient = "sodium"
)

// All lists every tracked nutrient, in display order.
var All = []Nutrient{Calories, Protein, Carbohydrate, Fat, Fiber, Sugars, Sodium}

// Nutrients holds per-100 g values. An absent key means FDC did not report
// that nutrient — absent, not zero.
type Nutrients map[Nutrient]float64

// Portion is one of FDC's household portions with its gram weight.
type Portion struct {
	Amount     float64 `json:"amount"`
	Unit       string  `json:"unit"`
	Modifier   string  `json:"modifier"`
	GramWeight float64 `json:"gramWeight"`
}

// SearchFood is one search result.
type SearchFood struct {
	FDCID       int    `json:"fdcId"`
	Description string `json:"description"`
	DataType    string `json:"dataType"`
}

// Food is a full food record. It is cached as JSON, so tags are stable.
type Food struct {
	FDCID       int       `json:"fdcId"`
	Description string    `json:"description"`
	DataType    string    `json:"dataType"`
	Per100g     Nutrients `json:"per100g"`
	Portions    []Portion `json:"portions"`
}

// Client fetches foods from FDC.
type Client interface {
	Search(ctx context.Context, query string) ([]SearchFood, error)
	Foods(ctx context.Context, ids []int) ([]Food, error)
}
```

`internal/usda/client.go`:

```go
package usda

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// maxBodyBytes caps how much of an FDC response is read.
const maxBodyBytes = 5 << 20

// HTTPClient implements Client over FDC's JSON API.
type HTTPClient struct {
	baseURL string
	apiKey  string
	http    *http.Client
}

// NewHTTPClient returns a client for baseURL (e.g.
// https://api.nal.usda.gov/fdc/v1). The key is sent as the api_key query
// parameter and never logged.
func NewHTTPClient(baseURL, apiKey string, timeout time.Duration) *HTTPClient {
	return &HTTPClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		apiKey:  apiKey,
		http:    &http.Client{Timeout: timeout},
	}
}

// do sends a request to path and returns the body of a 200 response. Errors
// name the path only, never the URL, so the key cannot leak into logs.
func (c *HTTPClient) do(ctx context.Context, method, path string, params url.Values, body []byte) ([]byte, error) {
	params.Set("api_key", c.apiKey)
	endpoint := c.baseURL + path + "?" + params.Encode()
	var reader io.Reader = http.NoBody
	if body != nil {
		reader = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, endpoint, reader)
	if err != nil {
		return nil, fmt.Errorf("usda: build request for %s: %w", path, err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("usda: %s: %w", path, sanitizeURLError(err, endpoint, c.baseURL+path))
	}
	defer func() {
		if cerr := resp.Body.Close(); cerr != nil {
			log.Printf("usda: close %s: %v", path, cerr)
		}
	}()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("usda: %s returned status %d", path, resp.StatusCode)
	}
	out, err := io.ReadAll(io.LimitReader(resp.Body, maxBodyBytes))
	if err != nil {
		return nil, fmt.Errorf("usda: read %s: %w", path, err)
	}
	return out, nil
}

// sanitizeURLError strips the query (which holds api_key) from transport
// errors, which embed the full URL.
func sanitizeURLError(err error, full, safe string) error {
	return fmt.Errorf("%s", strings.ReplaceAll(err.Error(), full, safe))
}

// Search returns generic foods matching query, best match first. Branded and
// survey foods are excluded: they are product- or survey-specific.
func (c *HTTPClient) Search(ctx context.Context, query string) ([]SearchFood, error) {
	params := url.Values{
		"query":    {query},
		"dataType": {"Foundation,SR Legacy"},
		"pageSize": {"10"},
	}
	body, err := c.do(ctx, http.MethodGet, "/foods/search", params, nil)
	if err != nil {
		return nil, err
	}
	var env struct {
		Foods []SearchFood `json:"foods"`
	}
	if err := json.Unmarshal(body, &env); err != nil {
		return nil, fmt.Errorf("usda: decode search: %w", err)
	}
	return env.Foods, nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/usda/ -v`
Expected: PASS (the `Foods` tests come in Task 3).

- [ ] **Step 5: Commit**

```bash
git add internal/usda/
git commit -m "feat(nutrition): USDA FDC client with search"
```

---

### Task 3: USDA client — Foods: batching, nutrients, portions

**Files:**
- Modify: `internal/usda/client.go`
- Test: `internal/usda/client_test.go`

**Interfaces:**
- Consumes: Task 2's types.
- Produces: `(*HTTPClient) Foods(ctx, ids []int) ([]Food, error)` — POSTs `{"fdcIds": [...], "format": "full"}` to `/foods` in batches of at most 20 ids; maps nutrient numbers 208 (falling back to 958, then 957), 203, 204, 205, 291, 269, 307 into `Per100g`; absent nutrients are absent keys.

- [ ] **Step 1: Write the failing tests**

Append to `internal/usda/client_test.go`:

```go
func TestFoodsMapsNutrientsAndPortions(t *testing.T) {
	c, srv := newTestClient(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/foods" {
			t.Errorf("got %s %s", r.Method, r.URL.Path)
		}
		var req struct {
			FDCIDs []int  `json:"fdcIds"`
			Format string `json:"format"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Format != "full" {
			t.Errorf("bad body: %+v, %v", req, err)
		}
		_, _ = w.Write([]byte(`[{
			"fdcId":174277,"description":"Soy sauce (shoyu)","dataType":"SR Legacy",
			"foodNutrients":[
				{"nutrient":{"number":"208"},"amount":53.0},
				{"nutrient":{"number":"203"},"amount":8.14},
				{"nutrient":{"number":"204"},"amount":0.57},
				{"nutrient":{"number":"205"},"amount":4.93},
				{"nutrient":{"number":"307"},"amount":5493.0},
				{"nutrient":{"number":"999"},"amount":1.0},
				{"nutrient":{"number":"291"}}
			],
			"foodPortions":[
				{"amount":1,"gramWeight":16.0,"modifier":"","measureUnit":{"name":"tablespoon","abbreviation":"tbsp"}},
				{"amount":1,"gramWeight":255.0,"modifier":"cup","measureUnit":{"name":"undetermined","abbreviation":"undetermined"}}
			]}]`))
	})
	defer srv.Close()

	foods, err := c.Foods(context.Background(), []int{174277})
	if err != nil {
		t.Fatalf("Foods: %v", err)
	}
	f := foods[0]
	want := Nutrients{Calories: 53.0, Protein: 8.14, Fat: 0.57, Carbohydrate: 4.93, Sodium: 5493.0}
	if !reflect.DeepEqual(f.Per100g, want) {
		t.Errorf("Per100g = %v, want %v (fiber had no amount: absent; 999: ignored)", f.Per100g, want)
	}
	wantPortions := []Portion{
		{Amount: 1, Unit: "tbsp", Modifier: "", GramWeight: 16},
		{Amount: 1, Unit: "undetermined", Modifier: "cup", GramWeight: 255},
	}
	if !reflect.DeepEqual(f.Portions, wantPortions) {
		t.Errorf("Portions = %+v, want %+v", f.Portions, wantPortions)
	}
}

func TestFoodsEnergyFallback(t *testing.T) {
	cases := []struct {
		name, nutrients string
		want            Nutrients
	}{
		{"958 preferred over 957",
			`[{"nutrient":{"number":"957"},"amount":100.0},{"nutrient":{"number":"958"},"amount":90.0}]`,
			Nutrients{Calories: 90}},
		{"957 alone",
			`[{"nutrient":{"number":"957"},"amount":100.0}]`,
			Nutrients{Calories: 100}},
		{"208 wins over both",
			`[{"nutrient":{"number":"208"},"amount":80.0},{"nutrient":{"number":"958"},"amount":90.0}]`,
			Nutrients{Calories: 80}},
		{"none: calories absent", `[]`, Nutrients{}},
	}
	for _, tc := range cases {
		c, srv := newTestClient(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(`[{"fdcId":1,"description":"x","dataType":"Foundation","foodNutrients":` + tc.nutrients + `,"foodPortions":[]}]`))
		})
		foods, err := c.Foods(context.Background(), []int{1})
		srv.Close()
		if err != nil || !reflect.DeepEqual(foods[0].Per100g, tc.want) {
			t.Errorf("%s: Per100g = %v, %v; want %v", tc.name, foods[0].Per100g, err, tc.want)
		}
	}
}

func TestFoodsBatchesOver20IDs(t *testing.T) {
	var batches [][]int
	c, srv := newTestClient(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			FDCIDs []int `json:"fdcIds"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		batches = append(batches, req.FDCIDs)
		_, _ = w.Write([]byte(`[]`))
	})
	defer srv.Close()

	ids := make([]int, 25)
	for i := range ids {
		ids[i] = i + 1
	}
	if _, err := c.Foods(context.Background(), ids); err != nil {
		t.Fatalf("Foods: %v", err)
	}
	if len(batches) != 2 || len(batches[0]) != 20 || len(batches[1]) != 5 {
		t.Errorf("batches = %v, want sizes [20 5]", batches)
	}
}

func TestFoodsUpstreamError(t *testing.T) {
	c, srv := newTestClient(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})
	defer srv.Close()
	if _, err := c.Foods(context.Background(), []int{1}); err == nil {
		t.Fatal("want error on 500")
	}
}
```

Add `"encoding/json"` to the test imports.

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/usda/ -run TestFoods -v`
Expected: FAIL — `c.Foods` undefined.

- [ ] **Step 3: Implement Foods**

Append to `internal/usda/client.go`:

```go
// foodsBatchSize is FDC's maximum ids per /foods request.
const foodsBatchSize = 20

// nutrientNumbers maps FDC nutrient numbers to tracked nutrients. Energy has
// Atwater fallbacks handled separately in toFood.
var nutrientNumbers = map[string]Nutrient{
	"203": Protein, "204": Fat, "205": Carbohydrate,
	"291": Fiber, "269": Sugars, "307": Sodium,
}

type rawFood struct {
	FDCID         int    `json:"fdcId"`
	Description   string `json:"description"`
	DataType      string `json:"dataType"`
	FoodNutrients []struct {
		Nutrient struct {
			Number string `json:"number"`
		} `json:"nutrient"`
		Amount *float64 `json:"amount"`
	} `json:"foodNutrients"`
	FoodPortions []struct {
		Amount      float64 `json:"amount"`
		GramWeight  float64 `json:"gramWeight"`
		Modifier    string  `json:"modifier"`
		MeasureUnit struct {
			Name         string `json:"name"`
			Abbreviation string `json:"abbreviation"`
		} `json:"measureUnit"`
	} `json:"foodPortions"`
}

// Foods returns full records for ids, splitting requests into batches of 20.
func (c *HTTPClient) Foods(ctx context.Context, ids []int) ([]Food, error) {
	out := make([]Food, 0, len(ids))
	for start := 0; start < len(ids); start += foodsBatchSize {
		end := min(start+foodsBatchSize, len(ids))
		body, err := json.Marshal(map[string]any{"fdcIds": ids[start:end], "format": "full"})
		if err != nil {
			return nil, fmt.Errorf("usda: encode foods request: %w", err)
		}
		resp, err := c.do(ctx, http.MethodPost, "/foods", url.Values{}, body)
		if err != nil {
			return nil, err
		}
		var raw []rawFood
		if err := json.Unmarshal(resp, &raw); err != nil {
			return nil, fmt.Errorf("usda: decode foods: %w", err)
		}
		for _, r := range raw {
			out = append(out, toFood(r))
		}
	}
	return out, nil
}

// toFood maps a raw record: nutrients by number (energy 208 falling back to
// 958 then 957) into Per100g, portions with the unit abbreviation or name.
func toFood(r rawFood) Food {
	per := Nutrients{}
	var atwaterSpecific, atwaterGeneral *float64
	for _, fn := range r.FoodNutrients {
		if fn.Amount == nil {
			continue
		}
		switch fn.Nutrient.Number {
		case "208":
			per[Calories] = *fn.Amount
		case "958":
			atwaterSpecific = fn.Amount
		case "957":
			atwaterGeneral = fn.Amount
		default:
			if n, ok := nutrientNumbers[fn.Nutrient.Number]; ok {
				per[n] = *fn.Amount
			}
		}
	}
	if _, ok := per[Calories]; !ok {
		if atwaterSpecific != nil {
			per[Calories] = *atwaterSpecific
		} else if atwaterGeneral != nil {
			per[Calories] = *atwaterGeneral
		}
	}
	portions := make([]Portion, 0, len(r.FoodPortions))
	for _, p := range r.FoodPortions {
		unit := p.MeasureUnit.Abbreviation
		if unit == "" {
			unit = p.MeasureUnit.Name
		}
		portions = append(portions, Portion{
			Amount: p.Amount, Unit: unit, Modifier: p.Modifier, GramWeight: p.GramWeight,
		})
	}
	return Food{
		FDCID: r.FDCID, Description: r.Description, DataType: r.DataType,
		Per100g: per, Portions: portions,
	}
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/usda/ -v`
Expected: PASS, all tests.

- [ ] **Step 5: Commit**

```bash
git add internal/usda/
git commit -m "feat(nutrition): FDC Foods with batching and nutrient mapping"
```

---

### Task 4: Overrides (`internal/nutrition/overrides`)

**Files:**
- Create: `internal/nutrition/overrides/overrides.go`, `internal/nutrition/overrides/overrides.json`
- Test: `internal/nutrition/overrides/overrides_test.go`

**Interfaces:**
- Consumes: nothing new.
- Produces (used by Tasks 8, 9, 11):
  - `type Override struct { FDCID int; ItemGrams float64; Note string }` (JSON tags `fdcId`, `itemGrams`, `note`).
  - `func Load() (*Set, error)` — parses the embedded `overrides.json`.
  - `func Parse(data []byte) (*Set, error)` — same, for tests.
  - `(*Set) Get(name string) (Override, bool)` — name is normalised internally.
  - `(*Set) Version() string` — 12-hex-char content hash; the matcher version in cache keys.
  - `func Normalize(name string) string` — lower-case, `_` as space, collapsed single spaces.

- [ ] **Step 1: Write the failing tests**

```go
package overrides

import (
	"strings"
	"testing"
)

func TestParseAndGet(t *testing.T) {
	set, err := Parse([]byte(`{
		"Chicken_Breasts": { "fdcId": 171077, "itemGrams": 174 },
		"salt":            { "fdcId": 173468 },
		"egg  yolks":      { "itemGrams": 17 }
	}`))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if o, ok := set.Get("chicken breasts"); !ok || o.FDCID != 171077 || o.ItemGrams != 174 {
		t.Errorf("Get(chicken breasts) = %+v, %v", o, ok)
	}
	if o, ok := set.Get("Chicken  Breasts"); !ok || o.FDCID != 171077 {
		t.Errorf("lookup must normalise; got %+v, %v", o, ok)
	}
	if o, ok := set.Get("egg yolks"); !ok || o.FDCID != 0 || o.ItemGrams != 17 {
		t.Errorf("itemGrams-only entry: %+v, %v", o, ok)
	}
	if _, ok := set.Get("butter"); ok {
		t.Error("Get(butter) must miss")
	}
}

func TestParseRejectsBadEntries(t *testing.T) {
	cases := map[string]string{
		"invalid JSON":       `{`,
		"negative fdcId":     `{"x": {"fdcId": -1}}`,
		"negative itemGrams": `{"x": {"fdcId": 1, "itemGrams": -2}}`,
		"empty entry":        `{"x": {}}`,
	}
	for name, data := range cases {
		if _, err := Parse([]byte(data)); err == nil {
			t.Errorf("%s: want error", name)
		}
	}
}

func TestVersionTracksContent(t *testing.T) {
	a, _ := Parse([]byte(`{"salt": {"fdcId": 1}}`))
	b, _ := Parse([]byte(`{"salt": {"fdcId": 2}}`))
	if a.Version() == b.Version() {
		t.Error("different content must give different versions")
	}
	if len(a.Version()) != 12 {
		t.Errorf("version %q: want 12 hex chars", a.Version())
	}
}

func TestLoadEmbeddedFile(t *testing.T) {
	if _, err := Load(); err != nil {
		t.Fatalf("embedded overrides.json must be valid: %v", err)
	}
}

func TestNormalize(t *testing.T) {
	if got := Normalize(" Chicken_ Breasts "); got != "chicken breasts" {
		t.Errorf("Normalize = %q", got)
	}
	_ = strings.TrimSpace("") // keep strings import if unused elsewhere
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/nutrition/overrides/ -v`
Expected: FAIL — package missing.

- [ ] **Step 3: Implement, with an empty initial file**

`internal/nutrition/overrides/overrides.json` starts as `{}` (Task 13 populates it with FDC-verified entries — ids are never committed unverified):

```json
{}
```

`internal/nutrition/overrides/overrides.go`:

```go
// Package overrides holds hand-checked corrections to automatic FDC
// matching: a pinned fdcId for known-bad matches and per-item gram weights
// for count measures FDC's portions do not cover.
package overrides

import (
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
)

//go:embed overrides.json
var embedded []byte

// Override corrects one ingredient. Either field may appear alone: FDCID
// pins the food (skipping search); ItemGrams is the weight of one item.
type Override struct {
	FDCID     int     `json:"fdcId,omitempty"`
	ItemGrams float64 `json:"itemGrams,omitempty"`
	Note      string  `json:"note,omitempty"`
}

// Set is a parsed, validated overrides file.
type Set struct {
	entries map[string]Override
	version string
}

// Load parses the embedded overrides.json. An invalid file is a startup
// error: the server must not run with silently-dropped overrides.
func Load() (*Set, error) {
	return Parse(embedded)
}

// Parse validates data and builds a Set. The version is a hash of the raw
// bytes, so any edit retires every cached result built from the old file.
func Parse(data []byte) (*Set, error) {
	var raw map[string]Override
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("overrides: invalid JSON: %w", err)
	}
	entries := make(map[string]Override, len(raw))
	for name, o := range raw {
		if o.FDCID < 0 {
			return nil, fmt.Errorf("overrides: %q: fdcId must be positive", name)
		}
		if o.ItemGrams < 0 {
			return nil, fmt.Errorf("overrides: %q: itemGrams must be positive", name)
		}
		if o.FDCID == 0 && o.ItemGrams == 0 {
			return nil, fmt.Errorf("overrides: %q: set fdcId and/or itemGrams", name)
		}
		entries[Normalize(name)] = o
	}
	sum := sha256.Sum256(data)
	return &Set{entries: entries, version: hex.EncodeToString(sum[:])[:12]}, nil
}

// Get returns the override for an ingredient name, normalising it first.
func (s *Set) Get(name string) (Override, bool) {
	o, ok := s.entries[Normalize(name)]
	return o, ok
}

// Version identifies the file's content; it is part of nutrition cache keys.
func (s *Set) Version() string {
	return s.version
}

// Normalize lower-cases name, treats "_" as a space and collapses runs of
// spaces — the same rule TheMealDB filter values use.
func Normalize(name string) string {
	return strings.Join(strings.Fields(strings.ReplaceAll(strings.ToLower(name), "_", " ")), " ")
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/nutrition/overrides/ -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/nutrition/overrides/
git commit -m "feat(nutrition): embedded overrides file with validation and content version"
```

---

### Task 5: Generalise the read-through cache helper

**Files:**
- Modify: `internal/services/recipe_service.go` (the `cached` function and `recipeService` struct)
- Modify: `internal/services/recipe_search.go` (call sites of `cached`)

**Interfaces:**
- Consumes: existing `repository.CacheRepository`, `models.CachedResponse`.
- Produces (used by Tasks 8, 9):
  - `type readCache struct { name string; repo repository.CacheRepository; now func() time.Time }` — `name` prefixes log lines (recipes keep `"recipe cache"`, so Part 1's log-assertion test passes unchanged).
  - `func newReadCache(name string, repo repository.CacheRepository, now func() time.Time) *readCache`
  - `func cached[T any](ctx context.Context, c *readCache, key string, ttl time.Duration, fetch func(context.Context) (T, error)) (T, error)` — same behavior as today; pass-through (no wrap, no stale) for `mealdb.ErrNotFound` **and** `ErrRecipeNotFound`.

This is a pure refactor: **no test file changes**. Part 1's tests are the safety net.

- [ ] **Step 1: Run the existing suite for a green baseline**

Run: `go test ./internal/services/ ./internal/handlers/`
Expected: PASS.

- [ ] **Step 2: Refactor**

In `internal/services/recipe_service.go`, replace the `recipeService` struct fields and `cached`:

```go
// readCache is a read-through cache over the mealdb_cache table, shared by
// the recipe and nutrition services. name prefixes its log lines.
type readCache struct {
	name string
	repo repository.CacheRepository
	now  func() time.Time
}

func newReadCache(name string, repo repository.CacheRepository, now func() time.Time) *readCache {
	return &readCache{name: name, repo: repo, now: now}
}

type recipeService struct {
	client mealdb.Client
	rc     *readCache
	ttl    RecipeCacheTTL
}

// NewRecipeService creates a RecipeService. now is injectable for tests; pass time.Now.
func NewRecipeService(client mealdb.Client, cache repository.CacheRepository, ttl RecipeCacheTTL, now func() time.Time) RecipeService {
	return &recipeService{client: client, rc: newReadCache("recipe cache", cache, now), ttl: ttl}
}

// passThrough reports whether err must reach the caller unchanged: not
// wrapped as unavailable, and never answered with a stale entry.
func passThrough(err error) bool {
	return errors.Is(err, mealdb.ErrNotFound) || errors.Is(err, ErrRecipeNotFound)
}

// cached returns the value for key: a fresh cache entry if there is one,
// otherwise the upstream result (which is then stored). If the upstream
// fails, an expired entry is served instead; with no entry at all the error
// is ErrUpstreamUnavailable. Pass-through errors are returned unchanged.
func cached[T any](ctx context.Context, c *readCache, key string, ttl time.Duration, fetch func(context.Context) (T, error)) (T, error) {
	var zero T
	entry, err := c.repo.Get(key)
	if err != nil {
		log.Printf("%s: read %q: %v", c.name, key, err)
		entry = nil
	}
	var stale *T
	if entry != nil {
		var v T
		if jsonErr := json.Unmarshal([]byte(entry.Payload), &v); jsonErr == nil {
			if c.now().Before(entry.ExpiresAt) {
				return v, nil
			}
			stale = &v
		} else {
			log.Printf("%s: decode %q: %v", c.name, key, jsonErr)
		}
	}

	v, err := fetch(ctx)
	if err != nil {
		if passThrough(err) {
			return zero, err
		}
		if stale != nil {
			log.Printf("%s: serving stale %q after upstream error: %v", c.name, key, err)
			return *stale, nil
		}
		return zero, fmt.Errorf("%w: %w", ErrUpstreamUnavailable, err)
	}

	if payload, mErr := json.Marshal(v); mErr == nil {
		now := c.now()
		if upErr := c.repo.Upsert(&models.CachedResponse{
			Key: key, Payload: string(payload),
			FetchedAt: now, ExpiresAt: now.Add(ttl),
		}); upErr != nil {
			log.Printf("%s: write %q: %v", c.name, key, upErr)
		}
	}
	return v, nil
}
```

Update every call site from `cached(ctx, s, …)` to `cached(ctx, s.rc, …)` — in `recipe_service.go` (`Categories`, `Cuisines`, `Get`) and `recipe_search.go` (`searchByName`, `filter`, `canonicalCuisine`). No other changes.

Note: the double `ErrUpstreamUnavailable` wrap when a nutrition fetch closure itself calls `RecipeService.Get` is harmless — `errors.Is` still matches — but pass-through of `ErrRecipeNotFound` is required so a missing meal is never cached or served stale as "nutrition".

- [ ] **Step 3: Run the full suite — unchanged tests must pass**

Run: `go test ./...`
Expected: PASS with **zero test-file edits** (`git status` shows only the two source files modified).

- [ ] **Step 4: Commit**

```bash
git add internal/services/recipe_service.go internal/services/recipe_search.go
git commit -m "refactor: extract shared read-through cache from recipeService"
```

---

### Task 6: USDA configuration

**Files:**
- Modify: `internal/config/config.go`
- Test: `internal/config/config_test.go` (append)

**Interfaces:**
- Produces (used by Task 11): `Config` fields `USDAAPIKey string`, `USDABaseURL string`, `USDATimeout time.Duration`, `USDAMatchTTL time.Duration`, `USDAFoodTTL time.Duration`, `NutritionTTL time.Duration`.

- [ ] **Step 1: Write the failing test**

Append to `internal/config/config_test.go` (follow the file's existing pattern of setting env vars with `t.Setenv` and calling `Load()`):

```go
func TestUSDADefaults(t *testing.T) {
	cfg := Load()
	if cfg.USDAAPIKey != "DEMO_KEY" {
		t.Errorf("USDAAPIKey = %q", cfg.USDAAPIKey)
	}
	if cfg.USDABaseURL != "https://api.nal.usda.gov/fdc/v1" {
		t.Errorf("USDABaseURL = %q", cfg.USDABaseURL)
	}
	if cfg.USDATimeout != 5*time.Second {
		t.Errorf("USDATimeout = %v", cfg.USDATimeout)
	}
	if cfg.USDAMatchTTL != 720*time.Hour || cfg.USDAFoodTTL != 2160*time.Hour || cfg.NutritionTTL != 168*time.Hour {
		t.Errorf("TTLs = %v %v %v", cfg.USDAMatchTTL, cfg.USDAFoodTTL, cfg.NutritionTTL)
	}
}

func TestUSDAOverrides(t *testing.T) {
	t.Setenv("USDA_API_KEY", "real-key")
	t.Setenv("USDA_TIMEOUT_SECONDS", "9")
	t.Setenv("NUTRITION_TTL_HOURS", "1")
	cfg := Load()
	if cfg.USDAAPIKey != "real-key" || cfg.USDATimeout != 9*time.Second || cfg.NutritionTTL != time.Hour {
		t.Errorf("overrides not applied: %+v", cfg)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/config/ -run TestUSDA -v`
Expected: FAIL — fields undefined.

- [ ] **Step 3: Implement**

Add to the `Config` struct:

```go
	// USDA FoodData Central nutrition source
	USDAAPIKey   string
	USDABaseURL  string
	USDATimeout  time.Duration
	USDAMatchTTL time.Duration // ingredient-name -> FDC id decisions
	USDAFoodTTL  time.Duration // per-food nutrients and portions
	NutritionTTL time.Duration // finished per-recipe estimates
```

Add to `Load()`:

```go
		// USDA FoodData Central
		USDAAPIKey:   getEnv("USDA_API_KEY", "DEMO_KEY"),
		USDABaseURL:  getEnv("USDA_BASE_URL", "https://api.nal.usda.gov/fdc/v1"),
		USDATimeout:  time.Duration(getEnvAsPositiveInt("USDA_TIMEOUT_SECONDS", 5)) * time.Second,
		USDAMatchTTL: time.Duration(getEnvAsPositiveInt("USDA_MATCH_TTL_HOURS", 720)) * time.Hour,
		USDAFoodTTL:  time.Duration(getEnvAsPositiveInt("USDA_FOOD_TTL_HOURS", 2160)) * time.Hour,
		NutritionTTL: time.Duration(getEnvAsPositiveInt("NUTRITION_TTL_HOURS", 168)) * time.Hour,
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/config/ -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/config/
git commit -m "feat(nutrition): USDA configuration with env defaults"
```

---

### Task 7: Ranking and conversion helpers (`internal/services/nutrition_convert.go`)

**Files:**
- Create: `internal/services/nutrition_convert.go`
- Test: `internal/services/nutrition_convert_test.go`

**Interfaces:**
- Consumes: `usda.SearchFood`, `usda.Portion` (Task 2), `measure.Amount` (Task 1), `overrides.Override` (Task 4).
- Produces (used by Task 8):
  - `func pickMatch(name string, foods []usda.SearchFood) int` — name is already normalised lower-case; returns 0 for no results.
  - `func gramsFor(amt measure.Amount, name string, o overrides.Override, portions []usda.Portion) (float64, bool)`.
  - `func portionMl(p usda.Portion) float64` — total millilitres of the portion, 0 when it is not a volume.
  - `func round1(x float64) float64`.

- [ ] **Step 1: Write the failing tests**

```go
package services

import (
	"testing"

	"github.com/meal-planner/backend/internal/nutrition/measure"
	"github.com/meal-planner/backend/internal/nutrition/overrides"
	"github.com/meal-planner/backend/internal/usda"
)

func TestPickMatchRanking(t *testing.T) {
	foods := []usda.SearchFood{
		{FDCID: 1, Description: "Soup, chicken noodle, canned"},
		{FDCID: 2, Description: "Chicken breast, raw"},
		{FDCID: 3, Description: "Chicken breasts, oven-roasted"},
	}
	// 1st rule: first comma segment equals the name (singular or plural).
	if got := pickMatch("chicken breasts", foods); got != 2 {
		t.Errorf("segment match (plural tolerant) = %d, want 2", got)
	}
	// 2nd rule: contains "raw".
	if got := pickMatch("poultry", foods); got != 2 {
		t.Errorf("raw preference = %d, want 2", got)
	}
	// 3rd rule: FDC's own order.
	cooked := []usda.SearchFood{{FDCID: 7, Description: "Soup, canned"}, {FDCID: 8, Description: "Broth, cubes"}}
	if got := pickMatch("stock", cooked); got != 7 {
		t.Errorf("fallback order = %d, want 7", got)
	}
	if got := pickMatch("anything", nil); got != 0 {
		t.Errorf("no results = %d, want 0", got)
	}
}

func TestGramsForMass(t *testing.T) {
	g, ok := gramsFor(measure.Amount{Kind: measure.Mass, Value: 340}, "beef", overrides.Override{}, nil)
	if !ok || g != 340 {
		t.Errorf("mass = %v, %v", g, ok)
	}
}

func TestGramsForVolumeUsesAnyVolumePortion(t *testing.T) {
	// Only a tbsp portion exists (1 tbsp = 16 g); 177.44 ml (3/4 cup) of it:
	// 177.44 * 16 / 14.7868 = 192.0 g.
	portions := []usda.Portion{
		{Amount: 1, Unit: "slice", Modifier: "", GramWeight: 28},
		{Amount: 1, Unit: "tbsp", Modifier: "", GramWeight: 16},
	}
	g, ok := gramsFor(measure.Amount{Kind: measure.Volume, Value: 177.44}, "soy sauce", overrides.Override{}, portions)
	if !ok || g < 191 || g > 193 {
		t.Errorf("volume = %v, %v; want about 192", g, ok)
	}
	// No volume portion at all -> not counted.
	if _, ok := gramsFor(measure.Amount{Kind: measure.Volume, Value: 100}, "x", overrides.Override{},
		[]usda.Portion{{Amount: 1, Unit: "slice", GramWeight: 28}}); ok {
		t.Error("no volume portion must fail")
	}
}

func TestGramsForCount(t *testing.T) {
	// Override itemGrams wins.
	g, ok := gramsFor(measure.Amount{Kind: measure.Count, Value: 2}, "chicken breasts",
		overrides.Override{ItemGrams: 174}, nil)
	if !ok || g != 348 {
		t.Errorf("override count = %v, %v", g, ok)
	}
	// Portion whose text names one item: unit/modifier containing whole,
	// large, medium, fruit, unit, or the name's own last word.
	portions := []usda.Portion{
		{Amount: 1, Unit: "cup", Modifier: "chopped", GramWeight: 140},
		{Amount: 1, Unit: "undetermined", Modifier: "breast, bone removed", GramWeight: 172},
	}
	g, ok = gramsFor(measure.Amount{Kind: measure.Count, Value: 2}, "chicken breasts", overrides.Override{}, portions)
	if !ok || g != 344 {
		t.Errorf("last-word portion = %v, %v; want 344", g, ok)
	}
	g, ok = gramsFor(measure.Amount{Kind: measure.Count, Value: 3},
		"lemon", overrides.Override{}, []usda.Portion{{Amount: 1, Unit: "fruit", GramWeight: 58}})
	if !ok || g != 174 {
		t.Errorf("fruit portion = %v, %v", g, ok)
	}
	// Nothing usable -> not counted.
	if _, ok := gramsFor(measure.Amount{Kind: measure.Count, Value: 1}, "saffron", overrides.Override{},
		[]usda.Portion{{Amount: 1, Unit: "cup", GramWeight: 100}}); ok {
		t.Error("count with no item portion must fail")
	}
}

func TestGramsForSkipsZeroWeightPortions(t *testing.T) {
	// gramWeight or amount of 0 must be skipped, never divided by.
	bad := []usda.Portion{
		{Amount: 0, Unit: "cup", GramWeight: 0},
		{Amount: 1, Unit: "cup", GramWeight: 0},
	}
	if _, ok := gramsFor(measure.Amount{Kind: measure.Volume, Value: 100}, "x", overrides.Override{}, bad); ok {
		t.Error("zero-weight volume portions must be unusable")
	}
	if _, ok := gramsFor(measure.Amount{Kind: measure.Count, Value: 1}, "lemon", overrides.Override{},
		[]usda.Portion{{Amount: 0, Unit: "fruit", GramWeight: 0}}); ok {
		t.Error("zero-weight count portions must be unusable")
	}
}

func TestGramsForUnmeasurable(t *testing.T) {
	if _, ok := gramsFor(measure.Amount{Kind: measure.Unmeasurable}, "salt", overrides.Override{}, nil); ok {
		t.Error("unmeasurable must never convert")
	}
}

func TestRound1(t *testing.T) {
	if round1(191.2599) != 191.3 || round1(2.04) != 2.0 {
		t.Error("round1 must round to one decimal")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/services/ -run "TestPickMatch|TestGramsFor|TestRound1" -v`
Expected: FAIL — functions undefined.

- [ ] **Step 3: Implement**

`internal/services/nutrition_convert.go`:

```go
package services

import (
	"math"
	"strings"

	"github.com/meal-planner/backend/internal/nutrition/measure"
	"github.com/meal-planner/backend/internal/nutrition/overrides"
	"github.com/meal-planner/backend/internal/usda"
)

// Millilitres per unit, for converting FDC volume portions:
// 1 cup = 16 tbsp = 48 tsp = 236.6 ml.
const (
	mlPerCup  = 236.588
	mlPerTbsp = 14.7868
	mlPerTsp  = 4.92892
	mlPerFlOz = 29.5735
)

// pickMatch chooses the best FDC search result for a normalised ingredient
// name: first a description whose first comma segment matches the name
// (singular or plural), then one containing "raw", then FDC's own order.
// 0 means no match.
func pickMatch(name string, foods []usda.SearchFood) int {
	if len(foods) == 0 {
		return 0
	}
	for _, f := range foods {
		seg := strings.ToLower(strings.TrimSpace(strings.SplitN(f.Description, ",", 2)[0]))
		if seg == name || seg == name+"s" || seg+"s" == name {
			return f.FDCID
		}
	}
	for _, f := range foods {
		if strings.Contains(strings.ToLower(f.Description), "raw") {
			return f.FDCID
		}
	}
	return foods[0].FDCID
}

// gramsFor converts a parsed amount of the named ingredient into grams using
// the override and the matched food's portions. ok is false when the amount
// cannot be converted honestly (reason noPortion for Volume/Count).
func gramsFor(amt measure.Amount, name string, o overrides.Override, portions []usda.Portion) (float64, bool) {
	switch amt.Kind {
	case measure.Mass:
		return amt.Value, true
	case measure.Volume:
		for _, p := range portions {
			if ml := portionMl(p); ml > 0 && p.GramWeight > 0 {
				return amt.Value * p.GramWeight / ml, true
			}
		}
		return 0, false
	case measure.Count:
		if o.ItemGrams > 0 {
			return amt.Value * o.ItemGrams, true
		}
		words := []string{"whole", "large", "medium", "fruit", "unit", lastWord(name)}
		for _, p := range portions {
			if p.Amount <= 0 || p.GramWeight <= 0 {
				continue
			}
			desc := strings.ToLower(p.Unit + " " + p.Modifier)
			for _, w := range words {
				if w != "" && strings.Contains(desc, w) {
					return amt.Value * p.GramWeight / p.Amount, true
				}
			}
		}
		return 0, false
	default:
		return 0, false
	}
}

// portionMl returns the portion's total volume in millilitres, or 0 when its
// text names no known volume unit or its amount is not positive.
func portionMl(p usda.Portion) float64 {
	if p.Amount <= 0 {
		return 0
	}
	desc := strings.ToLower(p.Unit + " " + p.Modifier)
	var unit float64
	switch {
	case strings.Contains(desc, "cup"):
		unit = mlPerCup
	case strings.Contains(desc, "tbsp"), strings.Contains(desc, "tablespoon"):
		unit = mlPerTbsp
	case strings.Contains(desc, "tsp"), strings.Contains(desc, "teaspoon"):
		unit = mlPerTsp
	case strings.Contains(desc, "fl oz"), strings.Contains(desc, "fluid ounce"):
		unit = mlPerFlOz
	case strings.Contains(desc, "milliliter"), strings.Contains(desc, "millilitre"), desc == "ml " || strings.HasPrefix(desc, "ml "):
		unit = 1
	default:
		return 0
	}
	return unit * p.Amount
}

// lastWord returns the singular-ish last word of a normalised name
// ("chicken breasts" -> "breast"), used to spot per-item portions.
func lastWord(name string) string {
	fields := strings.Fields(name)
	if len(fields) == 0 {
		return ""
	}
	return strings.TrimSuffix(fields[len(fields)-1], "s")
}

// round1 rounds to one decimal place, the response's precision for grams.
func round1(x float64) float64 {
	return math.Round(x*10) / 10
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/services/ -run "TestPickMatch|TestGramsFor|TestRound1" -v`
Expected: PASS. Then run `go test ./internal/services/` to confirm nothing else broke.

- [ ] **Step 5: Commit**

```bash
git add internal/services/nutrition_convert.go internal/services/nutrition_convert_test.go
git commit -m "feat(nutrition): FDC match ranking and gram conversion"
```

---

### Task 8: NutritionService — estimation core

**Files:**
- Create: `internal/services/nutrition_service.go`, `internal/testutil/usda_client.go`
- Test: `internal/services/nutrition_service_test.go`

**Interfaces:**
- Consumes: `RecipeService.Get`, `usda.Client`, `measure.Parse`, `overrides.Set`, Task 7's helpers, `readCache` (constructed; caching behavior lands in Task 9).
- Produces (used by Tasks 9–11):
  - `type NutritionTTL struct { Match, Food, Result time.Duration }`
  - `type NutritionService interface { Estimate(ctx context.Context, mealID string) (*RecipeNutrition, error) }`
  - `func NewNutritionService(recipes RecipeService, client usda.Client, cache repository.CacheRepository, ov *overrides.Set, ttl NutritionTTL, now func() time.Time) NutritionService`
  - Response types (JSON contract §3 of the spec):

```go
// NutritionTotals sums every counted ingredient. Calories are kcal and
// Sodium mg (integers); the rest are grams to one decimal place.
type NutritionTotals struct {
	Calories     int     `json:"calories"`
	Protein      float64 `json:"protein"`
	Carbohydrate float64 `json:"carbohydrate"`
	Fat          float64 `json:"fat"`
	Fiber        float64 `json:"fiber"`
	Sugars       float64 `json:"sugars"`
	Sodium       int     `json:"sodium"`
}

// NutritionFoodRef points a counted ingredient at its FDC food.
type NutritionFoodRef struct {
	FDCID       int    `json:"fdcId"`
	Description string `json:"description"`
}

// IngredientNutrition traces one ingredient line: counted with its grams,
// food and calories, or notCounted with a reason.
type IngredientNutrition struct {
	Name     string            `json:"name"`
	Measure  string            `json:"measure"`
	Status   string            `json:"status"` // "counted" or "notCounted"
	Grams    float64           `json:"grams,omitempty"`
	Food     *NutritionFoodRef `json:"food,omitempty"`
	Calories int               `json:"calories,omitempty"`
	Reason   string            `json:"reason,omitempty"` // "unmeasurable", "noMatch", "noPortion"
}

// NutritionCoverage says how many ingredient lines were counted.
type NutritionCoverage struct {
	Counted int `json:"counted"`
	Total   int `json:"total"`
}

// RecipeNutrition is the whole-recipe estimate returned by the API.
type RecipeNutrition struct {
	RecipeID    string                `json:"recipeId"`
	Source      string                `json:"source"`
	Totals      NutritionTotals       `json:"totals"`
	Incomplete  []string              `json:"incomplete,omitempty"`
	Coverage    NutritionCoverage     `json:"coverage"`
	Ingredients []IngredientNutrition `json:"ingredients"`
}
```

  - Constants: `statusCounted = "counted"`, `statusNotCounted = "notCounted"`, `reasonUnmeasurable = "unmeasurable"`, `reasonNoMatch = "noMatch"`, `reasonNoPortion = "noPortion"`, `nutritionSource = "USDA FoodData Central"`.
  - `testutil.USDAClient` fake:

```go
// Package testutil: USDAClient is a fake usda.Client backed by maps. Safe
// for concurrent use; set fields before the calls under test start.
type USDAClient struct {
	SearchResults map[string][]usda.SearchFood // keyed by query
	FoodsByID     map[int]usda.Food
	// Err, when set, is returned by every method; SearchErr/FoodsErr by one.
	Err, SearchErr, FoodsErr error
	// SearchCalls / FoodsCalls count calls; MaxConcurrentSearches records the
	// highest number of Search calls in flight at once.
	SearchCalls, FoodsCalls int
	MaxConcurrentSearches   int
	// mu guards everything; inFlight tracks concurrent Search calls.
	mu       sync.Mutex
	inFlight int
}
```

`Search` increments `inFlight` under the lock, updates `MaxConcurrentSearches`, unlocks, sleeps 2 ms (so overlap is observable), relocks, decrements, then returns `SearchResults[query]` (empty slice when absent) or the error. `Foods` returns `FoodsByID[id]` for each requested id that exists (missing ids are simply omitted, as FDC does) or the error.

- [ ] **Step 1: Write the testutil fake**

Write `internal/testutil/usda_client.go` exactly per the interface block above, mirroring the style of `testutil/mealdb_client.go` (constructor `NewUSDAClient()` initialising the maps).

- [ ] **Step 2: Write the failing service tests**

`internal/services/nutrition_service_test.go`:

```go
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
	if got.Totals.Calories != 102 || got.Totals.Sodium != 10546 {
		t.Errorf("totals = %+v (sodium: 192.0/100*5493 = 10546.6 -> 10547? compute exactly in step 4)", got.Totals)
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
	// Each line: 14.7868 ml * 16/14.7868 = 16 g -> 16/100*53 = 8.48 -> 8 kcal; total 17 (16.96 rounds once, at the total).
	if got.Totals.Calories != 17 {
		t.Errorf("calories = %d, want 17 (totals rounded once, from unrounded sums)", got.Totals.Calories)
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
```

- [ ] **Step 3: Run tests to verify they fail**

Run: `go test ./internal/services/ -run TestEstimate -v`
Expected: FAIL — `NutritionService` undefined.

- [ ] **Step 4: Implement the service**

`internal/services/nutrition_service.go` (this task implements the full flow **without** per-key caching or the concurrency limit — direct client calls; Task 9 adds them):

```go
package services

import (
	"context"
	"math"
	"sort"
	"time"

	"github.com/meal-planner/backend/internal/mealdb"
	"github.com/meal-planner/backend/internal/nutrition/measure"
	"github.com/meal-planner/backend/internal/nutrition/overrides"
	"github.com/meal-planner/backend/internal/repository"
	"github.com/meal-planner/backend/internal/usda"
)

// nutritionSource labels every estimate with its data source.
const nutritionSource = "USDA FoodData Central"

// Ingredient statuses and not-counted reasons in the API contract.
const (
	statusCounted      = "counted"
	statusNotCounted   = "notCounted"
	reasonUnmeasurable = "unmeasurable"
	reasonNoMatch      = "noMatch"
	reasonNoPortion    = "noPortion"
)

// NutritionTTL sets how long each cached layer stays fresh.
type NutritionTTL struct {
	Match  time.Duration // ingredient-name -> FDC id decisions
	Food   time.Duration // per-food nutrients and portions
	Result time.Duration // finished per-recipe estimates
}

// [Response types exactly as in this task's Interfaces block: NutritionTotals,
// NutritionFoodRef, IngredientNutrition, NutritionCoverage, RecipeNutrition.]

// NutritionService estimates a recipe's nutrition from USDA data.
type NutritionService interface {
	Estimate(ctx context.Context, mealID string) (*RecipeNutrition, error)
}

type nutritionService struct {
	recipes RecipeService
	client  usda.Client
	rc      *readCache
	ov      *overrides.Set
	ttl     NutritionTTL
}

// NewNutritionService creates a NutritionService sharing the mealdb_cache
// table. now is injectable for tests; pass time.Now.
func NewNutritionService(recipes RecipeService, client usda.Client, cache repository.CacheRepository,
	ov *overrides.Set, ttl NutritionTTL, now func() time.Time) NutritionService {
	return &nutritionService{
		recipes: recipes, client: client,
		rc: newReadCache("nutrition cache", cache, now), ov: ov, ttl: ttl,
	}
}

func (s *nutritionService) Estimate(ctx context.Context, mealID string) (*RecipeNutrition, error) {
	v, err := s.estimate(ctx, mealID) // Task 9 wraps this in the result cache
	if err != nil {
		return nil, err
	}
	return v, nil
}

// estimate builds a fresh estimate: load the meal, match measurable
// ingredients to FDC foods, fetch those foods, then assemble the totals.
func (s *nutritionService) estimate(ctx context.Context, mealID string) (*RecipeNutrition, error) {
	meal, err := s.recipes.Get(ctx, mealID)
	if err != nil {
		return nil, err
	}
	matches, err := s.matchAll(ctx, meal.Ingredients)
	if err != nil {
		return nil, err
	}
	foods, err := s.fetchFoods(ctx, matchedIDs(matches))
	if err != nil {
		return nil, err
	}
	out := assemble(meal, matches, foods, s.ov)
	return &out, nil
}

// matchAll resolves each distinct measurable ingredient name to an FDC id
// (0 = no match). Overrides with a pinned fdcId skip the search entirely.
func (s *nutritionService) matchAll(ctx context.Context, ingredients []mealdb.Ingredient) (map[string]int, error) {
	names := distinctMeasurableNames(ingredients)
	out := make(map[string]int, len(names))
	for _, name := range names {
		if o, ok := s.ov.Get(name); ok && o.FDCID > 0 {
			out[name] = o.FDCID
			continue
		}
		results, err := s.client.Search(ctx, name)
		if err != nil {
			return nil, wrapUnavailable(err)
		}
		out[name] = pickMatch(name, results)
	}
	return out, nil
}

// distinctMeasurableNames returns the normalised names of ingredients whose
// measures parse to something countable, first occurrence order, deduped.
func distinctMeasurableNames(ingredients []mealdb.Ingredient) []string {
	seen := map[string]bool{}
	var names []string
	for _, ing := range ingredients {
		if measure.Parse(ing.Measure).Kind == measure.Unmeasurable {
			continue
		}
		n := overrides.Normalize(ing.Name)
		if n == "" || seen[n] {
			continue
		}
		seen[n] = true
		names = append(names, n)
	}
	return names
}

func matchedIDs(matches map[string]int) []int {
	seen := map[int]bool{}
	var ids []int
	for _, id := range matches {
		if id > 0 && !seen[id] {
			seen[id] = true
			ids = append(ids, id)
		}
	}
	sort.Ints(ids) // deterministic batches and cache order
	return ids
}

// fetchFoods loads the needed foods (Task 9 adds the per-food cache).
func (s *nutritionService) fetchFoods(ctx context.Context, ids []int) (map[int]usda.Food, error) {
	out := make(map[int]usda.Food, len(ids))
	if len(ids) == 0 {
		return out, nil
	}
	foods, err := s.client.Foods(ctx, ids)
	if err != nil {
		return nil, wrapUnavailable(err)
	}
	for _, f := range foods {
		out[f.FDCID] = f
	}
	return out, nil
}

// wrapUnavailable marks an FDC failure the way handlers expect.
func wrapUnavailable(err error) error {
	return fmt.Errorf("%w: %w", ErrUpstreamUnavailable, err)
}

// assemble walks the ingredient lines in recipe order, converting and
// summing every counted one. A nutrient some counted ingredient lacks makes
// the total incomplete, never zero-padded. Totals are rounded once, here.
func assemble(meal *mealdb.Meal, matches map[string]int, foods map[int]usda.Food, ov *overrides.Set) RecipeNutrition {
	res := RecipeNutrition{
		RecipeID: meal.ID, Source: nutritionSource,
		Coverage:    NutritionCoverage{Total: len(meal.Ingredients)},
		Ingredients: make([]IngredientNutrition, 0, len(meal.Ingredients)),
	}
	sums := usda.Nutrients{}
	incomplete := map[usda.Nutrient]bool{}

	for _, ing := range meal.Ingredients {
		line := IngredientNutrition{Name: ing.Name, Measure: ing.Measure, Status: statusNotCounted}
		norm := overrides.Normalize(ing.Name)
		amt := measure.Parse(ing.Measure)
		o, _ := ov.Get(norm)

		switch food, ok := foods[matches[norm]]; {
		case amt.Kind == measure.Unmeasurable:
			line.Reason = reasonUnmeasurable
		case !ok:
			line.Reason = reasonNoMatch
		default:
			grams, convertible := gramsFor(amt, norm, o, food.Portions)
			if !convertible {
				line.Reason = reasonNoPortion
				break
			}
			line.Status = statusCounted
			line.Grams = round1(grams)
			line.Food = &NutritionFoodRef{FDCID: food.FDCID, Description: food.Description}
			if kcal, has := food.Per100g[usda.Calories]; has {
				line.Calories = int(math.Round(grams / 100 * kcal))
			}
			res.Coverage.Counted++
			for _, n := range usda.All {
				v, has := food.Per100g[n]
				if !has {
					incomplete[n] = true
					continue
				}
				sums[n] += grams / 100 * v
			}
		}
		res.Ingredients = append(res.Ingredients, line)
	}

	res.Totals = NutritionTotals{
		Calories:     int(math.Round(sums[usda.Calories])),
		Protein:      round1(sums[usda.Protein]),
		Carbohydrate: round1(sums[usda.Carbohydrate]),
		Fat:          round1(sums[usda.Fat]),
		Fiber:        round1(sums[usda.Fiber]),
		Sugars:       round1(sums[usda.Sugars]),
		Sodium:       int(math.Round(sums[usda.Sodium])),
	}
	for n := range incomplete {
		res.Incomplete = append(res.Incomplete, string(n))
	}
	sort.Strings(res.Incomplete)
	return res
}
```

Add `"fmt"` to the imports. Write out the five response types verbatim from the Interfaces block (the `[...]` marker above is plan shorthand, not code).

- [ ] **Step 5: Run tests, fixing exact expected numbers**

Run: `go test ./internal/services/ -run TestEstimate -v`
Where a test's expected number was left approximate (the sodium note in `TestEstimateCountsAndTotals`), compute the exact value the implementation produces by hand (192.0378 g × 54.93 mg/g = 10548.7… — verify against the actual grams value 177.441 × 16 ÷ 14.7868 = 192.0378; sodium = 192.0378/100 × 5493 = 10548.6 → 10549) and pin the test to that hand-computed value. Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/services/nutrition_service.go internal/services/nutrition_service_test.go internal/testutil/usda_client.go
git commit -m "feat(nutrition): NutritionService estimation core"
```

---

### Task 9: NutritionService — caching and concurrency

**Files:**
- Modify: `internal/services/nutrition_service.go`
- Test: `internal/services/nutrition_service_test.go` (append)

**Interfaces:**
- Consumes: `cached[T]` and `readCache` (Task 5).
- Produces: final cache behavior —
  - result key `nutrition:<version>:<mealId>` (TTL `ttl.Result`), match key `usda:match:<version>:<name>` (TTL `ttl.Match`, negative results cached too), food key `usda:food:<fdcId>` (TTL `ttl.Food`, stale usable on upstream error);
  - searches limited to 4 in flight via `errgroup.SetLimit`;
  - `type matchResult struct { FDCID int ` + "`json:\"fdcId\"`" + ` }` (0 records "no match").

- [ ] **Step 1: Write the failing tests**

Append to `internal/services/nutrition_service_test.go`:

```go
func TestEstimateResultCachedAndVersioned(t *testing.T) {
	f := newNutritionFixture(t, `{}`)
	f.seedSoySauce()
	f.seedMeal("52772", mealdb.Ingredient{Name: "soy sauce", Measure: "1 tbs"})

	ctx := context.Background()
	if _, err := f.svc.Estimate(ctx, "52772"); err != nil {
		t.Fatalf("first: %v", err)
	}
	searches, foods := f.client.SearchCalls, f.client.FoodsCalls
	if _, err := f.svc.Estimate(ctx, "52772"); err != nil {
		t.Fatalf("second: %v", err)
	}
	if f.client.SearchCalls != searches || f.client.FoodsCalls != foods {
		t.Error("repeat view must make no FDC calls")
	}
	if _, ok := f.cache.Entries["nutrition:"+f.ov.Version()+":52772"]; !ok {
		t.Errorf("result cache key missing; have %v", keys(f.cache.Entries))
	}

	// A different overrides version misses the old result cache.
	ov2, _ := overrides.Parse([]byte(`{"salt": {"fdcId": 173468}}`))
	svc2 := NewNutritionService(f.recipes.svc, f.client, f.cache, ov2,
		NutritionTTL{Match: time.Hour, Food: time.Hour, Result: time.Hour},
		func() time.Time { return f.now })
	if _, err := svc2.Estimate(ctx, "52772"); err != nil {
		t.Fatalf("versioned: %v", err)
	}
	if f.client.SearchCalls == searches {
		t.Error("new matcher version must refetch, not reuse the old cache")
	}
}

func keys(m map[string]*models.CachedResponse) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

func TestMatchCacheSharedAcrossCaseVariants(t *testing.T) {
	f := newNutritionFixture(t, `{}`)
	f.seedSoySauce()
	f.seedMeal("7", mealdb.Ingredient{Name: "Soy_Sauce", Measure: "1 tbs"})
	f.seedMeal("8", mealdb.Ingredient{Name: "soy  sauce", Measure: "2 tbs"})

	ctx := context.Background()
	if _, err := f.svc.Estimate(ctx, "7"); err != nil {
		t.Fatal(err)
	}
	if _, err := f.svc.Estimate(ctx, "8"); err != nil {
		t.Fatal(err)
	}
	if f.client.SearchCalls != 1 {
		t.Errorf("SearchCalls = %d, want 1: variants share usda:match key", f.client.SearchCalls)
	}
}

func TestNoMatchIsCachedToo(t *testing.T) {
	f := newNutritionFixture(t, `{}`)
	f.client.SearchResults["unicorn"] = nil
	f.seedMeal("9", mealdb.Ingredient{Name: "unicorn", Measure: "1"})
	f.seedMeal("10", mealdb.Ingredient{Name: "unicorn", Measure: "2"})
	ctx := context.Background()
	_, _ = f.svc.Estimate(ctx, "9")
	_, _ = f.svc.Estimate(ctx, "10")
	if f.client.SearchCalls != 1 {
		t.Errorf("SearchCalls = %d, want 1: a no-match decision is cached", f.client.SearchCalls)
	}
}

func TestStaleResultServedOnUpstreamError(t *testing.T) {
	f := newNutritionFixture(t, `{}`)
	f.seedSoySauce()
	f.seedMeal("52772", mealdb.Ingredient{Name: "soy sauce", Measure: "1 tbs"})
	ctx := context.Background()
	if _, err := f.svc.Estimate(ctx, "52772"); err != nil {
		t.Fatal(err)
	}

	f.now = f.now.Add(30 * 24 * time.Hour) // result expired
	f.client.Err = errors.New("fdc down")
	got, err := f.svc.Estimate(ctx, "52772")
	if err != nil || got.Coverage.Counted != 1 {
		t.Fatalf("stale result must be served: %+v, %v", got, err)
	}
}

func TestPartialFailureCachesNothing(t *testing.T) {
	f := newNutritionFixture(t, `{}`)
	f.seedSoySauce()
	f.seedMeal("11", mealdb.Ingredient{Name: "soy sauce", Measure: "1 tbs"})
	f.client.FoodsErr = errors.New("fdc down mid-flight") // search succeeds, foods fails
	if _, err := f.svc.Estimate(context.Background(), "11"); !errors.Is(err, ErrUpstreamUnavailable) {
		t.Fatalf("err = %v", err)
	}
	if _, ok := f.cache.Entries["nutrition:"+f.ov.Version()+":11"]; ok {
		t.Error("a partial result must never be cached")
	}
}

func TestSearchConcurrencyLimitedToFour(t *testing.T) {
	f := newNutritionFixture(t, `{}`)
	ings := make([]mealdb.Ingredient, 10)
	for i := range ings {
		name := "ingredient" + string(rune('a'+i))
		ings[i] = mealdb.Ingredient{Name: name, Measure: "100g"}
		f.client.SearchResults[name] = []usda.SearchFood{{FDCID: 100 + i, Description: name + ", raw"}}
		f.client.FoodsByID[100+i] = usda.Food{FDCID: 100 + i, Description: name, Per100g: usda.Nutrients{usda.Calories: 1}}
	}
	f.seedMeal("12", ings...)
	if _, err := f.svc.Estimate(context.Background(), "12"); err != nil {
		t.Fatal(err)
	}
	if f.client.MaxConcurrentSearches > 4 {
		t.Errorf("MaxConcurrentSearches = %d, want <= 4", f.client.MaxConcurrentSearches)
	}
	if f.client.SearchCalls != 10 {
		t.Errorf("SearchCalls = %d, want 10", f.client.SearchCalls)
	}
}
```

Add `"github.com/meal-planner/backend/internal/models"` to the test imports.

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/services/ -run "TestEstimateResultCached|TestMatchCache|TestNoMatch|TestStaleResult|TestPartialFailure|TestSearchConcurrency" -v`
Expected: FAIL — repeat views re-call the client; no cache keys written.

- [ ] **Step 3: Implement caching and concurrency**

In `internal/services/nutrition_service.go`:

Replace `Estimate` to route through the result cache:

```go
func (s *nutritionService) Estimate(ctx context.Context, mealID string) (*RecipeNutrition, error) {
	key := "nutrition:" + s.ov.Version() + ":" + mealID
	v, err := cached(ctx, s.rc, key, s.ttl.Result, func(ctx context.Context) (RecipeNutrition, error) {
		out, err := s.estimate(ctx, mealID)
		if err != nil {
			return RecipeNutrition{}, err
		}
		return *out, nil
	})
	if err != nil {
		return nil, err
	}
	return &v, nil
}
```

Replace `matchAll` with the concurrent, cached version:

```go
// maxConcurrentSearches caps in-flight FDC search calls per estimate.
const maxConcurrentSearches = 4

// matchResult is a cached matching decision; FDCID 0 records "no match" so
// hopeless ingredients do not re-hit FDC for every recipe that uses them.
type matchResult struct {
	FDCID int `json:"fdcId"`
}

func (s *nutritionService) matchAll(ctx context.Context, ingredients []mealdb.Ingredient) (map[string]int, error) {
	names := distinctMeasurableNames(ingredients)
	results := make([]int, len(names))
	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(maxConcurrentSearches)
	for i, name := range names {
		g.Go(func() error {
			if o, ok := s.ov.Get(name); ok && o.FDCID > 0 {
				results[i] = o.FDCID
				return nil
			}
			key := "usda:match:" + s.ov.Version() + ":" + name
			m, err := cached(gctx, s.rc, key, s.ttl.Match, func(ctx context.Context) (matchResult, error) {
				found, err := s.client.Search(ctx, name)
				if err != nil {
					return matchResult{}, err
				}
				return matchResult{FDCID: pickMatch(name, found)}, nil
			})
			if err != nil {
				return err //nolint:wrapcheck // already an ErrUpstreamUnavailable from cached
			}
			results[i] = m.FDCID
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return nil, err //nolint:wrapcheck // already wrapped
	}
	out := make(map[string]int, len(names))
	for i, name := range names {
		out[name] = results[i]
	}
	return out, nil
}
```

Replace `fetchFoods` with the per-food cache plus batched fetch and stale fallback:

```go
func foodKey(id int) string { return "usda:food:" + strconv.Itoa(id) }

// fetchFoods returns the foods for ids: fresh cache entries first, then one
// batched Foods call for the rest. On an upstream error, expired entries
// stand in where they exist; an id with nothing at all fails the estimate.
func (s *nutritionService) fetchFoods(ctx context.Context, ids []int) (map[int]usda.Food, error) {
	out := make(map[int]usda.Food, len(ids))
	stale := map[int]usda.Food{}
	var missing []int
	for _, id := range ids {
		entry, err := s.rc.repo.Get(foodKey(id))
		if err != nil {
			log.Printf("nutrition cache: read %q: %v", foodKey(id), err)
			entry = nil
		}
		if entry != nil {
			var f usda.Food
			if jsonErr := json.Unmarshal([]byte(entry.Payload), &f); jsonErr == nil {
				if s.rc.now().Before(entry.ExpiresAt) {
					out[id] = f
					continue
				}
				stale[id] = f
			}
		}
		missing = append(missing, id)
	}
	if len(missing) == 0 {
		return out, nil
	}

	fetched, err := s.client.Foods(ctx, missing)
	if err != nil {
		for _, id := range missing {
			f, ok := stale[id]
			if !ok {
				return nil, wrapUnavailable(err)
			}
			out[id] = f
		}
		log.Printf("nutrition cache: serving stale foods after upstream error: %v", err)
		return out, nil
	}
	now := s.rc.now()
	for _, f := range fetched {
		out[f.FDCID] = f
		if payload, mErr := json.Marshal(f); mErr == nil {
			if upErr := s.rc.repo.Upsert(&models.CachedResponse{
				Key: foodKey(f.FDCID), Payload: string(payload),
				FetchedAt: now, ExpiresAt: now.Add(s.ttl.Food),
			}); upErr != nil {
				log.Printf("nutrition cache: write %q: %v", foodKey(f.FDCID), upErr)
			}
		}
	}
	return out, nil
}
```

Add imports: `"encoding/json"`, `"log"`, `"strconv"`, `"golang.org/x/sync/errgroup"`, `"github.com/meal-planner/backend/internal/models"`.

Design note: an id FDC no longer returns is simply absent from `out`; `assemble` then reports that ingredient `noMatch`. That is honest (the match is dead) and cannot poison the totals.

- [ ] **Step 4: Run the whole services suite**

Run: `go test ./internal/services/ -v`
Expected: PASS, including every Task 8 test unchanged.

- [ ] **Step 5: Run the race detector over the concurrent path**

Run: `go test ./internal/services/ -race -run "TestSearchConcurrency|TestEstimate"`
Expected: PASS, no data races.

- [ ] **Step 6: Commit**

```bash
git add internal/services/nutrition_service.go internal/services/nutrition_service_test.go
git commit -m "feat(nutrition): read-through caching and bounded concurrency"
```

---

### Task 10: Golden test

**Files:**
- Create: `internal/services/testdata/nutrition_golden_meal.json`, `internal/services/testdata/nutrition_golden_foods.json`
- Test: `internal/services/nutrition_golden_test.go`

**Interfaces:**
- Consumes: everything from Tasks 8–9.
- Produces: a regression net over the whole pipeline's arithmetic.

- [ ] **Step 1: Build the fixtures**

`testdata/nutrition_golden_meal.json` — a realistic teriyaki-style meal (this is the shape of `mealdb.Meal`):

```json
{
  "id": "52772",
  "name": "Teriyaki Chicken Casserole",
  "category": "Chicken",
  "cuisine": "Japanese",
  "thumbnail": "",
  "ingredients": [
    { "name": "soy sauce", "measure": "3/4 cup" },
    { "name": "water", "measure": "1/2 cup" },
    { "name": "brown sugar", "measure": "1/4 cup" },
    { "name": "ground ginger", "measure": "1/2 teaspoon" },
    { "name": "minced garlic", "measure": "1/2 teaspoon" },
    { "name": "honey", "measure": "4 Tablespoons" },
    { "name": "chicken breasts", "measure": "2" },
    { "name": "stir-fry vegetables", "measure": "1 (12 oz.)" },
    { "name": "brown rice", "measure": "3 cups" },
    { "name": "salt", "measure": "pinch" }
  ],
  "instructions": ["Cook."],
  "tags": []
}
```

`testdata/nutrition_golden_foods.json` — a map of search query → search results plus a list of `usda.Food` records with realistic per-100 g values and portions. Give: soy sauce (tbsp portion, 16 g), water (cup portion, 237 g, all nutrients 0 present), brown sugar (cup 220 g), ground ginger (tsp 1.8 g), minced garlic (tsp 2.8 g), honey (tbsp 21 g), chicken breast (portion `"breast, bone and skin removed"` 172 g), brown rice (cup 195 g — note: no `raw` in description). Omit `sugars` from the chicken record so `incomplete` is exercised. Give `stir-fry vegetables` an empty search result (`noMatch`). Shape:

```json
{
  "searches": {
    "soy sauce": [{ "fdcId": 174277, "description": "Soy sauce made from soy and wheat (shoyu)", "dataType": "SR Legacy" }],
    "stir-fry vegetables": []
  },
  "foods": [
    { "fdcId": 174277, "description": "Soy sauce made from soy and wheat (shoyu)", "dataType": "SR Legacy",
      "per100g": { "calories": 53, "protein": 8.14, "carbohydrate": 4.93, "fat": 0.57, "fiber": 0.8, "sugars": 0.4, "sodium": 5493 },
      "portions": [{ "amount": 1, "unit": "tbsp", "modifier": "", "gramWeight": 16 }] }
  ]
}
```

(Complete both files for all eight foods; the values need to be plausible, internally consistent and committed — the test's authority is "the maths never drifts", not "FDC agrees".)

- [ ] **Step 2: Write the golden test**

```go
package services

import (
	"context"
	"encoding/json"
	"math"
	"os"
	"testing"
	"time"

	"github.com/meal-planner/backend/internal/mealdb"
	"github.com/meal-planner/backend/internal/nutrition/overrides"
	"github.com/meal-planner/backend/internal/testutil"
	"github.com/meal-planner/backend/internal/usda"
)

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

	// Recorded expectations. On first run, compute each by hand from the
	// fixtures (grams x per100g/100, summed, rounded once) and pin the
	// exact values here; the 1% tolerance then catches arithmetic drift
	// without being brittle to float noise.
	const wantCalories = 0 // PIN ME on first run
	if wantCalories == 0 {
		t.Logf("PIN: totals=%+v coverage=%+v incomplete=%v", got.Totals, got.Coverage, got.Incomplete)
		t.Fatal("pin the golden expectations (see comment)")
	}
	within1pc := func(got, want float64) bool {
		return want == 0 && got == 0 || math.Abs(got-want) <= 0.01*math.Abs(want)
	}
	if !within1pc(float64(got.Totals.Calories), wantCalories) {
		t.Errorf("calories = %d, want about %v", got.Totals.Calories, wantCalories)
	}
	// ... same pinned checks for protein/carbohydrate/fat/fiber/sugars/sodium,
	// coverage {8, 10}, incomplete == ["sugars"], and the two notCounted
	// reasons (salt: unmeasurable, stir-fry vegetables: noMatch).
	_ = time.Now
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
```

- [ ] **Step 3: Run once, pin, run again**

Run: `go test ./internal/services/ -run TestGoldenEstimate -v` — it fails with the `PIN` log. Verify the logged totals by hand against the fixtures (spot-check at least calories and sodium end-to-end on paper). Replace the placeholder with the verified numbers and the remaining pinned checks (protein, carbohydrate, fat, fiber, sugars, sodium as `float64` constants; coverage; incomplete; the two reasons). Re-run.
Expected: PASS.

- [ ] **Step 4: Commit**

```bash
git add internal/services/nutrition_golden_test.go internal/services/testdata/
git commit -m "test(nutrition): golden estimate over recorded fixtures"
```

---

### Task 11: Handler, router and main wiring

**Files:**
- Create: `internal/handlers/nutrition_handler.go`
- Modify: `internal/handlers/recipe_handler.go` (one new message constant), `internal/router/router.go`, `cmd/server/main.go`
- Modify (call sites only): `internal/router/router_test.go`, `internal/handlers/helpers_test.go`
- Test: `internal/handlers/nutrition_handler_test.go`

**Interfaces:**
- Consumes: `services.NutritionService` (Task 8), `overrides.Load` (Task 4), config (Task 6).
- Produces:
  - `GET /api/recipes/:id/nutrition` (protected).
  - `handlers.NewNutritionHandler(svc services.NutritionService) *NutritionHandler` with method `Get(c *gin.Context)`.
  - `router.New(userRepo repository.UserRepository, recipes services.RecipeService, nutrition services.NutritionService, cfg *config.Config) *gin.Engine` — **signature change**; `router.Setup(db *gorm.DB, cfg *config.Config, ov *overrides.Set) *gin.Engine` — **signature change**.
  - Message constant `msgNutritionDown = "nutrition is temporarily unavailable, please try again shortly"`.

- [ ] **Step 1: Write the failing handler tests**

`internal/handlers/nutrition_handler_test.go`, in the existing `handlers_test` style (reuse `newTestServer` after step 3 extends it):

```go
package handlers_test

import (
	"net/http"
	"testing"

	"github.com/meal-planner/backend/internal/mealdb"
	"github.com/meal-planner/backend/internal/usda"
)

func TestNutritionEndpoint(t *testing.T) {
	ts := newTestServer(t)
	token := ts.registerAndLogin(t) // reuse the file's existing auth helper name; check helpers_test.go for the exact one

	ts.mealdb.Meals["52772"] = mealdb.Meal{
		ID: "52772", Name: "Teriyaki",
		Ingredients:  []mealdb.Ingredient{{Name: "soy sauce", Measure: "1 tbs"}},
		Instructions: []string{}, Tags: []string{},
	}
	ts.usda.SearchResults["soy sauce"] = []usda.SearchFood{{FDCID: 174277, Description: "Soy sauce (shoyu)"}}
	ts.usda.FoodsByID[174277] = usda.Food{
		FDCID: 174277, Description: "Soy sauce (shoyu)",
		Per100g:  usda.Nutrients{usda.Calories: 53, usda.Protein: 8.14, usda.Carbohydrate: 4.93, usda.Fat: 0.57, usda.Fiber: 0.8, usda.Sugars: 0.4, usda.Sodium: 5493},
		Portions: []usda.Portion{{Amount: 1, Unit: "tbsp", GramWeight: 16}},
	}

	// 200 with the documented shape
	var body map[string]any
	res := ts.getJSON(t, "/api/recipes/52772/nutrition", token, &body)
	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, body %v", res.Code, body)
	}
	if body["recipeId"] != "52772" || body["source"] != "USDA FoodData Central" {
		t.Errorf("body = %v", body)
	}

	// 400 non-numeric id
	if res := ts.getJSON(t, "/api/recipes/abc/nutrition", token, &body); res.Code != http.StatusBadRequest {
		t.Errorf("bad id status = %d", res.Code)
	}
	// 404 unknown meal
	if res := ts.getJSON(t, "/api/recipes/99999/nutrition", token, &body); res.Code != http.StatusNotFound {
		t.Errorf("missing meal status = %d", res.Code)
	}
	// 401 without a token
	if res := ts.getJSON(t, "/api/recipes/52772/nutrition", "", &body); res.Code != http.StatusUnauthorized {
		t.Errorf("no-auth status = %d", res.Code)
	}
}

func TestNutritionEndpoint503(t *testing.T) {
	ts := newTestServer(t)
	token := ts.registerAndLogin(t)
	ts.mealdb.Meals["52772"] = mealdb.Meal{
		ID: "52772", Name: "Teriyaki",
		Ingredients:  []mealdb.Ingredient{{Name: "soy sauce", Measure: "1 tbs"}},
		Instructions: []string{}, Tags: []string{},
	}
	ts.usda.Err = errUpstream // reuse/declare a test error like the file's errRepo pattern
	var body map[string]any
	res := ts.getJSON(t, "/api/recipes/52772/nutrition", token, &body)
	if res.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d", res.Code)
	}
	if body["error"] != "nutrition is temporarily unavailable, please try again shortly" {
		t.Errorf("error = %v", body["error"])
	}
}
```

Before running, open `internal/handlers/helpers_test.go` and adjust the helper names in the test above to the file's actual helpers (registration/login helper, JSON GET helper) — reuse what exists rather than inventing parallel ones; add a `getJSON`-style helper only if none exists.

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/handlers/ -run TestNutrition -v`
Expected: FAIL — compile error (`ts.usda` undefined, handler missing).

- [ ] **Step 3: Implement handler and wiring**

`internal/handlers/nutrition_handler.go`:

```go
package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/meal-planner/backend/internal/services"
)

const msgNutritionDown = "nutrition is temporarily unavailable, please try again shortly"

// NutritionHandler serves USDA-based nutrition estimates.
type NutritionHandler struct {
	nutrition services.NutritionService
}

// NewNutritionHandler creates a NutritionHandler.
func NewNutritionHandler(nutrition services.NutritionService) *NutritionHandler {
	return &NutritionHandler{nutrition: nutrition}
}

// Get returns the whole-recipe nutrition estimate for one recipe id.
func (h *NutritionHandler) Get(c *gin.Context) {
	id := c.Param("id")
	if !isPositiveInt(id) {
		c.JSON(http.StatusBadRequest, gin.H{"error": msgInvalidRecipeID})
		return
	}
	est, err := h.nutrition.Estimate(c.Request.Context(), id)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrRecipeNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": msgRecipeNotFound})
		case errors.Is(err, services.ErrUpstreamUnavailable):
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": msgNutritionDown})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": msgInternalServerErr})
		}
		return
	}
	c.JSON(http.StatusOK, est)
}
```

`internal/router/router.go` — extend `Setup` and `New`:

```go
// Setup initializes and configures the router backed by the given database.
func Setup(db *gorm.DB, cfg *config.Config, ov *overrides.Set) *gin.Engine {
	cache := repository.NewCacheRepository(db)
	recipes := services.NewRecipeService(
		mealdb.NewHTTPClient(cfg.MealDBBaseURL, cfg.MealDBTimeout),
		cache,
		services.RecipeCacheTTL{Detail: cfg.MealDBDetailTTL, Search: cfg.MealDBSearchTTL},
		time.Now,
	)
	nutrition := services.NewNutritionService(
		recipes,
		usda.NewHTTPClient(cfg.USDABaseURL, cfg.USDAAPIKey, cfg.USDATimeout),
		cache, ov,
		services.NutritionTTL{Match: cfg.USDAMatchTTL, Food: cfg.USDAFoodTTL, Result: cfg.NutritionTTL},
		time.Now,
	)
	return New(repository.NewUserRepository(db), recipes, nutrition, cfg)
}

// New builds the router on top of the given repositories and services.
func New(userRepo repository.UserRepository, recipes services.RecipeService, nutrition services.NutritionService, cfg *config.Config) *gin.Engine {
```

Inside `New`: `nutritionHandler := handlers.NewNutritionHandler(nutrition)`, the route `recipeRoutes.GET("/:id/nutrition", nutritionHandler.Get)`, and in the `/api` info map's `recipes` section add `"nutrition": "GET /api/recipes/:id/nutrition (protected)"`. Add imports `usda` and `overrides`.

`cmd/server/main.go` — after `cfg := config.Load()`:

```go
	// Validate the committed overrides file before serving anything: running
	// with silently-broken overrides would mis-estimate every recipe.
	ov, err := overrides.Load()
	if err != nil {
		log.Fatalf("Invalid nutrition overrides: %v", err)
	}
	if cfg.USDAAPIKey == "DEMO_KEY" {
		log.Println("USDA_API_KEY not set: using DEMO_KEY (10 requests/hour)")
	} else {
		log.Println("USDA_API_KEY is set")
	}
```

and change the router call to `router.Setup(db, cfg, ov)`.

Update the two test call sites of `router.New` to build and pass a nutrition service: in `helpers_test.go`, add `usda *testutil.USDAClient` to `testServer`, construct it with `testutil.NewUSDAClient()`, build `ov, _ := overrides.Parse([]byte("{}"))` and

```go
nutrition := services.NewNutritionService(recipes, usdaClient, cacheRepo, ov,
	services.NutritionTTL{Match: time.Hour, Food: time.Hour, Result: time.Hour}, time.Now)
```

passing it to `router.New(repo, recipes, nutrition, cfg)` (mirror whatever cache repo the helper already builds for recipes; share one `testutil.CacheRepo`). Same pattern in `router_test.go`.

- [ ] **Step 4: Run the full suite**

Run: `go test ./...`
Expected: PASS everywhere; then `make fmt && make vet && make lint`.

- [ ] **Step 5: Commit**

```bash
git add internal/handlers/ internal/router/ cmd/server/main.go
git commit -m "feat(nutrition): GET /api/recipes/:id/nutrition endpoint"
```

---

### Task 12: Environment files and README

**Files:**
- Modify: `.env.example`, `.env.docker` (if present — check; otherwise skip), `docker-compose.yml`, `docker-compose.dev.yml`, `README.md`

- [ ] **Step 1: Add the env vars**

To `.env.example` (and `.env.docker` if it exists), following each file's existing comment style:

```bash
# USDA FoodData Central (nutrition estimates)
# Get a free key at https://api.data.gov/signup/ - DEMO_KEY allows only 10 requests/hour
USDA_API_KEY=DEMO_KEY
USDA_BASE_URL=https://api.nal.usda.gov/fdc/v1
USDA_TIMEOUT_SECONDS=5
USDA_MATCH_TTL_HOURS=720
USDA_FOOD_TTL_HOURS=2160
NUTRITION_TTL_HOURS=168
```

In both compose files, add to the backend service's `environment:` block (matching how `MEALDB_*` vars are passed there — mirror the existing pattern exactly, including any `${VAR:-default}` style):

```yaml
      USDA_API_KEY: ${USDA_API_KEY:-DEMO_KEY}
      USDA_BASE_URL: ${USDA_BASE_URL:-https://api.nal.usda.gov/fdc/v1}
```

- [ ] **Step 2: Update the README**

- Endpoints table: add `| GET | /api/recipes/:id/nutrition | Bearer | Whole-recipe nutrition estimated from USDA FoodData Central: totals, coverage ("11 of 13 ingredients") and a per-ingredient breakdown. 503 when FDC is unreachable and nothing is cached |`.
- Environment-variable table: the six `USDA_*`/`NUTRITION_*` rows with defaults and notes ("Get a free key at api.data.gov; DEMO_KEY allows 10 requests/hour").
- A short "Nutrition (USDA FoodData Central)" section after the TheMealDB one: what it estimates, that unmeasurable/unmatched ingredients are listed with reasons rather than guessed, whole-recipe only (no serving count exists), caching through `mealdb_cache`, stale-on-error, and FDC attribution ("Data: USDA FoodData Central").
- Status line and Roadmap: move nutrition from planned to implemented.

- [ ] **Step 3: Verify the stack still boots**

Run: `go build ./...` and, if Docker is available, `make docker-build`.
Expected: both succeed.

- [ ] **Step 4: Commit**

```bash
git add .env.example docker-compose.yml docker-compose.dev.yml README.md
git commit -m "docs(nutrition): USDA configuration and endpoint documentation"
```

(Include `.env.docker` in the `git add` if it exists.)

---### Task 13: Manual FDC verification and overrides population

This task is interactive — it calls the real FDC API once, by hand, before merge. It needs a personal key (free at https://api.data.gov/signup/); `DEMO_KEY`'s 10 requests/hour is not enough.

- [ ] **Step 1: Live smoke test**

```bash
USDA_API_KEY=<personal key> make run
# in another shell, register/login to get a token, then:
curl -s -H "Authorization: Bearer $TOKEN" http://localhost:3001/api/recipes/52772/nutrition | python -m json.tool
```

For each of the five sample meals (52772, 52874, 52959, 52940, 52795): record totals and coverage in the PR description; check the ingredient breakdown for obviously wrong matches or absurd totals (e.g. 50,000 kcal — a unit bug; 0 counted — a matching bug). A second request for the same meal must return instantly (result cache) — verify no FDC calls appear in the log.

- [ ] **Step 2: Populate `overrides.json`**

From the breakdowns in step 1, list every ingredient that (a) matched a clearly wrong food, or (b) is a `Count` with reason `noPortion`. For each, find the right food with a manual search (`curl "https://api.nal.usda.gov/fdc/v1/foods/search?api_key=$KEY&query=...&dataType=Foundation,SR%20Legacy"`), confirm the id resolves (`/v1/food/<id>`), and add an entry with a `note`. Target the spec's 30–50 common ingredients: chicken (whole and breasts), egg yolks, salt, onion, garlic cloves, lemon/lime (juice-of), common counted vegetables. **Every committed fdcId must have been resolved against the live API.**

- [ ] **Step 3: Re-verify and commit**

Restart the server (new overrides version), re-run the five recipes, confirm the fixes took. Run `go test ./...` one final time (the embedded-file validation test guards the JSON).

```bash
git add internal/nutrition/overrides/overrides.json
git commit -m "feat(nutrition): populate verified ingredient overrides"
```

---

## Self-review notes

- **Spec coverage:** §2.1 client → Tasks 2–3; §2.2 parser → Task 1; §2.3 overrides → Tasks 4, 13; §2.4 service flow → Tasks 7–8; §2.5 caching/limits → Task 9; §2.6 config → Tasks 6, 12; §3 API contract → Tasks 8 (types), 11 (handler); §5 error handling → Tasks 5, 8, 9, 11; §6 testing → each task carries its tests, golden in Task 10, manual in Task 13. §4 (frontend) is deliberately out: separate plan in the frontend repo.
- **Type consistency check done:** `usda.Nutrients`/`usda.All` (Tasks 2→8), `measure.Amount{Kind, Value}` (1→7→8), `overrides.Set.Get/Version/Normalize` (4→8→9→11), `readCache`/`cached` (5→8→9), `NutritionTTL{Match, Food, Result}` (8→9→11), router signatures (11).
- **Review Focus items** are each pinned to a named test (Tasks 7, 8, 9).
