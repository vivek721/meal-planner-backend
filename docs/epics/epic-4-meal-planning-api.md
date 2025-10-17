# Epic 4: Meal Planning API
## Weekly Meal Plan Management

---

**Epic ID:** EPIC-4
**Priority:** P0 (Critical)
**Estimated Effort:** 40 hours
**Sprint:** Week 4
**Owner:** Backend Developer
**Status:** Not Started
**Dependencies:** Epic 2 (Authentication), Epic 3 (Recipe Service)

---

## Overview

Build the meal planning API that allows users to create, manage, and organize weekly meal plans. Replace frontend's localStorage meal plans with database-backed persistence and real recipe integration.

## Goals

1. Implement meal plan CRUD operations
2. Add/remove meals from specific day/slot combinations
3. Build copy day functionality
4. Calculate nutritional summaries
5. Implement caching for meal plan data
6. Ensure one meal plan per user per week (unique constraint)

## User Stories

### US-4.1: As a user, I can create a meal plan for a specific week

**Acceptance Criteria:**
- POST `/api/v1/meal-plans` creates new plan
- `weekStart` must be a Sunday (validation)
- Unique constraint: one plan per user per week
- Empty meal structure initialized (7 days × 4 slots)
- Returns 201 Created with plan ID

**Request:**
```json
{
  "weekStart": "2025-10-13"
}
```

**Response:**
```json
{
  "data": {
    "id": "plan-001",
    "userId": "user-001",
    "weekStart": "2025-10-13",
    "meals": {
      "sunday": { "breakfast": null, "lunch": null, "dinner": null, "snacks": [] },
      "monday": { "breakfast": null, "lunch": null, "dinner": null, "snacks": [] },
      ...
    },
    "isActive": true,
    "createdAt": "2025-10-13T08:00:00Z"
  }
}
```

---

### US-4.2: As a user, I can view my meal plans

**Acceptance Criteria:**
- GET `/api/v1/meal-plans?weekStart=2025-10-13` returns specific week
- GET `/api/v1/meal-plans` returns all user's plans (paginated)
- GET `/api/v1/meal-plans/:id` returns single plan with populated recipes
- Recipe data hydrated from recipe service
- Cached in Redis (10 min TTL)

**Response (populated):**
```json
{
  "data": {
    "id": "plan-001",
    "weekStart": "2025-10-13",
    "meals": {
      "monday": {
        "breakfast": {
          "id": "recipe-003",
          "name": "Overnight Oats",
          "imageUrl": "...",
          "prepTime": 10,
          "nutrition": { "calories": 320 }
        },
        "dinner": {
          "id": "recipe-005",
          "name": "Grilled Salmon",
          "imageUrl": "...",
          "prepTime": 10,
          "cookTime": 15,
          "nutrition": { "calories": 420 }
        }
      }
    },
    "summary": {
      "totalMeals": 15,
      "avgCaloriesPerDay": 1850,
      "avgProteinPerDay": 95
    }
  }
}
```

---

### US-4.3: As a user, I can add meals to specific slots

**Acceptance Criteria:**
- POST `/api/v1/meal-plans/:id/meals` adds meal to day/slot
- Request includes: `day`, `mealType`, `recipeId`
- Validates recipe exists before adding
- Updates `updated_at` timestamp
- Invalidates meal plan cache

**Request:**
```json
{
  "day": "monday",
  "mealType": "dinner",
  "recipeId": "recipe-001"
}
```

---

### US-4.4: As a user, I can remove meals from slots

**Acceptance Criteria:**
- DELETE `/api/v1/meal-plans/:id/meals` removes meal
- Request includes: `day`, `mealType`
- Sets slot to null
- Invalidates meal plan cache

---

### US-4.5: As a user, I can copy meals from one day to another

**Acceptance Criteria:**
- POST `/api/v1/meal-plans/:id/copy-day` copies all meals
- Request includes: `sourceDay`, `targetDay`
- Overwrites existing meals in target day
- Useful for repeating meal patterns

**Request:**
```json
{
  "sourceDay": "monday",
  "targetDay": "tuesday"
}
```

---

### US-4.6: As a user, I can update entire meal plan

**Acceptance Criteria:**
- PUT `/api/v1/meal-plans/:id` updates meal structure
- Allows bulk updates to meals object
- Validates all recipe IDs exist
- Returns updated plan

---

### US-4.7: As a user, I can delete a meal plan

**Acceptance Criteria:**
- DELETE `/api/v1/meal-plans/:id` removes plan
- Cascade deletes related shopping lists
- Returns 204 No Content
- Invalidates all plan caches

---

### US-4.8: As a system, I calculate nutritional summaries

**Acceptance Criteria:**
- Calculate total calories, protein, carbs, fat per day
- Calculate weekly averages
- Include in meal plan response
- Handle missing nutrition data gracefully

**Summary Calculation:**
```typescript
function calculateSummary(mealPlan) {
  const dailyTotals = {};

  for (const [day, meals] of Object.entries(mealPlan.meals)) {
    let dailyCalories = 0;
    let dailyProtein = 0;

    for (const meal of Object.values(meals)) {
      if (meal?.nutrition) {
        dailyCalories += meal.nutrition.calories || 0;
        dailyProtein += meal.nutrition.protein || 0;
      }
    }

    dailyTotals[day] = { calories: dailyCalories, protein: dailyProtein };
  }

  const avgCalories = Object.values(dailyTotals).reduce((sum, d) => sum + d.calories, 0) / 7;
  const avgProtein = Object.values(dailyTotals).reduce((sum, d) => sum + d.protein, 0) / 7;

  return {
    dailyTotals,
    avgCaloriesPerDay: Math.round(avgCalories),
    avgProteinPerDay: Math.round(avgProtein),
    totalMeals: countMeals(mealPlan),
  };
}
```

---

## Technical Requirements

### Database Schema

**meal_plans** table:
```sql
CREATE TABLE meal_plans (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  week_start DATE NOT NULL,
  meals JSONB NOT NULL DEFAULT '{}'::jsonb,
  is_active BOOLEAN NOT NULL DEFAULT TRUE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

  CONSTRAINT unique_user_week UNIQUE (user_id, week_start),
  CONSTRAINT check_week_start_sunday CHECK (EXTRACT(DOW FROM week_start) = 0)
);

CREATE INDEX idx_meal_plans_user_id ON meal_plans(user_id);
CREATE INDEX idx_meal_plans_week_start ON meal_plans(week_start);
CREATE INDEX idx_meal_plans_user_week ON meal_plans(user_id, week_start);
```

### Caching Strategy

| Resource | Cache Key | TTL | Invalidation |
|----------|-----------|-----|--------------|
| Meal plan | `meal-plan:{userId}:{weekStart}` | 10 min | Meal added/removed/updated |
| Meal plan list | `meal-plans:{userId}` | 5 min | Plan created/deleted |

### API Endpoints

| Method | Endpoint | Auth | Description |
|--------|----------|------|-------------|
| GET | `/meal-plans` | Required | List user's plans |
| GET | `/meal-plans/:id` | Required | Get plan details |
| POST | `/meal-plans` | Required | Create plan |
| PUT | `/meal-plans/:id` | Required | Update plan |
| DELETE | `/meal-plans/:id` | Required | Delete plan |
| POST | `/meal-plans/:id/meals` | Required | Add meal to slot |
| DELETE | `/meal-plans/:id/meals` | Required | Remove meal from slot |
| POST | `/meal-plans/:id/copy-day` | Required | Copy day meals |

---

## Testing Requirements

### Integration Tests

```typescript
describe('Meal Planning API', () => {
  let userId, token, planId;

  beforeEach(async () => {
    const user = await createTestUser();
    userId = user.id;
    token = user.token;
  });

  describe('POST /meal-plans', () => {
    it('should create meal plan for new week', async () => {
      const res = await request(app)
        .post('/api/v1/meal-plans')
        .set('Authorization', `Bearer ${token}`)
        .send({ weekStart: '2025-10-13' });

      expect(res.status).toBe(201);
      expect(res.body.data.weekStart).toBe('2025-10-13');
      planId = res.body.data.id;
    });

    it('should enforce unique constraint per week', async () => {
      await createMealPlan(userId, '2025-10-13');

      const res = await request(app)
        .post('/api/v1/meal-plans')
        .set('Authorization', `Bearer ${token}`)
        .send({ weekStart: '2025-10-13' });

      expect(res.status).toBe(409);
    });

    it('should validate weekStart is Sunday', async () => {
      const res = await request(app)
        .post('/api/v1/meal-plans')
        .set('Authorization', `Bearer ${token}`)
        .send({ weekStart: '2025-10-14' }); // Monday

      expect(res.status).toBe(400);
    });
  });

  describe('POST /meal-plans/:id/meals', () => {
    beforeEach(async () => {
      const plan = await createMealPlan(userId, '2025-10-13');
      planId = plan.id;
    });

    it('should add meal to slot', async () => {
      const recipe = await createTestRecipe();

      const res = await request(app)
        .post(`/api/v1/meal-plans/${planId}/meals`)
        .set('Authorization', `Bearer ${token}`)
        .send({
          day: 'monday',
          mealType: 'dinner',
          recipeId: recipe.id,
        });

      expect(res.status).toBe(200);
      expect(res.body.data.meals.monday.dinner).toBe(recipe.id);
    });

    it('should validate recipe exists', async () => {
      const res = await request(app)
        .post(`/api/v1/meal-plans/${planId}/meals`)
        .set('Authorization', `Bearer ${token}`)
        .send({
          day: 'monday',
          mealType: 'dinner',
          recipeId: 'non-existent-recipe',
        });

      expect(res.status).toBe(404);
    });
  });

  describe('POST /meal-plans/:id/copy-day', () => {
    it('should copy all meals from one day to another', async () => {
      const plan = await createMealPlan(userId, '2025-10-13');
      const recipe = await createTestRecipe();

      await addMealToSlot(plan.id, 'monday', 'dinner', recipe.id);

      const res = await request(app)
        .post(`/api/v1/meal-plans/${plan.id}/copy-day`)
        .set('Authorization', `Bearer ${token}`)
        .send({
          sourceDay: 'monday',
          targetDay: 'tuesday',
        });

      expect(res.status).toBe(200);
      expect(res.body.data.meals.tuesday.dinner).toBe(recipe.id);
    });
  });
});
```

---

## Acceptance Criteria

### Definition of Done

- [ ] All 8 meal planning endpoints implemented
- [ ] Unique constraint enforced (one plan per user per week)
- [ ] Recipe validation before adding to slot
- [ ] Copy day functionality working
- [ ] Nutritional summary calculation
- [ ] Redis caching implemented
- [ ] Test coverage > 85%
- [ ] API documentation updated
- [ ] Frontend integration tested

---

## Timeline

| Day | Tasks | Hours |
|-----|-------|-------|
| **Monday** | Meal plan CRUD, database schema | 8 |
| **Tuesday** | Add/remove meals, validation | 8 |
| **Wednesday** | Copy day, nutritional summary | 8 |
| **Thursday** | Caching, optimization | 8 |
| **Friday** | Testing, bug fixes | 8 |

**Total:** 40 hours

---

## Success Metrics

- API response time P95 < 200ms
- Cache hit rate > 75%
- Test coverage > 85%
- Zero data loss on concurrent updates

---

**Epic Status:** Not Started
**Last Updated:** 2025-10-14
**Next Review:** End of Week 4
