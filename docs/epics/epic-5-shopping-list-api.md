# Epic 5: Shopping List API
## Automated Shopping List Generation

---

**Epic ID:** EPIC-5
**Priority:** P1 (High)
**Estimated Effort:** 35 hours
**Sprint:** Week 5
**Owner:** Backend Developer
**Status:** Not Started
**Dependencies:** Epic 3 (Recipe Service), Epic 4 (Meal Planning)

---

## Overview

Build intelligent shopping list generation that consolidates ingredients from meal plans, organizes by category, and provides user-friendly shopping management features.

## Goals

1. Generate shopping lists from meal plans
2. Consolidate duplicate ingredients with quantity addition
3. Organize items by store category
4. Implement check/uncheck functionality
5. Allow manual item management
6. Add sharing capability

## User Stories

### US-5.1: Generate shopping list from meal plan

**Endpoint:** POST `/api/v1/shopping-lists/generate`

**Request:**
```json
{
  "mealPlanId": "plan-001"
}
```

**Algorithm:**
1. Fetch meal plan with all recipe IDs
2. Fetch full recipe data for all meals
3. Extract all ingredients
4. Consolidate duplicates (sum quantities)
5. Categorize by ingredient category
6. Sort within categories alphabetically
7. Create shopping list record

**Consolidation Logic:**
```typescript
function consolidateIngredients(recipes) {
  const consolidated = new Map();

  for (const recipe of recipes) {
    for (const ingredient of recipe.ingredients) {
      const key = ingredient.name.toLowerCase();

      if (consolidated.has(key)) {
        const existing = consolidated.get(key);
        existing.quantity = addQuantities(existing.quantity, ingredient.quantity, existing.unit);
        existing.recipeIds.push(recipe.id);
      } else {
        consolidated.set(key, {
          name: ingredient.name,
          quantity: ingredient.quantity,
          unit: ingredient.unit,
          category: ingredient.category,
          recipeIds: [recipe.id],
          checked: false,
        });
      }
    }
  }

  return Array.from(consolidated.values());
}
```

---

### US-5.2: View shopping list

**Endpoint:** GET `/api/v1/shopping-lists/:id`

**Response:**
```json
{
  "data": {
    "id": "list-001",
    "userId": "user-001",
    "mealPlanId": "plan-001",
    "weekStart": "2025-10-13",
    "items": [
      {
        "id": "item-001",
        "name": "Chicken breast",
        "quantity": "2.5",
        "unit": "lbs",
        "category": "Meat & Seafood",
        "checked": false,
        "recipeIds": ["recipe-001", "recipe-005"]
      },
      {
        "id": "item-002",
        "name": "Eggs",
        "quantity": "12",
        "unit": "",
        "category": "Dairy & Eggs",
        "checked": true,
        "recipeIds": ["recipe-003"]
      }
    ],
    "checkedCount": 1,
    "totalCount": 23
  }
}
```

---

### US-5.3: Check/uncheck items

**Endpoint:** PATCH `/api/v1/shopping-lists/:id/items/:itemId`

**Request:**
```json
{
  "checked": true
}
```

**Implementation:**
- Update item in JSONB array
- Recalculate checkedCount
- Return updated list

---

### US-5.4: Add manual items

**Endpoint:** POST `/api/v1/shopping-lists/:id/items`

**Request:**
```json
{
  "name": "Milk",
  "quantity": "1",
  "unit": "gallon",
  "category": "Dairy & Eggs"
}
```

---

### US-5.5: Delete items

**Endpoint:** DELETE `/api/v1/shopping-lists/:id/items/:itemId`

---

### US-5.6: Share shopping list

**Endpoint:** POST `/api/v1/shopping-lists/:id/share`

**Response:**
```json
{
  "data": {
    "shareId": "abc123xyz",
    "shareUrl": "https://app.mealplanner.com/shopping/abc123xyz"
  }
}
```

**Implementation:**
- Generate unique share ID (nanoid)
- Store in shopping_lists.share_id
- Public endpoint: GET `/api/v1/shopping-lists/shared/:shareId` (no auth)

---

## Technical Requirements

### Database Schema

```sql
CREATE TABLE shopping_lists (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  meal_plan_id UUID REFERENCES meal_plans(id) ON DELETE SET NULL,
  week_start DATE NOT NULL,
  items JSONB NOT NULL DEFAULT '[]'::jsonb,
  checked_count INTEGER DEFAULT 0,
  total_count INTEGER DEFAULT 0,
  share_id VARCHAR(50) UNIQUE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_shopping_lists_user_id ON shopping_lists(user_id);
CREATE INDEX idx_shopping_lists_share_id ON shopping_lists(share_id) WHERE share_id IS NOT NULL;
```

### Ingredient Categories

```typescript
const CATEGORIES = [
  'Produce',
  'Meat & Seafood',
  'Dairy & Eggs',
  'Bakery',
  'Pantry',
  'Canned & Jarred',
  'Frozen',
  'Beverages',
  'Condiments & Sauces',
  'Snacks',
  'Other',
];
```

---

## Testing Requirements

```typescript
describe('Shopping List Generation', () => {
  it('should generate list from meal plan', async () => {
    const mealPlan = await createMealPlanWithRecipes(userId);

    const res = await request(app)
      .post('/api/v1/shopping-lists/generate')
      .set('Authorization', `Bearer ${token}`)
      .send({ mealPlanId: mealPlan.id });

    expect(res.status).toBe(201);
    expect(res.body.data.items.length).toBeGreaterThan(0);
  });

  it('should consolidate duplicate ingredients', async () => {
    const recipe1 = { ingredients: [{ name: 'Chicken breast', quantity: '1', unit: 'lb' }] };
    const recipe2 = { ingredients: [{ name: 'Chicken breast', quantity: '1.5', unit: 'lbs' }] };

    const consolidated = consolidateIngredients([recipe1, recipe2]);

    expect(consolidated).toHaveLength(1);
    expect(consolidated[0].quantity).toBe('2.5');
  });

  it('should organize by category', async () => {
    const list = await generateShoppingList(mealPlanId);

    const categories = [...new Set(list.items.map(i => i.category))];
    expect(categories).toContain('Produce');
    expect(categories).toContain('Meat & Seafood');
  });
});
```

---

## API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/shopping-lists/generate` | Generate from meal plan |
| GET | `/shopping-lists` | List user's lists |
| GET | `/shopping-lists/:id` | Get list details |
| PUT | `/shopping-lists/:id` | Re-generate list |
| DELETE | `/shopping-lists/:id` | Delete list |
| PATCH | `/shopping-lists/:id/items/:itemId` | Check/uncheck item |
| POST | `/shopping-lists/:id/items` | Add manual item |
| DELETE | `/shopping-lists/:id/items/:itemId` | Delete item |
| POST | `/shopping-lists/:id/share` | Generate share link |
| GET | `/shopping-lists/shared/:shareId` | Public view (no auth) |

---

## Acceptance Criteria

- [ ] Shopping list generated from meal plan
- [ ] Duplicate ingredients consolidated correctly
- [ ] Items organized by category
- [ ] Check/uncheck updates checkedCount
- [ ] Manual items can be added/deleted
- [ ] Share link generates unique ID
- [ ] Public sharing works without auth
- [ ] Test coverage > 80%

---

## Timeline

| Day | Tasks | Hours |
|-----|-------|-------|
| **Monday** | Generation algorithm, consolidation | 8 |
| **Tuesday** | Category organization, CRUD | 7 |
| **Wednesday** | Check/uncheck, manual items | 7 |
| **Thursday** | Sharing functionality | 7 |
| **Friday** | Testing, optimization | 6 |

**Total:** 35 hours

---

**Epic Status:** Not Started
**Last Updated:** 2025-10-14
