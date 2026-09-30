# USDA Nutrition for Recipes — Design (Part 2)

**Date:** 2026-09-29
**Status:** Approach and architecture approved in design review; pending written-spec review
**Repos:** `meal-planner-backend` (this repo, owns the API contract), `meal-planner-frontend`
**Builds on:** [Part 1, TheMealDB integration](2026-09-28-themealdb-integration-design.md): its `mealdb` client, `mealdb_cache` read-through cache and 30-day cache purge.

## 1. Goal and context

Part 1 replaced the bundled recipes with TheMealDB and removed nutrition from the UI, because TheMealDB has none. The recipe page shows "Nutrition information coming soon". Part 2 estimates each recipe's nutrition from its ingredient list using [USDA FoodData Central](https://fdc.nal.usda.gov/) (FDC).

**Why:** the project is a job-hunting portfolio piece. Turning messy real-world text ("3/4 cup", "2 chicken breasts", "pinch") into a traceable estimate, and being honest about what could not be counted, is a good showcase of backend data handling.

**The data we have**
- TheMealDB gives up to 20 `{name, measure}` pairs per meal. Measures are free text: `1kg`, `200ml`, `2 tbs`, `¼ cup`, `1 (12 oz.)`, `2 chicken breasts`, `1 whole`, `Juice of 1`, `pinch`, `to taste`, `to serve`, or empty. It gives **no serving count**.
- FDC gives nutrients per 100 g, and for most foods a list of portions with gram weights (for shoyu soy sauce: 1 tbsp = 16 g, 1 cup = 255 g).

**Success criteria**
- The recipe page shows whole-recipe calories, protein, carbohydrate, fat, fibre, sugars and sodium, clearly labelled as an estimate.
- It says how many ingredients were counted ("based on 11 of 13 ingredients"). Each ingredient can be traced to the FDC food it matched, or to the reason it was not counted.
- After a recipe's first view, repeat views make no FDC calls.
- If FDC is down or rate-limited, only the nutrition panel shows an error; the recipe itself is unaffected.
- The new code has tests in the repo's existing style, CI never calls FDC, and CI passes.

**Decisions made in design review**
| Decision | Choice |
|---|---|
| What to show | **Whole-recipe totals plus coverage.** No per-serving figures, because no serving count exists and inventing one would be an estimate built on a guess. |
| Matching | **Hybrid:** automatic ranked search, plus a small committed overrides file for known-bad matches and per-item weights. |
| API shape | A **separate endpoint**, `GET /api/recipes/:id/nutrition`, so nutrition can fail without breaking the recipe. |
| Unconvertible amounts | Not counted, and listed with a reason. Never guessed. |
| API key | From `USDA_API_KEY`; falls back to FDC's shared `DEMO_KEY` with a start-up warning. |

## 2. Backend architecture

New units follow the existing client → service → handler → router pattern. Each has one job and can be tested alone.

### 2.1 `internal/usda` — FDC client
- An interface, `Client`, with two methods:
  - `Search(ctx, query) ([]SearchFood, error)`: `GET /fdc/v1/foods/search` with `dataType=Foundation,SR Legacy` and `pageSize=10`. Branded and survey foods are excluded: they are product- or survey-specific, not generic ingredients.
  - `Foods(ctx, ids) ([]Food, error)`: `POST /fdc/v1/foods` with up to 20 ids and `format=full`, returning nutrients and portions. More than 20 ids are split into batches.
- `SearchFood` is `{FDCID, Description, DataType}`. `Food` is `{FDCID, Description, DataType, Per100g Nutrients, Portions []Portion}`, where `Portion` is `{Amount, Unit, Modifier, GramWeight}`.
- Nutrients are read by FDC nutrient **number**: energy 208 (kcal), protein 203, fat 204, carbohydrate 205, fibre 291, total sugars 269 (g) and sodium 307 (mg). Some Foundation foods report energy only as Atwater factors (numbers 957 and 958), so energy falls back to 958 and then 957. A nutrient FDC does not report is **absent**, not zero.
- The base URL (default `https://api.nal.usda.gov/fdc/v1`), timeout (default 5s) and API key come from config. The key is sent as the `api_key` query parameter and never logged.
- Non-2xx responses, timeouts and malformed JSON return an error. A 429 (rate limit) is an error like any other upstream failure.

### 2.2 `internal/nutrition/measure` — measure parser
A pure function, `Parse(measure string) Amount`, with no I/O.
- `Amount` is `{Quantity float64, Unit Unit, Kind Kind}`. `Kind` is `Mass`, `Volume`, `Count` or `Unmeasurable`.
- Quantities: integers, decimals (`1.2`), fractions (`3/4`), mixed numbers (`1 1/2`), Unicode fractions (`¼`, `½`, `¾`, `⅓`, `⅔`, `⅛`) and ranges (`2-3`, which takes the midpoint).
- Mass units: `g`, `gram(s)`, `kg`, `oz`, `ounce(s)`, `lb`, `lbs`, `pound(s)`, converted to grams.
- Volume units: `ml`, `l`, `litre(s)`/`liter(s)`, `tsp`/`teaspoon(s)`, `tbs`/`tbsp`/`tablespoon(s)`, `cup(s)`, `fl oz`, `pint(s)`. Unit matching ignores case, so `4 Tablespoons` is handled.
- A parenthesised package size wins: `1 (12 oz.)` is 340 g.
- A bare number, or a number followed by words that are not units (`2`, `2 chopped`, `2 free-range`, `5 thinly sliced`, `1 whole`, `Juice of 1`), is a `Count`.
- `pinch`, `dash`, `to taste`, `to serve`, `to garnish`, `for frying`, `drizzle`, `handful`, empty text and anything else with no quantity are `Unmeasurable`.
- The parser is table-tested against every measure in the five sample meals below, plus the edge cases above.

### 2.3 `internal/nutrition/overrides` — committed overrides
`overrides.json` is embedded with `go:embed` and keyed by normalised ingredient name (lower-case, `_` as a space, single spaces):

```json
{
  "chicken":         { "fdcId": 171077, "itemGrams": 1500, "note": "whole chicken, raw, meat and skin" },
  "chicken breasts": { "fdcId": 171077, "itemGrams": 174 },
  "egg yolks":       { "fdcId": 172184, "itemGrams": 17 },
  "salt":            { "fdcId": 173468 }
}
```

- `fdcId` pins the FDC food, skipping search. `itemGrams` gives the weight of one item for `Count` measures that FDC's portions don't cover. Either field may appear alone.
- The initial file covers about 30 to 50 common ingredients whose automatic match is wrong, or whose count measure needs a weight. Every `fdcId` is checked against the FDC API when the file is written. The ids in the example above are illustrative.
- The file's content hash is the **matcher version**, which is part of the cache keys in §2.5. Editing the file retires every result built from the old version.

### 2.4 `NutritionService`
`Estimate(ctx, mealID) (*RecipeNutrition, error)`. It uses the existing `RecipeService.Get` to load the meal, so TheMealDB caching and errors are unchanged.

For each ingredient line:
1. **Parse** the measure (§2.2). `Unmeasurable` → not counted, reason `unmeasurable`.
2. **Match** a food: use the override's `fdcId` if one exists. Otherwise take the best FDC search result for the ingredient name, ranked:
   1. descriptions whose first comma-separated segment matches the name (singular or plural)
   2. then descriptions containing `raw`
   3. then FDC's own order
   No result → not counted, reason `noMatch`.
3. **Convert** to grams:
   - `Mass`: already grams.
   - `Volume`: use the food's FDC portion for that unit (`cup`, `tbsp`, `tsp`, `fl oz`, `ml`), deriving the others from any one with 1 cup = 16 tbsp = 48 tsp = 236.6 ml. No volume portion → not counted, reason `noPortion`.
   - `Count`: use the override's `itemGrams`. Otherwise use the first FDC portion whose description names one item: `whole`, `large`, `medium`, `fruit`, `unit`, or the ingredient's own last word (e.g. `breast`, `clove`). Neither → not counted, reason `noPortion`. A known approximation: "Juice of 1 | Lemon" counts a whole lemon unless an override says otherwise.
4. **Scale** the food's per-100 g nutrients by grams ÷ 100.

Then **sum** every counted ingredient. If a counted ingredient lacks a nutrient, that nutrient's total is marked incomplete, not treated as zero.

Fetching:
- Search calls for different ingredients run concurrently, at most 4 at a time.
- All the food details a recipe needs are then fetched with batched `Foods` calls.
- Any upstream failure fails the whole estimate with `ErrUpstreamUnavailable`, unless an older cached copy exists (§2.5). **A partial result is never cached.**

### 2.5 Caching and rate limits
Everything reuses Part 1's `mealdb_cache` table and its generic `cached[T]` read-through helper: fresh hit, else fetch and store, else serve an older copy on upstream error, else fail. Old rows are removed by Part 1's 30-day purge.

`cached[T]` currently takes a `*recipeService`. It is generalised into a small read-through cache type that holds the repository and clock, shared by both services, with no change in behaviour. Part 1's tests must keep passing unchanged.

| Key | Holds | TTL |
|---|---|---|
| `usda:match:<version>:<normalised name>` | the chosen FDC id, or "no match" | 30 days |
| `usda:food:<fdcId>` | one food's nutrients and portions | 90 days |
| `nutrition:<version>:<mealId>` | the finished `RecipeNutrition` | 7 days |

- A cold recipe with 13 ingredients costs about 13 search calls plus 1 batched details call.
- Ingredients are shared across recipes, so later recipes cost fewer calls. A repeat view costs none.
- FDC's personal-key limit is 1,000 requests an hour, and `DEMO_KEY` allows 10. The TTLs are env-configurable like Part 1's: `USDA_MATCH_TTL_HOURS`, `USDA_FOOD_TTL_HOURS` and `NUTRITION_TTL_HOURS`.

### 2.6 Config
| Env var | Default | Notes |
|---|---|---|
| `USDA_API_KEY` | `DEMO_KEY` | Logged at start-up as "set" or "using DEMO_KEY (10 requests/hour)"; never logged in full |
| `USDA_BASE_URL` | `https://api.nal.usda.gov/fdc/v1` | Overridable for tests |
| `USDA_TIMEOUT_SECONDS` | `5` | |
| TTLs | see §2.5 | |

`.env.example`, `.env.docker`, both compose files and the README are updated. The README explains how to get a free key at api.data.gov.

## 3. API contract

`GET /api/recipes/:id/nutrition` (protected, like the other recipe routes)

```json
{
  "recipeId": "52772",
  "source": "USDA FoodData Central",
  "totals": {
    "calories": 2340, "protein": 182.1, "carbohydrate": 210.4, "fat": 78.0,
    "fiber": 21.3, "sugars": 64.2, "sodium": 9120
  },
  "incomplete": ["sugars"],
  "coverage": { "counted": 11, "total": 13 },
  "ingredients": [
    { "name": "soy sauce", "measure": "3/4 cup", "status": "counted", "grams": 191.3,
      "food": { "fdcId": 174277, "description": "Soy sauce made from soy and wheat (shoyu)" },
      "calories": 101 },
    { "name": "salt", "measure": "pinch", "status": "notCounted", "reason": "unmeasurable" }
  ]
}
```

- Units: `calories` in kcal (integer), `sodium` in mg (integer), and everything else in grams to one decimal place.
- `incomplete` lists the nutrients some counted ingredient lacked, so the total may be low. It is omitted when empty.
- `reason` is one of `unmeasurable`, `noMatch` or `noPortion`.
- `ingredients` keeps the recipe's order.

| Case | Status | Body |
|---|---|---|
| Success, including 0 counted ingredients | 200 | as above (`coverage.counted` may be 0) |
| Non-numeric id | 400 | existing invalid-id message |
| Meal not found | 404 | existing not-found message |
| TheMealDB or FDC down, nothing cached | 503 | `{"error": "nutrition is temporarily unavailable, please try again shortly"}` |

## 4. Frontend

- `recipesApi.getNutrition(id)` calls the endpoint and returns a typed `RecipeNutrition`. Errors map through the existing `RecipeApiError` kinds; a 503 shows the nutrition message above. Results are kept in the same per-session cache as recipe details.
- `NutritionCard` takes the recipe id, loads with `useAsync` and replaces the "coming soon" text:
  - **Loading:** a small skeleton inside the card.
  - **Error:** a compact `ErrorPanel` with Retry. The rest of the page is unaffected.
  - **Ready:**
    - The heading reads "Nutrition (estimate) · whole recipe".
    - Calories are shown large, with a grid of protein, carbohydrate, fat, fibre, sugars and sodium.
    - "Based on 11 of 13 ingredients".
    - An incomplete nutrient is shown as "at least 64 g".
    - When fewer than half the ingredients were counted, a note reads "Partial estimate: most ingredients could not be measured".
    - When none were counted, the card says so instead of showing zeros.
  - **Breakdown:** a native `<details>` "Ingredient breakdown" lists each ingredient with its matched food and calories, or "not counted" with a plain-English reason. It is keyboard- and screen-reader-accessible by default.
  - **Attribution:** a "Data: USDA FoodData Central" link, as FDC asks.
- The card never computes nutrition itself; it only formats what the API returns.

## 5. Error handling summary
- An upstream failure with an older cached copy serves that copy.
- An upstream failure with no cached copy returns 503, and the frontend shows the error with Retry.
- A malformed measure is never an error, just an ingredient that isn't counted.
- The overrides file is validated when the server starts. Invalid JSON, a non-positive `fdcId` or a non-positive `itemGrams` stops the server with a clear message, and a unit test covers it.

## 6. Testing
- **`measure`:** table tests covering every measure in the sample meals 52772, 52874, 52959, 52940 and 52795, plus the §2.2 edge cases.
- **`usda` client:** `httptest` server tests for search filtering, batching over 20 ids, nutrient-number mapping (including the energy fallback), absent nutrients, and non-2xx, 429, timeout and malformed-JSON responses.
- **`NutritionService`:**
  - fakes for `usda.Client` and `RecipeService`, plus the in-memory cache
  - override precedence, ranking, each conversion path and each `reason`
  - incomplete-nutrient totals, rounding, and concurrency limited to 4
  - no partial result cached, serving an older copy on failure, and a changed matcher version missing the old cache
- **Golden test:** one recipe whose FDC responses are recorded fixtures, asserting totals and coverage within 1%, so accidental changes to the maths are caught.
- **Handler:** 200, 400, 404 and 503 cases.
- **Frontend:**
  - Vitest for response formatting (rounding, the "at least" wording, partial and none)
  - Playwright fixtures for the ready, partial, none and 503-with-Retry states
  - a check that the recipe page renders even when the nutrition endpoint fails
- **Manual:** one real run against FDC with a personal key before merging, recording a few recipes' totals in the PR as a sanity check.

## 7. Out of scope
- Per-serving figures, and editing servings or quantities.
- Micronutrients beyond sodium, and daily-value percentages.
- Showing nutrition on recipe cards, search results or meal-plan totals. A later part could add plan totals once recipe estimates exist.
- A UI for users to correct matches. Fixes go into `overrides.json`.
- Branded or survey foods.
