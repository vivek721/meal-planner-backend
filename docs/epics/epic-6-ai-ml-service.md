# Epic 6: AI/ML Service
## Intelligent Meal Recommendations & Nutrition Analysis

---

**Epic ID:** EPIC-6
**Priority:** P1 (High)
**Estimated Effort:** 60 hours
**Sprint:** Week 6
**Owner:** Backend Developer + ML Engineer
**Status:** Not Started
**Dependencies:** Epic 3 (Recipe Service), Epic 4 (Meal Planning)

---

## Overview

Replace mock AI recommendations with real machine learning-powered suggestions using AWS Personalize. Implement nutrition analysis and ingredient substitution features.

## Goals

1. Set up AWS Personalize for meal recommendations
2. Track user interactions for model training
3. Implement recommendation API with fallback
4. Build nutrition balance analyzer
5. Create ingredient substitution engine
6. Implement A/B testing framework

## User Stories

### US-6.1: Track user interactions for ML training

**Events to Track:**
- `meal_planned`: User adds recipe to meal plan
- `recipe_favorited`: User favorites recipe
- `recipe_viewed`: User views recipe detail
- `shopping_list_generated`: Indicates meal plan completion

**Implementation:**
```typescript
async trackInteraction(userId: string, recipeId: string, eventType: string) {
  // Store in activities table
  await db.activities.create({
    userId,
    action: eventType,
    relatedId: recipeId,
    relatedType: 'recipe',
    timestamp: new Date(),
  });

  // Send to AWS Personalize (async)
  await personalizeQueue.add({
    userId,
    itemId: recipeId,
    eventType,
    timestamp: Date.now(),
  });
}
```

---

### US-6.2: Generate personalized meal suggestions

**Endpoint:** POST `/api/v1/ai/suggestions`

**Request:**
```json
{
  "mealType": "dinner",
  "date": "2025-10-15",
  "count": 5,
  "excludeRecipeIds": ["recipe-001", "recipe-002"]
}
```

**Response:**
```json
{
  "data": {
    "suggestions": [
      {
        "recipeId": "recipe-010",
        "recipe": {
          "id": "recipe-010",
          "name": "Thai Green Curry",
          "imageUrl": "...",
          "prepTime": 20,
          "rating": 4.7
        },
        "score": 0.92,
        "reason": "Based on your preference for Asian cuisine"
      }
    ],
    "generatedAt": "2025-10-14T11:35:00Z"
  }
}
```

**Recommendation Algorithm (Hybrid):**
1. **AWS Personalize** (primary): User-personalization recipe
2. **Rule-based** (fallback):
   - Filter by user dietary preferences
   - Match cuisine from favorites
   - Time-appropriate (breakfast in AM, dinner in PM)
   - Exclude recently planned (last 2 weeks)
   - Sort by popularity (rating × review_count)

---

### US-6.3: Setup AWS Personalize

**Steps:**
1. Create dataset group: `meal-planner-recommendations`
2. Create schemas:
   - Users schema (userId, preferences, allergies)
   - Items schema (recipeId, category, tags, nutrition)
   - Interactions schema (userId, recipeId, eventType, timestamp)
3. Import historical data (initial: 10K interactions)
4. Train solution:
   - Recipe: `aws-user-personalization` (collaborative filtering)
   - Backup: `aws-sims` (similar items)
   - Fallback: `aws-popularity-count`
5. Create campaign for real-time inference

**AWS Personalize API Call:**
```typescript
import { PersonalizeRuntime } from '@aws-sdk/client-personalize-runtime';

async function getRecommendations(userId: string, count: number) {
  const client = new PersonalizeRuntime({ region: 'us-east-1' });

  const response = await client.getRecommendations({
    campaignArn: process.env.PERSONALIZE_CAMPAIGN_ARN,
    userId,
    numResults: count,
  });

  return response.itemList.map(item => ({
    recipeId: item.itemId,
    score: item.score,
  }));
}
```

---

### US-6.4: Analyze nutrition balance

**Endpoint:** POST `/api/v1/ai/nutrition-analysis`

**Request:**
```json
{
  "mealPlanId": "plan-001"
}
```

**Response:**
```json
{
  "data": {
    "summary": {
      "avgCaloriesPerDay": 1850,
      "avgProteinPerDay": 95,
      "avgCarbsPerDay": 185,
      "avgFatPerDay": 62
    },
    "balanceScore": 85,
    "insights": [
      {
        "type": "positive",
        "message": "Great protein balance!"
      },
      {
        "type": "suggestion",
        "message": "Add more vegetables on Thursday"
      }
    ],
    "macroDistribution": {
      "protein": 21,
      "carbohydrates": 40,
      "fat": 30
    },
    "variety": {
      "uniqueMeals": 18,
      "varietyScore": 92
    }
  }
}
```

**Balance Score Algorithm:**
```typescript
function calculateBalanceScore(weeklyNutrition) {
  let score = 100;

  // 1. Consistency (low variance in daily calories)
  const calorieStdDev = calculateStdDev(dailyCalories);
  if (calorieStdDev > 500) score -= 20;
  else if (calorieStdDev > 300) score -= 10;

  // 2. Macro balance (ideal: 30% protein, 40% carbs, 30% fat)
  const macroScore = scoreMacroDistribution(avgMacros);
  score = score * 0.6 + macroScore * 0.4;

  // 3. Variety (unique meals)
  const uniqueMeals = new Set(recipeIds).size;
  if (uniqueMeals < 7) score -= 10;
  if (uniqueMeals < 5) score -= 20;

  // 4. Calorie range (1500-2500 for average adult)
  const avgCalories = weeklyNutrition.avgCaloriesPerDay;
  if (avgCalories < 1200 || avgCalories > 3000) score -= 30;
  else if (avgCalories < 1500 || avgCalories > 2500) score -= 15;

  return Math.max(0, Math.min(100, score));
}
```

---

### US-6.5: Suggest ingredient substitutions

**Endpoint:** POST `/api/v1/ai/substitutions`

**Request:**
```json
{
  "ingredient": "chicken breast",
  "reason": "dietary",
  "dietary": ["Vegan"]
}
```

**Response:**
```json
{
  "data": {
    "original": "chicken breast",
    "substitutions": [
      {
        "ingredient": "extra-firm tofu",
        "ratio": "1:1",
        "reason": "High protein, similar texture when pressed",
        "nutritionImpact": {
          "calories": -30,
          "protein": -8,
          "fat": +2
        }
      },
      {
        "ingredient": "tempeh",
        "ratio": "1:1",
        "reason": "Nutty flavor, excellent protein source"
      }
    ]
  }
}
```

**Substitution Database:**
```typescript
const SUBSTITUTIONS = {
  'chicken breast': {
    vegan: ['extra-firm tofu', 'tempeh', 'seitan', 'chickpeas'],
    'gluten-free': ['chicken breast'], // No sub needed
  },
  'milk': {
    vegan: ['almond milk', 'oat milk', 'soy milk', 'coconut milk'],
    'lactose-free': ['lactose-free milk', 'almond milk', 'oat milk'],
  },
  // ... 100+ common ingredients
};
```

---

## Technical Requirements

### AWS Personalize Setup

**Dataset Schemas:**

Users schema:
```json
{
  "type": "record",
  "name": "Users",
  "namespace": "com.mealplanner",
  "fields": [
    {"name": "USER_ID", "type": "string"},
    {"name": "DIETARY_PREFERENCES", "type": "string"},
    {"name": "ALLERGIES", "type": "string"}
  ]
}
```

Items schema:
```json
{
  "type": "record",
  "name": "Items",
  "fields": [
    {"name": "ITEM_ID", "type": "string"},
    {"name": "CATEGORY", "type": "string"},
    {"name": "TAGS", "type": "string"},
    {"name": "PREP_TIME", "type": "int"}
  ]
}
```

Interactions schema:
```json
{
  "type": "record",
  "name": "Interactions",
  "fields": [
    {"name": "USER_ID", "type": "string"},
    {"name": "ITEM_ID", "type": "string"},
    {"name": "EVENT_TYPE", "type": "string"},
    {"name": "TIMESTAMP", "type": "long"}
  ]
}
```

### Event Tracking Pipeline

```typescript
// Queue worker to sync events to Personalize
class PersonalizeEventWorker {
  async processEvent(event) {
    const client = new PersonalizeEvents({ region: 'us-east-1' });

    await client.putEvents({
      trackingId: process.env.PERSONALIZE_TRACKING_ID,
      userId: event.userId,
      sessionId: event.sessionId,
      eventList: [
        {
          eventId: event.id,
          eventType: event.eventType,
          itemId: event.recipeId,
          sentAt: new Date(event.timestamp),
        },
      ],
    });
  }
}
```

---

## Testing Requirements

```typescript
describe('AI Recommendations', () => {
  it('should return personalized suggestions', async () => {
    const res = await request(app)
      .post('/api/v1/ai/suggestions')
      .set('Authorization', `Bearer ${token}`)
      .send({
        mealType: 'dinner',
        count: 5,
      });

    expect(res.status).toBe(200);
    expect(res.body.data.suggestions).toHaveLength(5);
    expect(res.body.data.suggestions[0].score).toBeGreaterThan(0);
  });

  it('should fallback to rule-based when Personalize unavailable', async () => {
    // Mock Personalize failure
    mockPersonalizeClient.getRecommendations.mockRejectedValue(new Error('Service unavailable'));

    const res = await request(app)
      .post('/api/v1/ai/suggestions')
      .set('Authorization', `Bearer ${token}`)
      .send({ mealType: 'dinner', count: 5 });

    expect(res.status).toBe(200);
    expect(res.body.data.suggestions).toHaveLength(5);
  });
});

describe('Nutrition Analysis', () => {
  it('should calculate balance score', () => {
    const nutrition = {
      dailyCalories: [1800, 1900, 1850, 1900, 1800, 2000, 1850],
      avgProtein: 95,
      avgCarbs: 185,
      avgFat: 62,
    };

    const score = calculateBalanceScore(nutrition);
    expect(score).toBeGreaterThan(80);
  });

  it('should provide actionable insights', async () => {
    const res = await request(app)
      .post('/api/v1/ai/nutrition-analysis')
      .set('Authorization', `Bearer ${token}`)
      .send({ mealPlanId: planId });

    expect(res.body.data.insights).toBeInstanceOf(Array);
    expect(res.body.data.insights.length).toBeGreaterThan(0);
  });
});
```

---

## API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/ai/suggestions` | Get personalized meal suggestions |
| POST | `/ai/nutrition-analysis` | Analyze meal plan nutrition |
| POST | `/ai/substitutions` | Get ingredient substitutions |

---

## Acceptance Criteria

- [ ] AWS Personalize configured and trained
- [ ] Event tracking pipeline operational
- [ ] Recommendation endpoint returns suggestions in < 2s
- [ ] Fallback to rule-based when Personalize unavailable
- [ ] Nutrition analysis calculates balance score
- [ ] Substitution database covers 100+ ingredients
- [ ] A/B testing framework for model comparison
- [ ] Test coverage > 75%

---

## Timeline

| Day | Tasks | Hours |
|-----|-------|-------|
| **Mon-Tue** | AWS Personalize setup, dataset import, training | 16 |
| **Wed** | Recommendation API, fallback logic | 12 |
| **Thu** | Nutrition analysis algorithm | 12 |
| **Fri** | Substitution engine, testing | 12 |
| **Weekend** | A/B testing, optimization | 8 |

**Total:** 60 hours

---

## Success Metrics

- Recommendation relevance (user acceptance rate > 30%)
- API response time < 2 seconds (P95)
- Fallback rate < 5%
- Nutrition insights accuracy > 90%

---

**Epic Status:** Not Started
**Last Updated:** 2025-10-14
