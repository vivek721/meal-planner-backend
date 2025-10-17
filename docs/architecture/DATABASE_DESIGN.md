# Database Design Document
## AI-Powered Meal Planner

---

**Version:** 1.0
**Last Updated:** 2025-10-14
**Database:** PostgreSQL 15+

---

## Table of Contents

1. [Entity Relationship Diagram](#entity-relationship-diagram)
2. [Schema Overview](#schema-overview)
3. [Table Specifications](#table-specifications)
4. [Indexes & Constraints](#indexes--constraints)
5. [Data Migration Strategy](#data-migration-strategy)
6. [Backup & Recovery](#backup--recovery)
7. [Performance Optimization](#performance-optimization)

---

## Entity Relationship Diagram

```
┌─────────────────┐         ┌──────────────────┐
│     users       │         │     recipes      │
├─────────────────┤         ├──────────────────┤
│ id (PK)         │         │ id (PK)          │
│ email (UNIQUE)  │         │ name             │
│ password_hash   │         │ description      │
│ name            │         │ category         │
│ role            │         │ prep_time        │
│ preferences     │         │ cook_time        │
│ created_at      │         │ servings         │
│ updated_at      │         │ difficulty       │
│ deleted_at      │         │ ingredients      │
└────────┬────────┘         │ instructions     │
         │                  │ nutrition        │
         │                  │ tags             │
         │                  │ image_url        │
         │                  │ rating           │
         │                  │ review_count     │
         │                  │ created_by_id    │
         │                  │ created_at       │
         │                  │ updated_at       │
         │                  │ search_vector    │
         │                  └─────────┬────────┘
         │                            │
         │  1:N favorites             │ 1:N
         │  ┌──────────────┐          │
         │  │  favorites   │          │ created_by
         │  ├──────────────┤          │
         ├──┤ user_id (FK) │          │
         │  │ recipe_id(FK)├──────────┘
         │  │ created_at   │
         │  └──────────────┘
         │
         │  1:N meal_plans
         │  ┌───────────────┐
         │  │  meal_plans   │
         │  ├───────────────┤
         ├──┤ user_id (FK)  │
         │  │ id (PK)       │
         │  │ week_start    │
         │  │ meals (JSONB) │───┐ References recipes
         │  │ created_at    │   │
         │  │ updated_at    │   │
         │  └───────┬───────┘   │
         │          │            │
         │          │ 1:N        │
         │          │            │
         │  ┌───────┴──────────┐│
         │  │ shopping_lists   ││
         │  ├──────────────────┤│
         ├──┤ user_id (FK)     ││
         │  │ meal_plan_id (FK)│┘
         │  │ id (PK)          │
         │  │ week_start       │
         │  │ items (JSONB)    │───┐ References recipes
         │  │ created_at       │   │
         │  │ updated_at       │   │
         │  └──────────────────┘   │
         │                          │
         │  1:N activities          │
         │  ┌──────────────────┐   │
         │  │   activities     │   │
         │  ├──────────────────┤   │
         └──┤ user_id (FK)     │   │
            │ id (PK)          │   │
            │ action           │   │
            │ details          │   │
            │ related_id       │───┘ Ref to recipe/plan
            │ timestamp        │
            └──────────────────┘
```

---

## Schema Overview

### Tables

| Table | Description | Est. Rows (Year 1) | Est. Rows (Year 5) |
|-------|-------------|--------------------|--------------------|
| `users` | User accounts, profiles, preferences | 10,000 | 1,000,000 |
| `recipes` | Recipe details, ingredients, nutrition | 1,000 | 100,000 |
| `favorites` | User-favorited recipes (junction table) | 50,000 | 5,000,000 |
| `meal_plans` | Weekly meal plans | 50,000 | 10,000,000 |
| `shopping_lists` | Generated shopping lists | 30,000 | 5,000,000 |
| `activities` | User activity log | 500,000 | 100,000,000 |
| `sessions` | Active JWT tokens (blacklist) | 10,000 | 100,000 |

### Database Size Estimation

**Year 1**:
- Users: 10,000 × 2 KB = 20 MB
- Recipes: 1,000 × 5 KB = 5 MB
- Favorites: 50,000 × 0.1 KB = 5 MB
- Meal Plans: 50,000 × 2 KB = 100 MB
- Shopping Lists: 30,000 × 3 KB = 90 MB
- Activities: 500,000 × 0.5 KB = 250 MB
- **Total: ~500 MB** (indexes add ~50%, total ~750 MB)

**Year 5**:
- **Total: ~50 GB** (raw data), ~75 GB with indexes

---

## Table Specifications

### Table: `users`

**Description**: User accounts, authentication, profiles, preferences.

```sql
CREATE TABLE users (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  email VARCHAR(255) NOT NULL UNIQUE,
  password_hash VARCHAR(255) NOT NULL,
  name VARCHAR(255) NOT NULL,
  role VARCHAR(50) NOT NULL DEFAULT 'user', -- 'user', 'admin', 'moderator'

  -- User preferences (JSONB for flexibility)
  preferences JSONB NOT NULL DEFAULT '{
    "dietary": [],
    "allergies": [],
    "householdSize": 2,
    "onboardingCompleted": false
  }'::jsonb,

  -- Profile info
  avatar_url TEXT,
  bio TEXT,

  -- Timestamps
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  deleted_at TIMESTAMPTZ, -- Soft delete for GDPR

  -- Session tracking
  last_login_at TIMESTAMPTZ,
  email_verified BOOLEAN NOT NULL DEFAULT FALSE,
  email_verification_token VARCHAR(255),

  -- Constraints
  CONSTRAINT check_role CHECK (role IN ('user', 'admin', 'moderator')),
  CONSTRAINT check_email_format CHECK (email ~* '^[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}$')
);

-- Indexes
CREATE INDEX idx_users_email ON users(email) WHERE deleted_at IS NULL;
CREATE INDEX idx_users_created_at ON users(created_at);
CREATE INDEX idx_users_deleted_at ON users(deleted_at) WHERE deleted_at IS NOT NULL;

-- Trigger: Auto-update updated_at
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
  NEW.updated_at = NOW();
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_users_updated_at
BEFORE UPDATE ON users
FOR EACH ROW
EXECUTE FUNCTION update_updated_at_column();
```

**Sample Data**:
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "email": "sarah@example.com",
  "password_hash": "$2b$12$...",
  "name": "Sarah Johnson",
  "role": "user",
  "preferences": {
    "dietary": ["Vegan", "Gluten-Free"],
    "allergies": ["Peanuts"],
    "householdSize": 2,
    "onboardingCompleted": true
  },
  "avatar_url": "https://cdn.mealplanner.com/users/550e8400.../avatar.webp",
  "created_at": "2025-09-15T10:00:00Z",
  "updated_at": "2025-10-12T14:30:00Z"
}
```

---

### Table: `recipes`

**Description**: Recipe details including ingredients, instructions, nutrition.

```sql
CREATE TABLE recipes (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name VARCHAR(255) NOT NULL,
  description TEXT,
  category VARCHAR(50) NOT NULL, -- 'Breakfast', 'Lunch', 'Dinner', 'Snacks', 'Desserts'
  cuisine VARCHAR(100), -- 'Italian', 'Mexican', 'Chinese', etc.

  -- Timing
  prep_time INTEGER NOT NULL, -- Minutes
  cook_time INTEGER NOT NULL, -- Minutes
  total_time INTEGER GENERATED ALWAYS AS (prep_time + cook_time) STORED,

  -- Servings & difficulty
  servings INTEGER NOT NULL DEFAULT 4,
  difficulty VARCHAR(20) NOT NULL DEFAULT 'Medium', -- 'Easy', 'Medium', 'Hard'

  -- Ingredients (JSONB array)
  ingredients JSONB NOT NULL DEFAULT '[]'::jsonb,
  -- Format: [{ "name": "Chicken breast", "quantity": "2", "unit": "lbs", "category": "Meat" }]

  -- Instructions (JSONB array)
  instructions JSONB NOT NULL DEFAULT '[]'::jsonb,
  -- Format: ["Step 1: ...", "Step 2: ..."]

  -- Nutrition (JSONB object, per serving)
  nutrition JSONB NOT NULL DEFAULT '{}'::jsonb,
  -- Format: { "calories": 450, "protein": 35, "carbohydrates": 30, "fat": 18, "fiber": 5 }

  -- Tags (JSONB array)
  tags JSONB NOT NULL DEFAULT '[]'::jsonb,
  -- Format: ["Vegan", "Keto", "Gluten-Free", "Quick"]

  -- Images
  image_url TEXT, -- Primary image (medium size)
  image_urls JSONB, -- All variants: { "original": "...", "large": "...", "medium": "...", "thumbnail": "..." }

  -- Ratings & reviews
  rating DECIMAL(3, 2) DEFAULT 0.0, -- 0.00 to 5.00
  review_count INTEGER DEFAULT 0,

  -- Authorship
  created_by_id UUID REFERENCES users(id) ON DELETE SET NULL,
  is_public BOOLEAN NOT NULL DEFAULT TRUE,
  is_featured BOOLEAN NOT NULL DEFAULT FALSE,

  -- Full-text search vector
  search_vector tsvector GENERATED ALWAYS AS (
    to_tsvector('english',
      coalesce(name, '') || ' ' ||
      coalesce(description, '') || ' ' ||
      coalesce(cuisine, '') || ' ' ||
      coalesce(tags::text, '') || ' ' ||
      coalesce(ingredients::text, '')
    )
  ) STORED,

  -- Timestamps
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  deleted_at TIMESTAMPTZ,

  -- Constraints
  CONSTRAINT check_category CHECK (category IN ('Breakfast', 'Lunch', 'Dinner', 'Snacks', 'Desserts')),
  CONSTRAINT check_difficulty CHECK (difficulty IN ('Easy', 'Medium', 'Hard')),
  CONSTRAINT check_rating CHECK (rating >= 0 AND rating <= 5),
  CONSTRAINT check_times CHECK (prep_time >= 0 AND cook_time >= 0)
);

-- Indexes
CREATE INDEX idx_recipes_category ON recipes(category) WHERE deleted_at IS NULL;
CREATE INDEX idx_recipes_created_at ON recipes(created_at);
CREATE INDEX idx_recipes_rating ON recipes(rating DESC) WHERE deleted_at IS NULL;
CREATE INDEX idx_recipes_search_vector ON recipes USING GIN(search_vector);
CREATE INDEX idx_recipes_tags ON recipes USING GIN(tags);
CREATE INDEX idx_recipes_total_time ON recipes(total_time) WHERE deleted_at IS NULL;
CREATE INDEX idx_recipes_created_by ON recipes(created_by_id);

-- Trigger
CREATE TRIGGER update_recipes_updated_at
BEFORE UPDATE ON recipes
FOR EACH ROW
EXECUTE FUNCTION update_updated_at_column();
```

**Sample Data**:
```json
{
  "id": "recipe-001",
  "name": "Chicken Tacos",
  "description": "Delicious and easy chicken tacos with fresh toppings",
  "category": "Dinner",
  "cuisine": "Mexican",
  "prep_time": 15,
  "cook_time": 20,
  "servings": 4,
  "difficulty": "Easy",
  "ingredients": [
    { "name": "Chicken breast", "quantity": "1", "unit": "lb", "category": "Meat" },
    { "name": "Taco seasoning", "quantity": "2", "unit": "tbsp", "category": "Pantry" },
    { "name": "Tortillas", "quantity": "8", "unit": "", "category": "Bakery" }
  ],
  "instructions": [
    "Season chicken with taco seasoning",
    "Cook chicken in skillet for 20 minutes",
    "Shred chicken and serve in tortillas with toppings"
  ],
  "nutrition": {
    "calories": 350,
    "protein": 28,
    "carbohydrates": 35,
    "fat": 10,
    "fiber": 5
  },
  "tags": ["Mexican", "Quick", "Family-Friendly"],
  "image_url": "https://cdn.mealplanner.com/recipes/recipe-001/medium.webp",
  "rating": 4.5,
  "review_count": 89,
  "is_public": true,
  "created_at": "2025-08-01T10:00:00Z"
}
```

---

### Table: `favorites`

**Description**: Junction table for user-favorited recipes.

```sql
CREATE TABLE favorites (
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  recipe_id UUID NOT NULL REFERENCES recipes(id) ON DELETE CASCADE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

  PRIMARY KEY (user_id, recipe_id)
);

-- Indexes
CREATE INDEX idx_favorites_user_id ON favorites(user_id);
CREATE INDEX idx_favorites_recipe_id ON favorites(recipe_id);
CREATE INDEX idx_favorites_created_at ON favorites(created_at);
```

---

### Table: `meal_plans`

**Description**: Weekly meal plans for users.

```sql
CREATE TABLE meal_plans (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  week_start DATE NOT NULL, -- Always a Sunday

  -- Meals structure (JSONB)
  meals JSONB NOT NULL DEFAULT '{}'::jsonb,
  /* Format:
  {
    "sunday": {
      "breakfast": "recipe-id-1",
      "lunch": "recipe-id-2",
      "dinner": "recipe-id-3",
      "snacks": ["recipe-id-4"]
    },
    "monday": { ... },
    ...
  }
  */

  -- Metadata
  is_active BOOLEAN NOT NULL DEFAULT TRUE,

  -- Timestamps
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

  -- Constraints
  CONSTRAINT unique_user_week UNIQUE (user_id, week_start),
  CONSTRAINT check_week_start_sunday CHECK (EXTRACT(DOW FROM week_start) = 0)
);

-- Indexes
CREATE INDEX idx_meal_plans_user_id ON meal_plans(user_id);
CREATE INDEX idx_meal_plans_week_start ON meal_plans(week_start);
CREATE INDEX idx_meal_plans_user_week ON meal_plans(user_id, week_start);

-- Trigger
CREATE TRIGGER update_meal_plans_updated_at
BEFORE UPDATE ON meal_plans
FOR EACH ROW
EXECUTE FUNCTION update_updated_at_column();
```

---

### Table: `shopping_lists`

**Description**: Generated shopping lists from meal plans.

```sql
CREATE TABLE shopping_lists (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  meal_plan_id UUID REFERENCES meal_plans(id) ON DELETE SET NULL,
  week_start DATE NOT NULL,

  -- Items (JSONB array)
  items JSONB NOT NULL DEFAULT '[]'::jsonb,
  /* Format:
  [
    {
      "id": "item-1",
      "name": "Chicken breast",
      "quantity": "2",
      "unit": "lbs",
      "category": "Meat & Seafood",
      "checked": false,
      "recipeIds": ["recipe-1", "recipe-3"],
      "addedManually": false
    },
    ...
  ]
  */

  -- Progress tracking
  checked_count INTEGER DEFAULT 0,
  total_count INTEGER DEFAULT 0,

  -- Sharing
  share_id VARCHAR(50) UNIQUE, -- For shareable links

  -- Timestamps
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

  -- Constraints
  CONSTRAINT check_counts CHECK (checked_count >= 0 AND total_count >= 0 AND checked_count <= total_count)
);

-- Indexes
CREATE INDEX idx_shopping_lists_user_id ON shopping_lists(user_id);
CREATE INDEX idx_shopping_lists_meal_plan_id ON shopping_lists(meal_plan_id);
CREATE INDEX idx_shopping_lists_week_start ON shopping_lists(week_start);
CREATE INDEX idx_shopping_lists_share_id ON shopping_lists(share_id) WHERE share_id IS NOT NULL;

-- Trigger
CREATE TRIGGER update_shopping_lists_updated_at
BEFORE UPDATE ON shopping_lists
FOR EACH ROW
EXECUTE FUNCTION update_updated_at_column();
```

---

### Table: `activities`

**Description**: User activity log for tracking and analytics.

```sql
CREATE TABLE activities (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  action VARCHAR(100) NOT NULL, -- 'meal_planned', 'recipe_favorited', 'shopping_list_generated', etc.
  details TEXT NOT NULL, -- Human-readable description
  related_id UUID, -- References recipe, meal_plan, etc. (no FK constraint for flexibility)
  related_type VARCHAR(50), -- 'recipe', 'meal_plan', 'shopping_list'
  timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW(),

  -- Constraints
  CONSTRAINT check_action CHECK (action IN (
    'meal_planned', 'meal_removed', 'recipe_favorited', 'recipe_unfavorited',
    'shopping_list_generated', 'preferences_updated', 'recipe_viewed'
  ))
);

-- Indexes
CREATE INDEX idx_activities_user_id ON activities(user_id);
CREATE INDEX idx_activities_timestamp ON activities(timestamp DESC);
CREATE INDEX idx_activities_user_timestamp ON activities(user_id, timestamp DESC);
CREATE INDEX idx_activities_action ON activities(action);

-- Partitioning by timestamp (for large scale)
-- CREATE TABLE activities_y2025m10 PARTITION OF activities
-- FOR VALUES FROM ('2025-10-01') TO ('2025-11-01');
```

---

### Table: `sessions`

**Description**: JWT token blacklist for logout functionality.

```sql
CREATE TABLE sessions (
  token_hash VARCHAR(255) PRIMARY KEY, -- SHA-256 hash of JWT token
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  expires_at TIMESTAMPTZ NOT NULL,
  invalidated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

  -- Constraints
  CONSTRAINT check_expires_future CHECK (expires_at > invalidated_at)
);

-- Indexes
CREATE INDEX idx_sessions_user_id ON sessions(user_id);
CREATE INDEX idx_sessions_expires_at ON sessions(expires_at);

-- Auto-delete expired sessions (cron job or pg_cron extension)
-- DELETE FROM sessions WHERE expires_at < NOW() - INTERVAL '1 day';
```

---

## Indexes & Constraints

### Index Strategy

**Primary Keys**: All tables use UUID for primary keys (globally unique, non-sequential)

**Foreign Keys**: All relationships enforced with FK constraints, cascading deletes where appropriate

**Search Indexes**:
- GIN index on `recipes.search_vector` for full-text search
- GIN index on `recipes.tags` for tag filtering
- B-tree indexes on frequently filtered columns (category, rating, created_at)

**Performance Indexes**:
- Composite index on `meal_plans(user_id, week_start)` for fast user meal plan lookups
- Index on `activities(user_id, timestamp DESC)` for activity feed queries
- Index on `favorites(user_id)` for fast favorite lookups

**Partial Indexes**:
- Only index non-deleted rows: `WHERE deleted_at IS NULL`
- Only index shared shopping lists: `WHERE share_id IS NOT NULL`

### Constraint Types

1. **CHECK Constraints**: Enum-like values (role, category, difficulty)
2. **UNIQUE Constraints**: Email, user_week combination
3. **NOT NULL**: All required fields
4. **Foreign Keys**: Referential integrity with cascade options

---

## Data Migration Strategy

### Migration Tools

**Recommended**: Prisma Migrate or Flyway

**Migration Workflow**:
1. Create migration file (SQL or Prisma schema)
2. Test in local environment
3. Run on staging
4. Run on production (during low-traffic window)
5. Monitor for errors, rollback if needed

### Sample Migration: Add `cuisine` Column to Recipes

```sql
-- Migration: 001_add_cuisine_to_recipes.sql
BEGIN;

-- Add column
ALTER TABLE recipes
ADD COLUMN cuisine VARCHAR(100);

-- Backfill existing data (optional)
UPDATE recipes SET cuisine = 'American' WHERE cuisine IS NULL;

-- Add index
CREATE INDEX idx_recipes_cuisine ON recipes(cuisine);

COMMIT;
```

### Rollback Strategy

```sql
-- Rollback: 001_add_cuisine_to_recipes.sql
BEGIN;

DROP INDEX IF EXISTS idx_recipes_cuisine;
ALTER TABLE recipes DROP COLUMN IF EXISTS cuisine;

COMMIT;
```

---

## Backup & Recovery

### Automated Backups (AWS RDS)

**Schedule**:
- Daily snapshots at 3 AM UTC
- Retention: 30 days
- Point-in-time recovery (PITR): Any time within 30 days

**Manual Snapshots**:
- Before major migrations
- Before production deployments
- Labeled clearly: `pre-migration-2025-10-14`

### Recovery Procedures

**Scenario 1: Accidental Data Deletion**
```sql
-- Restore specific table from backup
pg_restore -h production-db -U admin -d mealplanner -t recipes backup-2025-10-14.dump
```

**Scenario 2: Database Corruption**
1. Stop API servers (prevent writes)
2. Restore from latest snapshot
3. Replay WAL (Write-Ahead Log) to PITR
4. Verify data integrity
5. Resume API servers

**RTO (Recovery Time Objective)**: 1 hour
**RPO (Recovery Point Objective)**: 1 hour (max data loss)

---

## Performance Optimization

### Query Optimization

**Slow Query Example** (before optimization):
```sql
-- Fetch all recipes for a user's favorites (N+1 query problem)
SELECT * FROM recipes WHERE id IN (
  SELECT recipe_id FROM favorites WHERE user_id = 'user-123'
);
```

**Optimized Query** (after):
```sql
-- Use JOIN instead
SELECT r.*
FROM recipes r
INNER JOIN favorites f ON r.id = f.recipe_id
WHERE f.user_id = 'user-123' AND r.deleted_at IS NULL;
```

**EXPLAIN ANALYZE**:
```sql
EXPLAIN ANALYZE
SELECT * FROM recipes
WHERE search_vector @@ to_tsquery('chicken & tacos')
ORDER BY rating DESC
LIMIT 20;

-- Check for "Seq Scan" (bad) vs "Index Scan" (good)
```

### Connection Pooling

**PgBouncer Configuration**:
```ini
[databases]
mealplanner = host=rds-primary.amazonaws.com port=5432 dbname=mealplanner

[pgbouncer]
pool_mode = transaction
max_client_conn = 1000
default_pool_size = 25
max_db_connections = 100
```

### VACUUM & ANALYZE

**Schedule**:
- Auto-vacuum: Enabled (PostgreSQL default)
- Manual VACUUM FULL: Monthly (during maintenance window)
- ANALYZE: Weekly (updates query planner statistics)

```sql
-- Manual maintenance
VACUUM (VERBOSE, ANALYZE) recipes;
REINDEX TABLE recipes;
```

---

## Conclusion

This database design provides:
- **Scalability**: JSONB for flexible schemas, partitioning for large tables
- **Performance**: Strategic indexes, full-text search, connection pooling
- **Reliability**: ACID transactions, foreign keys, constraints
- **Maintainability**: Clear schema, migrations, backups

**Next Steps**: Implement API endpoints following this schema (see API_SPECIFICATION.md).
