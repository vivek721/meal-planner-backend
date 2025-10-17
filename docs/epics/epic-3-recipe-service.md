# Epic 3: Recipe Service
## Recipe Management & Search API

---

**Epic ID:** EPIC-3
**Priority:** P0 (Critical)
**Estimated Effort:** 50 hours
**Sprint:** Week 3
**Owner:** Senior Backend Developer
**Status:** Not Started
**Dependencies:** Epic 1 (Infrastructure), Epic 2 (Authentication)

---

## Overview

Implement complete recipe management system including CRUD operations, full-text search, image uploads, favorites, and admin controls. This epic transforms the frontend's mock recipe data into a production-ready service with database persistence and advanced search capabilities.

## Goals

1. Implement recipe CRUD endpoints for admins
2. Build full-text search with PostgreSQL
3. Add recipe filtering and pagination
4. Implement image upload to S3
5. Create favorites functionality
6. Add category and tag management
7. Implement Redis caching layer

## User Stories

### US-3.1: As a user, I can browse all recipes with filtering

**Acceptance Criteria:**
- GET `/api/v1/recipes` endpoint returns paginated recipes
- Filters: category, dietary tags, cuisine, max prep/cook time, difficulty
- Sorting: rating, created date, prep time
- Pagination: page, limit (max 100 per page)
- Results cached in Redis (15 min TTL)
- Response includes recipe preview data (no full ingredients)

**Query Examples:**
```
GET /api/v1/recipes?category=Dinner&dietary=Vegan&maxPrepTime=30&sort=rating:desc&page=1&limit=20
GET /api/v1/recipes?cuisine=Italian&difficulty=Easy&sort=createdAt:desc
```

---

### US-3.2: As a user, I can view full recipe details

**Acceptance Criteria:**
- GET `/api/v1/recipes/:id` returns complete recipe data
- Includes: ingredients, instructions, nutrition, images
- Recipe not found returns 404
- Results cached in Redis (1 hour TTL)
- Increments view count (async, non-blocking)

**Response Structure:**
```json
{
  "data": {
    "id": "recipe-001",
    "name": "Chicken Tacos",
    "description": "...",
    "category": "Dinner",
    "cuisine": "Mexican",
    "prepTime": 15,
    "cookTime": 20,
    "ingredients": [...],
    "instructions": [...],
    "nutrition": {...},
    "tags": ["Mexican", "Quick"],
    "imageUrl": "https://cdn.mealplanner.com/...",
    "rating": 4.5,
    "reviewCount": 89
  }
}
```

---

### US-3.3: As a user, I can search recipes by name, ingredients, or tags

**Acceptance Criteria:**
- GET `/api/v1/recipes/search?q=chicken` endpoint
- Full-text search using PostgreSQL tsvector
- Search includes: name, description, ingredients, tags, cuisine
- Results ranked by relevance score
- Autocomplete support (minimum 3 characters)
- Search results cached in Redis (15 min TTL)

**Search Implementation:**
```sql
SELECT id, name, image_url, prep_time, rating,
       ts_rank(search_vector, query) AS rank
FROM recipes,
     to_tsquery('english', 'chicken & tacos') AS query
WHERE search_vector @@ query
  AND deleted_at IS NULL
ORDER BY rank DESC, rating DESC
LIMIT 20 OFFSET 0;
```

---

### US-3.4: As a user, I can favorite/unfavorite recipes

**Acceptance Criteria:**
- POST `/api/v1/recipes/:id/favorite` adds to favorites
- DELETE `/api/v1/recipes/:id/favorite` removes from favorites
- GET `/api/v1/recipes/favorites` lists user's favorites
- Duplicate favorite returns 409 Conflict
- Favorites cached in Redis per user

**Favorites Table:**
```sql
CREATE TABLE favorites (
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  recipe_id UUID NOT NULL REFERENCES recipes(id) ON DELETE CASCADE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (user_id, recipe_id)
);
```

---

### US-3.5: As an admin, I can create new recipes

**Acceptance Criteria:**
- POST `/api/v1/recipes` endpoint (admin only)
- Validates all required fields (name, category, ingredients, etc.)
- Generates search_vector automatically
- Returns 201 Created with full recipe data
- Invalidates recipe list cache
- Requires `role: admin` in JWT

**Request Body:**
```json
{
  "name": "Vegan Buddha Bowl",
  "description": "Nutritious and colorful",
  "category": "Dinner",
  "cuisine": "Fusion",
  "prepTime": 20,
  "cookTime": 15,
  "servings": 2,
  "difficulty": "Easy",
  "ingredients": [
    {
      "name": "Quinoa",
      "quantity": "1",
      "unit": "cup",
      "category": "Grains"
    }
  ],
  "instructions": [
    "Cook quinoa according to package",
    "Prepare vegetables",
    "Assemble bowl"
  ],
  "nutrition": {
    "calories": 420,
    "protein": 18,
    "carbohydrates": 52,
    "fat": 16
  },
  "tags": ["Vegan", "Gluten-Free", "High-Protein"]
}
```

---

### US-3.6: As an admin, I can update existing recipes

**Acceptance Criteria:**
- PUT `/api/v1/recipes/:id` endpoint (admin only)
- Partial updates supported
- Updates `updated_at` timestamp
- Invalidates recipe cache
- Returns 404 if recipe not found

---

### US-3.7: As an admin, I can delete recipes

**Acceptance Criteria:**
- DELETE `/api/v1/recipes/:id` endpoint (admin only)
- Soft delete (sets `deleted_at` timestamp)
- Removes from search results
- Invalidates all recipe caches
- Removes from active meal plans (cascade)

---

### US-3.8: As a developer, I can upload recipe images to S3

**Acceptance Criteria:**
- POST `/api/v1/recipes/:id/upload-url` generates presigned S3 URL
- Frontend uploads directly to S3 (bypasses API)
- POST `/api/v1/recipes/:id/confirm-upload` triggers image processing
- Lambda/Fargate resizes images: original, large, medium, thumbnail
- WebP conversion for modern browsers
- CloudFront serves images with 1-week cache

**Image Upload Flow:**
1. Request presigned URL from API
2. Upload image to S3 temp bucket
3. Confirm upload to API
4. API triggers image processor (Lambda)
5. Processor generates variants (large, medium, thumb)
6. Processor moves to final S3 location
7. API updates recipe with image URLs

---

### US-3.9: As a user, I can get recipe categories

**Acceptance Criteria:**
- GET `/api/v1/recipes/categories` returns all categories with counts
- Cached in Redis (1 hour TTL)
- Updates when recipes added/removed

**Response:**
```json
{
  "data": [
    { "name": "Breakfast", "count": 45 },
    { "name": "Lunch", "count": 78 },
    { "name": "Dinner", "count": 156 },
    { "name": "Snacks", "count": 32 },
    { "name": "Desserts", "count": 24 }
  ]
}
```

---

## Technical Requirements

### Database Schema

**recipes** table (already defined in DATABASE_DESIGN.md):
- JSONB columns for ingredients, instructions, nutrition, tags
- Generated tsvector column for full-text search
- GIN indexes on search_vector and tags
- B-tree indexes on category, rating, created_at

### Caching Strategy

| Resource | Cache Key | TTL | Invalidation |
|----------|-----------|-----|--------------|
| Recipe list | `recipes:list:{filters}:{page}` | 15 min | Recipe created/updated/deleted |
| Recipe detail | `recipe:{id}` | 1 hour | Recipe updated/deleted |
| Search results | `search:{query}:{filters}` | 15 min | Recipe created/updated |
| Favorites | `favorites:{userId}` | 10 min | Favorite added/removed |
| Categories | `categories` | 1 hour | Recipe created/deleted |

### Image Processing

**S3 Bucket Structure:**
```
meal-planner-images-production/
├── recipes/
│   └── {recipeId}/
│       ├── original/image.jpg
│       ├── large/image_1200x800.webp
│       ├── medium/image_600x400.webp
│       └── thumbnail/image_200x200.webp
└── temp/
    └── {uploadId}/image.jpg (TTL: 24 hours)
```

**Lambda Function** (or separate Fargate task):
```javascript
// imageProcessor.js
const sharp = require('sharp');

exports.handler = async (event) => {
  const { bucket, key } = event;

  // Download original
  const original = await s3.getObject({ Bucket: bucket, Key: key }).promise();

  // Generate variants
  const variants = [
    { name: 'large', width: 1200, height: 800 },
    { name: 'medium', width: 600, height: 400 },
    { name: 'thumbnail', width: 200, height: 200 },
  ];

  for (const variant of variants) {
    const resized = await sharp(original.Body)
      .resize(variant.width, variant.height, { fit: 'cover' })
      .webp({ quality: 85 })
      .toBuffer();

    await s3.putObject({
      Bucket: bucket,
      Key: `recipes/${recipeId}/${variant.name}/image.webp`,
      Body: resized,
      ContentType: 'image/webp',
    }).promise();
  }
};
```

---

## API Endpoints

### Endpoint Summary

| Method | Endpoint | Auth | Description |
|--------|----------|------|-------------|
| GET | `/recipes` | Required | List recipes with filters |
| GET | `/recipes/:id` | Required | Get recipe details |
| POST | `/recipes` | Admin | Create recipe |
| PUT | `/recipes/:id` | Admin | Update recipe |
| DELETE | `/recipes/:id` | Admin | Delete recipe |
| GET | `/recipes/search` | Required | Search recipes |
| GET | `/recipes/categories` | Required | Get categories |
| POST | `/recipes/:id/favorite` | Required | Add to favorites |
| DELETE | `/recipes/:id/favorite` | Required | Remove from favorites |
| GET | `/recipes/favorites` | Required | List user favorites |
| POST | `/recipes/:id/upload-url` | Admin | Get S3 upload URL |
| POST | `/recipes/:id/confirm-upload` | Admin | Confirm image upload |

---

## Testing Requirements

### Unit Tests

```typescript
describe('RecipeService', () => {
  describe('getRecipes', () => {
    it('should return paginated recipes', async () => {
      const result = await recipeService.getRecipes({ page: 1, limit: 20 });
      expect(result.data).toHaveLength(20);
      expect(result.pagination.total).toBeGreaterThan(0);
    });

    it('should filter by category', async () => {
      const result = await recipeService.getRecipes({ category: 'Dinner' });
      result.data.forEach(recipe => {
        expect(recipe.category).toBe('Dinner');
      });
    });
  });

  describe('searchRecipes', () => {
    it('should search by name', async () => {
      const result = await recipeService.searchRecipes('chicken');
      expect(result.data.length).toBeGreaterThan(0);
      expect(result.data[0].name.toLowerCase()).toContain('chicken');
    });

    it('should rank by relevance', async () => {
      const result = await recipeService.searchRecipes('chicken tacos');
      expect(result.data[0].relevanceScore).toBeGreaterThan(result.data[1].relevanceScore);
    });
  });
});
```

### Integration Tests

```typescript
describe('GET /api/v1/recipes', () => {
  it('should return recipes', async () => {
    const res = await request(app)
      .get('/api/v1/recipes')
      .set('Authorization', `Bearer ${validToken}`);

    expect(res.status).toBe(200);
    expect(res.body.data).toBeInstanceOf(Array);
    expect(res.body.pagination).toBeDefined();
  });

  it('should filter by category', async () => {
    const res = await request(app)
      .get('/api/v1/recipes?category=Dinner')
      .set('Authorization', `Bearer ${validToken}`);

    expect(res.status).toBe(200);
    res.body.data.forEach(recipe => {
      expect(recipe.category).toBe('Dinner');
    });
  });

  it('should require authentication', async () => {
    const res = await request(app).get('/api/v1/recipes');
    expect(res.status).toBe(401);
  });
});

describe('POST /api/v1/recipes', () => {
  it('should create recipe (admin only)', async () => {
    const res = await request(app)
      .post('/api/v1/recipes')
      .set('Authorization', `Bearer ${adminToken}`)
      .send({
        name: 'Test Recipe',
        category: 'Dinner',
        prepTime: 15,
        cookTime: 20,
        servings: 4,
        difficulty: 'Easy',
        ingredients: [{ name: 'Chicken', quantity: '1', unit: 'lb' }],
        instructions: ['Step 1', 'Step 2'],
        nutrition: { calories: 350 },
        tags: ['Quick'],
      });

    expect(res.status).toBe(201);
    expect(res.body.data.name).toBe('Test Recipe');
  });

  it('should return 403 for non-admin', async () => {
    const res = await request(app)
      .post('/api/v1/recipes')
      .set('Authorization', `Bearer ${userToken}`)
      .send({ name: 'Test' });

    expect(res.status).toBe(403);
  });
});
```

### Performance Tests

```typescript
describe('Recipe Search Performance', () => {
  it('should return results in < 300ms (P95)', async () => {
    const times = [];

    for (let i = 0; i < 100; i++) {
      const start = Date.now();
      await recipeService.searchRecipes('chicken');
      times.push(Date.now() - start);
    }

    const p95 = times.sort((a, b) => a - b)[94];
    expect(p95).toBeLessThan(300);
  });
});
```

---

## Acceptance Criteria

### Definition of Done

- [ ] All 12 recipe endpoints implemented and tested
- [ ] Full-text search working with relevance ranking
- [ ] Pagination implemented on list endpoints
- [ ] Filtering by category, dietary, cuisine, time
- [ ] Image upload to S3 with presigned URLs
- [ ] Image processing Lambda creates all variants
- [ ] Redis caching implemented for all read operations
- [ ] Favorites functionality complete
- [ ] Admin CRUD operations require admin role
- [ ] Test coverage > 85%
- [ ] API documentation updated
- [ ] Search response time P95 < 300ms

---

## Dependencies

### Upstream Dependencies (Blockers)

- Epic 1: Infrastructure (S3, CloudFront, Redis)
- Epic 2: Authentication (JWT middleware)

### Downstream Dependencies (Unblocks)

- Epic 4: Meal Planning (requires recipe data)
- Epic 5: Shopping List (requires recipe ingredients)
- Epic 6: AI Service (requires recipe catalog)

---

## Risks & Mitigation

### Risk 1: Full-Text Search Performance Degradation

**Impact:** High | **Probability:** Medium

**Mitigation:**
- Index optimization (GIN index on tsvector)
- Query result caching (Redis, 15 min TTL)
- Pagination (max 100 results per page)
- Consider Elasticsearch at 100K+ recipes

---

### Risk 2: S3 Image Upload Failures

**Impact:** Medium | **Probability:** Low

**Mitigation:**
- Presigned URL with 15-minute expiry
- Retry logic on frontend
- Upload progress indicator
- Fallback placeholder image

---

### Risk 3: Cache Invalidation Complexity

**Impact:** Medium | **Probability:** Medium

**Mitigation:**
- Clear strategy: invalidate on write operations
- Wildcard cache key patterns for easy invalidation
- Monitor cache hit rates (target > 80%)
- TTL fallback if invalidation fails

---

## Deliverables

### Code

- `src/modules/recipe/recipe.controller.ts`
- `src/modules/recipe/recipe.service.ts`
- `src/modules/recipe/recipe.repository.ts`
- `src/modules/recipe/recipe.search.ts` (search logic)
- `src/modules/recipe/recipe.validator.ts`
- `src/modules/recipe/recipe.routes.ts`
- `src/modules/recipe/image-upload.service.ts`
- `src/modules/recipe/favorites.service.ts`

### Lambda Functions

- `lambdas/image-processor/index.js` (resize images)
- `lambdas/image-processor/package.json` (sharp dependency)

### Tests

- `src/modules/recipe/__tests__/recipe.service.test.ts`
- `src/modules/recipe/__tests__/recipe.search.test.ts`
- `src/modules/recipe/__tests__/recipe.integration.test.ts`

---

## Timeline

### Week 3 Breakdown

| Day | Tasks | Hours |
|-----|-------|-------|
| **Monday** | Recipe CRUD endpoints, database queries | 10 |
| **Tuesday** | Full-text search, filtering, pagination | 10 |
| **Wednesday** | Image upload, S3 integration, Lambda | 10 |
| **Thursday** | Favorites, caching, optimization | 10 |
| **Friday** | Testing, bug fixes, documentation | 10 |

**Total:** 50 hours

---

## Success Metrics

- Recipe list API P95 response time < 200ms
- Search API P95 response time < 300ms
- Cache hit rate > 80%
- Image upload success rate > 98%
- Test coverage > 85%

---

**Epic Status:** Not Started
**Last Updated:** 2025-10-14
**Next Review:** End of Week 3

This epic provides the recipe catalog foundation for meal planning and shopping list generation.
