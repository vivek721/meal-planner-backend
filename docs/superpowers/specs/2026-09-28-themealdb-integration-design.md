# TheMealDB Recipe Integration — Design (Part 1)

**Date:** 2026-09-28
**Status:** Approved in design review, pending written-spec review
**Repos:** `meal-planner-backend` (this repo, owns the API contract), `meal-planner-frontend`
**Follow-up:** Part 2, USDA nutrition, gets its own spec and builds on this recipe cache.

## 1. Goal and context

The frontend currently bundles 70 hand-written sample recipes (`src/data/mockRecipes.ts`) with generated placeholder images. This project replaces them with real recipes from [TheMealDB](https://www.themealdb.com), served through the Go backend.

**Why:** the project is a job-hunting portfolio piece. Real recipes and photos make the demo convincing, and a backend that consumes and caches an external API shows full-stack skills.

**Success criteria**
- The app shows real TheMealDB recipes with real photos on the browse, detail, favourites, meal-plan picker and suggestions screens.
- Browsing, recipe detail, favourites, meal planning and suggestions all keep working.
- The app never shows data it doesn't have. Fields TheMealDB lacks are removed from the UI, not estimated.
- The new backend code has client, service and handler tests in the repo's existing style, and CI passes.

**Decisions made in design review**
| Decision | Choice |
|---|---|
| Missing fields (time, rating, nutrition, servings, dietary labels) | Removed from the UI. Nutrition comes back in Part 2 (USDA). |
| Scope | Part 1 is TheMealDB only. Part 2 (USDA nutrition) is a separate spec. |
| Data flow | **Fetch on demand with a Postgres read-through cache** (not a bulk import, not direct frontend calls) |

## 2. Backend architecture

New units, following the existing model → repository → service → handler → router pattern.

### 2.1 `internal/mealdb` — TheMealDB client
- An interface, `Client`, with methods for `Search(q)`, `FilterByCategory(c)`, `FilterByArea(a)`, `FilterByIngredient(i)`, `Lookup(id)`, `Categories()` and `Areas()`. It is backed by `search.php`, `filter.php`, `lookup.php`, `categories.php` and `list.php?a=list`.
- An HTTP implementation using `net/http` with a configurable timeout (default 5s) and base URL.
- It normalises TheMealDB's format into Go structs: `strIngredient1..20` + `strMeasure1..20` become `[]Ingredient{Name, Measure}`, with empty pairs dropped. `strInstructions` is split into steps on line breaks, removing blank lines and bare `STEP n` markers. `strTags` (comma-separated) becomes `[]string`, and empty `strYoutube`/`strSource` become absent.
- TheMealDB returns `{"meals": null}` for no results. This maps to an empty list, or to "not found" for `Lookup`.
- Non-2xx responses, timeouts and malformed JSON return an error. They never panic.

### 2.2 `models.CachedResponse` — table `mealdb_cache`
| Column | Type | Notes |
|---|---|---|
| `key` | varchar PK | e.g. `lookup:52772`, `filter:c=Seafood`, `search:q=chicken`, `categories`, `areas` (normalised: lower-cased, trimmed) |
| `payload` | jsonb | the normalised result from the client |
| `fetched_at` | timestamptz | |
| `expires_at` | timestamptz | `fetched_at + TTL` |

Created by the existing `AutoMigrate`. The repository interface is `Get(key)` and `Upsert(entry)`, with a GORM implementation and an in-memory test implementation in `internal/testutil`.

### 2.3 `services.RecipeService` — read-through cache
For every cached call:
1. A fresh cache entry is returned.
2. If the entry is missing or expired, fetch from TheMealDB, upsert the result, and return it.
3. If the fetch fails and an expired entry exists, return the **stale** entry and log a warning (stale-on-error).
4. If the fetch fails and nothing is cached, return `ErrUpstreamUnavailable`, which the handler maps to 503.

**Combined filters:**
- Each requested filter (`category`, `cuisine`, `ingredient`) is fetched separately, through the cache, and the results are intersected by ID. The intersection keeps the order of the first list.
- With `q`, the name search runs first (it returns full meals). The results are then narrowed by `category`/`cuisine` using the fields already in those meals. For `ingredient`, they are intersected with the ingredient filter's ID list.
- Paging is applied after combining: `page` starts at 1, `limit` defaults to 24 and is capped at 50.

**TTLs:** `lookup`, `categories` and `areas` last 7 days. `search` and `filter` last 24 hours. Both are configurable.

**Summary fields:** a `RecipeSummary` includes `category`/`cuisine` when they are known. That is when the request filtered on that value, or when the result came from a name search, which returns full meals.

### 2.4 `handlers.RecipeHandler` and routes
All routes are under `/api/recipes` and require authentication (`middleware.AuthMiddleware`), like the other user-facing routes. `router.New` gains the recipe repository and client as parameters. The `/api` info map lists the new endpoints.

### 2.5 Configuration
| Env var | Default |
|---|---|
| `MEALDB_BASE_URL` | `https://www.themealdb.com/api/json/v1/1` |
| `MEALDB_TIMEOUT_SECONDS` | `5` |
| `MEALDB_DETAIL_TTL_HOURS` | `168` |
| `MEALDB_SEARCH_TTL_HOURS` | `24` |

Added to `config.Load`, `.env.example`, `.env.docker` and both compose files. The public key `1` is part of the URL; no secret is needed.

## 3. API contract

Errors use the existing `{ "error": "..." }` shape.

| Endpoint | Response | Errors |
|---|---|---|
| `GET /api/recipes/categories` | `[{ name, thumbnail, description }]` | 503 |
| `GET /api/recipes/cuisines` | `string[]` (sorted) | 503 |
| `GET /api/recipes?q=&category=&cuisine=&ingredient=&page=&limit=` | `{ recipes: RecipeSummary[], total, page, totalPages }` | 400 if no `q`/`category`/`cuisine`/`ingredient` is given, or if `page`/`limit` is invalid; 503 |
| `GET /api/recipes/:id` | `Recipe` | 400 if the id is not numeric, 404 if not found, 503 |

```jsonc
// RecipeSummary
{ "id": "52772", "name": "Teriyaki Chicken Casserole",
  "thumbnail": "https://www.themealdb.com/images/media/meals/wvpsxx1468256321.jpg",
  "category": "Chicken",   // optional
  "cuisine": "Japanese" }  // optional

// Recipe
{ "id": "52772", "name": "...", "thumbnail": "...", "category": "Chicken", "cuisine": "Japanese",
  "ingredients": [ { "name": "soy sauce", "measure": "3/4 cup" } ],
  "instructions": [ "Preheat oven to 350° F.", "..." ],
  "tags": [ "Meat", "Casserole" ],
  "youtubeUrl": "https://www.youtube.com/watch?v=4aZr5hZXP_s",   // optional
  "sourceUrl": "https://..." }                                   // optional
```

`category` values are TheMealDB's categories (Beef, Breakfast, Chicken, Dessert, Goat, Lamb, Miscellaneous, Pasta, Pork, Seafood, Side, Starter, Vegan, Vegetarian). These replace the frontend's old `Breakfast | Lunch | Dinner | Snack | Dessert` union.

## 4. Frontend changes (`meal-planner-frontend`)

**Data access**
- A new `src/services/api/recipesApi.ts` with typed calls, built on the existing `apiClient` (auth header and 401 handling included).
- An in-memory, per-session cache for recipe details in the recipes data layer, so each recipe is fetched at most once per session.
- Deleted: `src/data/mockRecipes.ts`, `src/utils/recipeThumbnail.ts`, the synchronous `RecipeService` search, sort and scaling code, and `adjustServingSize`.
- The `Recipe`/`RecipeSummary` types are updated to the §3 contract. `prepTime`, `cookTime`, `servings`, `nutrition`, `rating`, `reviewCount`, `description` and `dietaryTags` are removed.

**Screens**
- **Recipes (`/recipes`):**
  - Lands on the category grid (with photos).
  - Search by name, and filter by cuisine and main ingredient.
  - Results are paged.
  - Shows loading, empty and error states.
  - The time, rating and popularity sorts are removed.
- **Recipe detail:**
  - Shows the photo, category, cuisine, tags, ingredients with measures, steps, and a "Watch on YouTube" link when present.
  - The nutrition panel shows "Nutrition information coming soon".
  - The serving adjuster is removed.
  - "Similar recipes" shows up to 4 from the same category, excluding the current one.
- **Favourites:** IDs stay in localStorage. Each saved recipe's details are loaded (cached). You can search by name and filter by category; the time and rating sorts are removed.
- **Meal plan recipe picker (`RecipeBrowserModal`):** search, plus category and cuisine filters, via the API.
- **Meal slots:**
  - `MealPlanService.addMeal` takes the chosen recipe's summary (`id`, `name`, `thumbnail`, and `category` when known), which is already loaded in the picker, instead of looking it up by ID.
  - A slot stores `recipeId`, `recipeName`, `thumbnail`, `category` (optional; suggestions use it for variety) and `addedAt`. The calendar renders from the stored slot with no API calls. The time label is removed.
- **Suggestions:** `MockAIService` is renamed `SuggestionService`; it is still a heuristic with no AI.
  - The time of day chooses the categories: before 11:00 → Breakfast; otherwise one of Chicken, Beef, Pasta, Seafood, Vegetarian, Lamb, Pork.
  - It prefers categories not yet planned this week and excludes recipes already planned.
  - It fetches one category through the API and picks 4 at random, each with a short reason such as "Breakfast idea" or "Something different this week".

**Old saved data**
- Meal-plan slots saved with the old IDs (`recipe-001`…) still render, from their stored name and photo. Opening one shows "This recipe is no longer available".
- Favourites with non-numeric IDs are dropped the first time favourites load.

**Errors and loading**
- A shared error panel with a Retry button. A 503 shows "Recipes are temporarily unavailable, please try again shortly".

**Copy**
- The onboarding "Discover Recipes" slide and the README no longer mention the 70 bundled recipes, and the README lists the removed features as known limitations.

## 5. Testing
**Backend**
- `mealdb` client tests against `httptest` fake servers:
  - normalising ingredients, steps and tags
  - `meals: null` responses
  - non-2xx responses, timeouts and malformed JSON
- `RecipeService` tests with a fake client and the in-memory cache:
  - fresh, expired-then-refetch, stale-on-error and nothing-cached-and-error (`ErrUpstreamUnavailable`)
  - combined-filter intersection and `q` + filters
  - paging bounds
- Handler tests through `router.New` with `httptest`:
  - auth required
  - 400/404/503 mapping
  - response shapes
- The existing coverage gate (70%) must still pass.

**Frontend**
- Typecheck, lint and build pass.
- The Playwright specs that touch recipes intercept `/api/recipes*` with fixture data, so tests never call TheMealDB.

**Manual**
- Run the backend with Docker against the real TheMealDB. Check each endpoint, the cache rows in `mealdb_cache`, and stale-on-error (by pointing `MEALDB_BASE_URL` at an unreachable host once the cache is warm).
- Click through the full frontend flow in Chrome before merging the frontend PR.

## 6. Rollout
1. **Backend PR** (additive; no existing endpoint changes): merged first after tests and a local check against real TheMealDB.
2. **Frontend PR:** merged after the Chrome click-through against the local stack.

Both READMEs document the change. The backend README credits TheMealDB and notes its terms: the key `1` is for development and educational use, and a public production deployment should follow their supporter terms.

## 7. Out of scope
- USDA nutrition (Part 2).
- Estimated cooking times, ratings or servings.
- AI-generated suggestions, shopping lists, user-created recipes.
- Moving favourites or meal plans from localStorage to the backend.
- Explicit rate limiting of TheMealDB calls. The cache keeps volume low, and TTLs are configurable.
