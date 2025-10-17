# Backend Architecture Document
## AI-Powered Meal Planner

---

**Version:** 1.0
**Last Updated:** 2025-10-14
**Status:** Design Phase

---

## Table of Contents

1. [Architecture Overview](#architecture-overview)
2. [System Architecture Diagram](#system-architecture-diagram)
3. [Service Architecture](#service-architecture)
4. [API Architecture](#api-architecture)
5. [Authentication & Authorization](#authentication--authorization)
6. [Data Flow](#data-flow)
7. [Caching Strategy](#caching-strategy)
8. [File Storage Architecture](#file-storage-architecture)
9. [Search Architecture](#search-architecture)
10. [AI/ML Service Architecture](#aiml-service-architecture)
11. [Notification Service Architecture](#notification-service-architecture)
12. [Scalability Architecture](#scalability-architecture)
13. [High Availability & Disaster Recovery](#high-availability--disaster-recovery)
14. [Security Architecture](#security-architecture)
15. [Monitoring & Observability](#monitoring--observability)

---

## Architecture Overview

### Design Principles

1. **Modular Monolith**: Start with a well-organized monolith, extract microservices only when necessary
2. **Stateless Services**: All API servers stateless for easy horizontal scaling
3. **Cache-First**: Aggressive caching to reduce database load
4. **Async Where Possible**: Non-blocking operations, job queues for heavy tasks
5. **Fail-Safe**: Graceful degradation, circuit breakers, fallbacks
6. **Security by Default**: Encryption, authentication, input validation everywhere
7. **Observability**: Comprehensive logging, metrics, tracing from day one

### Technology Choices

| Component | Technology | Rationale |
|-----------|-----------|-----------|
| **Runtime** | Node.js 20 LTS | JavaScript consistency, async I/O, mature ecosystem |
| **Framework** | Express.js | Battle-tested, flexible, large community |
| **Database** | PostgreSQL 15 | ACID, JSON support, full-text search, reliability |
| **Cache** | Redis 7 | Fast, versatile (cache, sessions, queues) |
| **Storage** | AWS S3 + CloudFront | Scalable, durable, global CDN |
| **Container** | Docker | Portable, consistent environments |
| **Orchestration** | AWS ECS Fargate | Managed, serverless containers |
| **CI/CD** | GitHub Actions | Integrated, easy to use, free for public repos |
| **Monitoring** | CloudWatch + Sentry | AWS native + excellent error tracking |

---

## System Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────────┐
│                          CLIENT LAYER                                │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐              │
│  │  Web Browser │  │ Mobile App   │  │  Admin Panel │              │
│  │  (React SPA) │  │  (Future)    │  │  (React)     │              │
│  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘              │
│         │                  │                  │                      │
│         └──────────────────┴──────────────────┘                      │
│                            │ HTTPS                                   │
└────────────────────────────┼─────────────────────────────────────────┘
                             ▼
┌─────────────────────────────────────────────────────────────────────┐
│                         CDN LAYER                                    │
│                      AWS CloudFront                                  │
│  - Global Edge Locations                                            │
│  - Static Asset Caching (Images, JS, CSS)                           │
│  - SSL/TLS Termination                                              │
│  - DDoS Protection (AWS Shield)                                     │
└────────────────────────────┬────────────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────────────┐
│                      API GATEWAY LAYER                               │
│              AWS Application Load Balancer (ALB)                     │
│  - SSL/TLS Termination                                              │
│  - Health Checks                                                     │
│  - Request Routing                                                   │
│  - Rate Limiting (AWS WAF)                                          │
└────────────────────────────┬────────────────────────────────────────┘
                             │
              ┌──────────────┼──────────────┐
              │              │              │
              ▼              ▼              ▼
┌──────────────────────────────────────────────────────────────────────┐
│                      APPLICATION LAYER                                │
│                  (ECS Fargate Cluster)                               │
│                                                                       │
│  ┌───────────────┐  ┌───────────────┐  ┌───────────────┐           │
│  │  API Server 1 │  │  API Server 2 │  │  API Server N │           │
│  │  (Container)  │  │  (Container)  │  │  (Container)  │           │
│  │               │  │               │  │               │           │
│  │  - Auth       │  │  - Auth       │  │  - Auth       │           │
│  │  - Recipe     │  │  - Recipe     │  │  - Recipe     │           │
│  │  - MealPlan   │  │  - MealPlan   │  │  - MealPlan   │           │
│  │  - Shopping   │  │  - Shopping   │  │  - Shopping   │           │
│  │  - User       │  │  - User       │  │  - User       │           │
│  └───────┬───────┘  └───────┬───────┘  └───────┬───────┘           │
│          │                  │                  │                     │
│          └──────────────────┴──────────────────┘                     │
│                             │                                         │
└─────────────────────────────┼─────────────────────────────────────────┘
                             │
              ┌──────────────┼──────────────┬───────────────┐
              │              │              │               │
              ▼              ▼              ▼               ▼
┌────────────────┐  ┌───────────────┐  ┌──────────┐  ┌────────────┐
│   PostgreSQL   │  │     Redis     │  │   S3     │  │ AI Service │
│   (Primary)    │  │  (Cache)      │  │ (Images) │  │ (Lambda or │
│                │  │               │  │          │  │  Fargate)  │
│  - Users       │  │ - Sessions    │  │ - Recipe │  │            │
│  - Recipes     │  │ - Cache       │  │   Images │  │ - Recs     │
│  - MealPlans   │  │ - Rate Limit  │  │ - User   │  │ - Nutrition│
│  - ShopLists   │  │ - Pub/Sub     │  │   Avatars│  │ - Substit. │
└────────┬───────┘  └───────────────┘  └──────────┘  └────────────┘
         │
         ▼
┌────────────────┐
│   PostgreSQL   │
│ Read Replica 1 │
│  (Read-Only)   │
└────────────────┘
         │
         ▼
┌────────────────┐
│   PostgreSQL   │
│ Read Replica 2 │
│  (Read-Only)   │
└────────────────┘
```

### External Services

```
┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐
│   SendGrid/SES  │  │ AWS Personalize │  │  USDA Food API  │
│  (Email)        │  │  (ML Recs)      │  │  (Nutrition)    │
└─────────────────┘  └─────────────────┘  └─────────────────┘
```

---

## Service Architecture

### Modular Monolith Structure

```
src/
├── modules/
│   ├── auth/
│   │   ├── auth.controller.ts      # HTTP endpoints
│   │   ├── auth.service.ts         # Business logic
│   │   ├── auth.middleware.ts      # JWT validation
│   │   ├── auth.routes.ts          # Route definitions
│   │   ├── auth.validator.ts       # Input validation
│   │   └── auth.types.ts           # TypeScript types
│   │
│   ├── user/
│   │   ├── user.controller.ts
│   │   ├── user.service.ts
│   │   ├── user.repository.ts      # Database layer
│   │   ├── user.routes.ts
│   │   └── user.types.ts
│   │
│   ├── recipe/
│   │   ├── recipe.controller.ts
│   │   ├── recipe.service.ts
│   │   ├── recipe.repository.ts
│   │   ├── recipe.search.ts        # Search logic
│   │   ├── recipe.routes.ts
│   │   └── recipe.types.ts
│   │
│   ├── meal-plan/
│   │   ├── meal-plan.controller.ts
│   │   ├── meal-plan.service.ts
│   │   ├── meal-plan.repository.ts
│   │   ├── meal-plan.routes.ts
│   │   └── meal-plan.types.ts
│   │
│   ├── shopping-list/
│   │   ├── shopping-list.controller.ts
│   │   ├── shopping-list.service.ts
│   │   ├── shopping-list.repository.ts
│   │   ├── ingredient-consolidator.ts
│   │   ├── ingredient-categorizer.ts
│   │   ├── shopping-list.routes.ts
│   │   └── shopping-list.types.ts
│   │
│   ├── ai/
│   │   ├── ai.controller.ts
│   │   ├── recommendation.service.ts
│   │   ├── nutrition.service.ts
│   │   ├── substitution.service.ts
│   │   ├── ai.routes.ts
│   │   └── ai.types.ts
│   │
│   ├── notification/
│   │   ├── notification.controller.ts
│   │   ├── email.service.ts
│   │   ├── template.service.ts
│   │   ├── notification.routes.ts
│   │   └── notification.types.ts
│   │
│   └── admin/
│       ├── admin.controller.ts
│       ├── admin.service.ts
│       ├── analytics.service.ts
│       ├── admin.routes.ts
│       └── admin.types.ts
│
├── common/
│   ├── database/
│   │   ├── connection.ts           # DB connection pool
│   │   ├── migrations/             # Schema migrations
│   │   └── seeds/                  # Test data
│   │
│   ├── cache/
│   │   ├── redis.client.ts
│   │   └── cache.service.ts
│   │
│   ├── storage/
│   │   ├── s3.client.ts
│   │   └── upload.service.ts
│   │
│   ├── middleware/
│   │   ├── auth.middleware.ts      # JWT verification
│   │   ├── error.middleware.ts     # Error handling
│   │   ├── logger.middleware.ts    # Request logging
│   │   ├── rate-limit.middleware.ts
│   │   └── validation.middleware.ts
│   │
│   ├── utils/
│   │   ├── logger.ts               # Winston logger
│   │   ├── errors.ts               # Custom error classes
│   │   ├── responses.ts            # Standard responses
│   │   └── helpers.ts              # Utility functions
│   │
│   └── config/
│       ├── database.config.ts
│       ├── redis.config.ts
│       ├── s3.config.ts
│       └── app.config.ts
│
├── app.ts                          # Express app setup
├── server.ts                       # Server entry point
└── routes.ts                       # Root route aggregation
```

### Layer Responsibilities

**Controller Layer**:
- Handle HTTP requests/responses
- Input validation (using middleware)
- Call service layer
- Transform data for response

**Service Layer**:
- Business logic
- Orchestrate multiple repositories
- Transaction management
- Call external services

**Repository Layer**:
- Database queries (using ORM or SQL)
- Data access only
- No business logic

**Middleware Layer**:
- Authentication, authorization
- Request validation
- Error handling
- Logging, rate limiting

---

## API Architecture

### RESTful API Design

**Base URL**: `https://api.mealplanner.com/api/v1`

**Versioning**: URL path versioning (`/api/v1`, `/api/v2`)

### Endpoint Structure

```
Authentication:
POST   /api/v1/auth/register
POST   /api/v1/auth/login
POST   /api/v1/auth/logout
POST   /api/v1/auth/refresh
GET    /api/v1/auth/me

Users:
GET    /api/v1/users/:userId
PATCH  /api/v1/users/:userId
DELETE /api/v1/users/:userId
PATCH  /api/v1/users/:userId/preferences
PATCH  /api/v1/users/:userId/password
GET    /api/v1/users/:userId/export

Recipes:
GET    /api/v1/recipes                    # List with filters
POST   /api/v1/recipes                    # Admin only
GET    /api/v1/recipes/:recipeId
PATCH  /api/v1/recipes/:recipeId          # Admin only
DELETE /api/v1/recipes/:recipeId          # Admin only
GET    /api/v1/recipes/search?q=chicken
POST   /api/v1/recipes/:recipeId/favorite
DELETE /api/v1/recipes/:recipeId/favorite
GET    /api/v1/users/:userId/favorites

Meal Plans:
GET    /api/v1/users/:userId/meal-plans?weekStart=2025-10-14
POST   /api/v1/users/:userId/meal-plans
PATCH  /api/v1/users/:userId/meal-plans/:planId
DELETE /api/v1/users/:userId/meal-plans/:planId
POST   /api/v1/meal-plans/:planId/meals
DELETE /api/v1/meal-plans/:planId/meals
POST   /api/v1/meal-plans/:planId/copy-day

Shopping Lists:
POST   /api/v1/shopping-lists/generate    # From meal plan
GET    /api/v1/shopping-lists/:listId
POST   /api/v1/shopping-lists/:listId/items
PATCH  /api/v1/shopping-lists/:listId/items/:itemId
DELETE /api/v1/shopping-lists/:listId/items/:itemId
POST   /api/v1/shopping-lists/:listId/share

AI Features:
POST   /api/v1/ai/suggestions             # Get meal suggestions
POST   /api/v1/ai/nutrition-balance       # Calculate balance
POST   /api/v1/ai/substitutions           # Ingredient substitutions

Admin:
GET    /api/v1/admin/users                # Paginated user list
GET    /api/v1/admin/analytics            # Dashboard metrics
GET    /api/v1/admin/health               # System health
```

### Request/Response Patterns

**Pagination**:
```
GET /api/v1/recipes?page=1&limit=20

Response:
{
  "data": [...],
  "pagination": {
    "page": 1,
    "limit": 20,
    "total": 245,
    "totalPages": 13
  }
}
```

**Filtering**:
```
GET /api/v1/recipes?category=Dinner&dietary=Vegan&maxPrepTime=30
```

**Sorting**:
```
GET /api/v1/recipes?sort=createdAt:desc,rating:desc
```

**Error Response**:
```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Invalid input data",
    "details": {
      "email": "Email is required",
      "password": "Password must be at least 8 characters"
    }
  }
}
```

### HTTP Status Codes

- `200 OK`: Successful GET, PATCH, DELETE
- `201 Created`: Successful POST
- `204 No Content`: Successful DELETE with no response body
- `400 Bad Request`: Validation error
- `401 Unauthorized`: Missing or invalid auth token
- `403 Forbidden`: Insufficient permissions
- `404 Not Found`: Resource doesn't exist
- `409 Conflict`: Resource already exists (e.g., duplicate email)
- `429 Too Many Requests`: Rate limit exceeded
- `500 Internal Server Error`: Server error
- `503 Service Unavailable`: Temporary service outage

---

## Authentication & Authorization

### JWT Authentication Flow

```
┌──────────┐                      ┌──────────┐
│  Client  │                      │   API    │
└────┬─────┘                      └────┬─────┘
     │                                  │
     │  POST /auth/login                │
     │  { email, password }             │
     ├─────────────────────────────────>│
     │                                  │
     │                         Validate credentials
     │                         Generate JWT access token
     │                         Generate refresh token
     │                         Store refresh token hash in DB
     │                                  │
     │  { accessToken, refreshToken }  │
     │<─────────────────────────────────┤
     │                                  │
     │  Store tokens in localStorage    │
     │  or httpOnly cookies             │
     │                                  │
     │  GET /users/123                  │
     │  Authorization: Bearer <token>   │
     ├─────────────────────────────────>│
     │                                  │
     │                         Verify JWT signature
     │                         Check expiration
     │                         Extract userId from payload
     │                                  │
     │  { user data }                  │
     │<─────────────────────────────────┤
     │                                  │
     │  (Access token expires)          │
     │                                  │
     │  POST /auth/refresh              │
     │  { refreshToken }                │
     ├─────────────────────────────────>│
     │                                  │
     │                         Verify refresh token
     │                         Check token not blacklisted
     │                         Generate new access token
     │                                  │
     │  { accessToken }                │
     │<─────────────────────────────────┤
     │                                  │
```

### JWT Token Structure

**Access Token** (expires in 1 hour):
```json
{
  "header": {
    "alg": "HS256",
    "typ": "JWT"
  },
  "payload": {
    "userId": "user-123",
    "email": "user@example.com",
    "role": "user",
    "iat": 1697280000,
    "exp": 1697283600
  },
  "signature": "..."
}
```

**Refresh Token** (expires in 7 days):
- Longer expiration
- Stored in database (hashed)
- Used only for `/auth/refresh` endpoint
- Single-use (invalidated after refresh)

### Authorization Levels

1. **Public**: No authentication required
   - `POST /auth/register`, `POST /auth/login`

2. **Authenticated User**: Valid JWT required
   - All `/users/:userId/*` endpoints (user can only access own data)
   - All `/recipes`, `/meal-plans`, `/shopping-lists`

3. **Admin**: `role: admin` in JWT payload
   - `POST /recipes`, `PATCH /recipes/:id`, `DELETE /recipes/:id`
   - All `/admin/*` endpoints

### Middleware Stack

```typescript
// Example route with middleware
router.get(
  '/users/:userId/meal-plans',
  authenticate,           // Verify JWT
  authorize('user'),      // Check role
  validateUserId,         // Ensure user can only access own data
  mealPlanController.getMealPlans
);

router.post(
  '/recipes',
  authenticate,
  authorize('admin'),     // Admin only
  validateRecipeInput,    // Validate request body
  recipeController.createRecipe
);
```

---

## Data Flow

### Example: User Plans a Meal

```
┌──────────┐
│  Client  │
│ (React)  │
└────┬─────┘
     │
     │ 1. User clicks "Add Meal to Monday Dinner"
     │    Opens recipe browser, selects "Chicken Tacos"
     │
     │ 2. POST /api/v1/meal-plans/plan-001/meals
     │    { date: "2025-10-14", mealType: "dinner", recipeId: "recipe-123" }
     │    Authorization: Bearer <JWT>
     │
     ▼
┌──────────────────┐
│   API Gateway    │ 3. ALB receives request
│   (ALB)          │    Checks rate limits (AWS WAF)
└────┬─────────────┘    Routes to API server
     │
     ▼
┌──────────────────┐
│   API Server     │ 4. Express app receives request
│   (Fargate)      │    Middleware chain executes:
└────┬─────────────┘
     │
     ├─> Auth Middleware:
     │   - Verify JWT signature
     │   - Check expiration
     │   - Extract userId from token
     │   - Attach user to req.user
     │
     ├─> Authorization Middleware:
     │   - Ensure user owns meal plan (planId belongs to userId)
     │
     ├─> Validation Middleware:
     │   - Validate request body schema
     │   - Check date, mealType, recipeId valid
     │
     ▼
┌──────────────────┐
│  MealPlanController │ 5. Call service layer
└────┬─────────────┘
     │
     ▼
┌──────────────────┐
│  MealPlanService │ 6. Business logic:
└────┬─────────────┘    - Fetch meal plan from DB
     │                  - Check recipe exists
     ├─────────────────> RecipeRepository.findById(recipeId)
     │                  - Add meal to plan
     │                  - Invalidate cache
     ▼
┌──────────────────┐
│   PostgreSQL     │ 7. Update meal_plans table:
│   (Primary DB)   │    UPDATE meal_plans
└────┬─────────────┘    SET meals = jsonb_set(meals, '{monday,dinner}', '"recipe-123"')
     │                  WHERE id = 'plan-001'
     │
     │ 8. Return updated meal plan
     │
     ▼
┌──────────────────┐
│      Redis       │ 9. Invalidate cached meal plan:
│   (Cache)        │    DEL meal-plan:user-001:2025-10-13
└────┬─────────────┘
     │
     │ 10. Log activity
     │
     ▼
┌──────────────────┐
│  ActivityService │ 11. INSERT INTO activities
└────┬─────────────┘     (userId, action, details, timestamp)
     │
     │ 12. Return response
     │
     ▼
┌──────────────────┐
│   Client         │ 13. Response: { mealPlan: {...} }
└──────────────────┘     Update UI, show success toast
```

---

## Caching Strategy

### Cache Hierarchy

```
┌──────────────────────────────────────────────┐
│           L1: Application Cache               │
│         (In-Memory, per API server)           │
│                                               │
│  - Config, constants                          │
│  - Static data (categories, units)            │
│  - LRU cache, 100MB max                       │
│  - TTL: 1 hour                                │
└───────────────────┬──────────────────────────┘
                    │ Cache Miss
                    ▼
┌──────────────────────────────────────────────┐
│           L2: Redis Cache                     │
│         (Distributed, shared)                 │
│                                               │
│  - User sessions                              │
│  - Meal plans, recipes                        │
│  - Search results                             │
│  - Rate limit counters                        │
│  - TTL: Varies (5 min - 1 hour)              │
└───────────────────┬──────────────────────────┘
                    │ Cache Miss
                    ▼
┌──────────────────────────────────────────────┐
│           L3: Database                        │
│         (PostgreSQL)                          │
│                                               │
│  - Persistent data                            │
│  - Query result caching (PostgreSQL internal) │
└───────────────────┬──────────────────────────┘
                    │
                    ▼
┌──────────────────────────────────────────────┐
│           L4: CDN Cache                       │
│         (CloudFront)                          │
│                                               │
│  - Static assets (images, JS, CSS)            │
│  - TTL: 1 week (immutable assets)            │
│  - Edge caching (global)                      │
└───────────────────────────────────────────────┘
```

### Cache Keys & TTL

| Resource | Key Pattern | TTL | Invalidation Trigger |
|----------|-------------|-----|----------------------|
| **User** | `user:{userId}` | 5 min | User update, preferences change |
| **Recipe** | `recipe:{recipeId}` | 1 hour | Recipe update (admin) |
| **Meal Plan** | `meal-plan:{userId}:{weekStart}` | 10 min | Meal added/removed, plan updated |
| **Shopping List** | `shopping-list:{listId}` | 5 min | Item added/removed/checked |
| **Search Results** | `search:recipes:{query}:{filters}` | 15 min | New recipe added, recipe updated |
| **Favorites** | `favorites:{userId}` | 10 min | Recipe favorited/unfavorited |
| **Suggestions** | `suggestions:{userId}` | 30 min | Preferences changed, meal plan updated |

### Cache Invalidation Strategy

**Write-Through**:
- On write operation, update database first
- Then invalidate (delete) cache key
- Next read will populate cache from DB

**Example**:
```typescript
async addMealToSlot(planId, date, mealType, recipeId) {
  // 1. Update database
  await db.mealPlans.update(planId, { ... });

  // 2. Invalidate cache
  const cacheKey = `meal-plan:${userId}:${weekStart}`;
  await redis.del(cacheKey);

  // 3. Return updated data
  return this.getMealPlan(planId); // Will cache on read
}
```

**Cache Warming**:
- On deployment, pre-populate cache with frequently accessed data
- Popular recipes, common search queries
- Reduces cold start latency

---

## File Storage Architecture

### S3 Bucket Structure

```
meal-planner-images-production/
├── recipes/
│   ├── {recipeId}/
│   │   ├── original/
│   │   │   └── {filename}.jpg
│   │   ├── large/
│   │   │   └── {filename}_1200x800.webp
│   │   ├── medium/
│   │   │   └── {filename}_600x400.webp
│   │   └── thumbnail/
│   │       └── {filename}_200x200.webp
│   └── ...
├── users/
│   └── {userId}/
│       ├── avatar_200x200.webp
│       └── avatar_original.jpg
└── temp/
    └── {uploadId}/
        └── {filename}  (auto-deleted after 24 hours)
```

### Image Upload Flow

```
┌──────────┐
│  Client  │
└────┬─────┘
     │
     │ 1. Request upload URL
     │    POST /api/v1/recipes/123/upload-url
     │    { filename: "chicken.jpg", contentType: "image/jpeg" }
     │
     ▼
┌──────────────────┐
│   API Server     │ 2. Generate presigned S3 URL
│                  │    - Valid for 15 minutes
│                  │    - Max file size: 10MB
└────┬─────────────┘    - Only image/* content types
     │
     │ 3. Return presigned URL
     │    { uploadUrl: "https://s3.../...", key: "recipes/123/..." }
     │
     ▼
┌──────────┐
│  Client  │ 4. Upload directly to S3
│          │    PUT to uploadUrl
└────┬─────┘    (bypasses API server)
     │
     ▼
┌──────────────────┐
│      S3          │ 5. Image stored in temp/
└────┬─────────────┘
     │
     │ 6. Confirm upload
     │    POST /api/v1/recipes/123/confirm-upload
     │    { key: "recipes/123/..." }
     │
     ▼
┌──────────────────┐
│   API Server     │ 7. Trigger image processing
│                  │    - Invoke Lambda or queue job
└────┬─────────────┘
     │
     ▼
┌──────────────────┐
│  Image Processor │ 8. Resize, optimize, convert:
│  (Lambda/Fargate)│    - Original → large, medium, thumb
└────┬─────────────┘    - Convert to WebP
     │                  - Optimize (reduce file size)
     │
     │ 9. Upload variants to S3
     │
     ▼
┌──────────────────┐
│      S3          │ 10. Store all variants
└────┬─────────────┘     - Delete temp/ original
     │
     │ 11. Update recipe with image URLs
     │
     ▼
┌──────────────────┐
│   PostgreSQL     │ 12. UPDATE recipes
│                  │     SET image_url = 'https://cdn.../medium/...'
└──────────────────┘         image_urls = { large: ..., medium: ..., thumb: ... }
```

### CDN Configuration

**CloudFront Distribution**:
- Origin: S3 bucket
- Cache behaviors:
  - Images: Cache for 1 week (immutable)
  - Cache key: Full URL (includes size variant)
- Compression: Gzip, Brotli
- Custom domain: `cdn.mealplanner.com`
- SSL/TLS: AWS Certificate Manager

---

## Search Architecture

### PostgreSQL Full-Text Search

**Approach**: Use PostgreSQL's built-in full-text search initially, migrate to Elasticsearch if needed at scale.

**Search Index**:
```sql
-- Add tsvector column for full-text search
ALTER TABLE recipes
ADD COLUMN search_vector tsvector
GENERATED ALWAYS AS (
  to_tsvector('english',
    coalesce(name, '') || ' ' ||
    coalesce(description, '') || ' ' ||
    coalesce(cuisine, '') || ' ' ||
    coalesce(array_to_string(tags, ' '), '') || ' ' ||
    coalesce((
      SELECT string_agg(ingredient_name, ' ')
      FROM jsonb_array_elements_text(ingredients)
    ), '')
  )
) STORED;

-- Create GIN index for fast search
CREATE INDEX recipes_search_idx ON recipes USING GIN(search_vector);
```

**Search Query**:
```sql
SELECT
  id, name, image_url, prep_time, rating,
  ts_rank(search_vector, query) AS rank
FROM recipes,
  to_tsquery('english', 'chicken & tacos') AS query
WHERE search_vector @@ query
  AND category = 'Dinner'  -- Filter by category
  AND tags @> '["Keto"]'   -- Filter by dietary tags
ORDER BY rank DESC, rating DESC
LIMIT 20 OFFSET 0;
```

**Autocomplete**:
- Use trigram similarity (pg_trgm extension)
- Index recipe names
- Query: `SELECT name FROM recipes WHERE name % 'chic' ORDER BY similarity(name, 'chic') DESC LIMIT 5;`

**Search Caching**:
- Cache search results in Redis
- Key: `search:recipes:{query}:{filters}:{sort}:{page}`
- TTL: 15 minutes
- Invalidate on new recipe added or updated

### Migration to Elasticsearch (Future)

**When to migrate**:
- Search queries taking >300ms
- Complex relevance tuning needed
- Millions of recipes

**Elasticsearch Setup**:
- Index: `recipes`
- Mapping: name (text), ingredients (text), tags (keyword), category (keyword)
- Analyzers: Standard, autocomplete (edge n-gram)
- Sync: River plugin or app-level (write to ES on recipe create/update)

---

## AI/ML Service Architecture

### Recommendation Engine

**Architecture**: Hybrid (Collaborative Filtering + Content-Based)

```
┌──────────────────────────────────────────────┐
│         User Interaction Events              │
│  (Meal planned, recipe favorited, viewed)    │
└───────────────────┬──────────────────────────┘
                    │
                    ▼
┌──────────────────────────────────────────────┐
│          AWS Personalize Dataset             │
│  - Users (userId, preferences, allergies)    │
│  - Items (recipeId, category, tags, nutrition│
│  - Interactions (userId, recipeId, event,    │
│                  timestamp)                   │
└───────────────────┬──────────────────────────┘
                    │
                    ▼ (Train weekly)
┌──────────────────────────────────────────────┐
│       AWS Personalize Model                  │
│  - User-Personalization recipe               │
│  - SIMS (Similar Items) recipe               │
│  - Popularity-Count recipe (fallback)        │
└───────────────────┬──────────────────────────┘
                    │
                    ▼ (Real-time inference)
┌──────────────────────────────────────────────┐
│       Recommendation API Endpoint            │
│  POST /api/v1/ai/suggestions                 │
│  { userId, context: { mealType, date } }     │
│                                               │
│  Response:                                   │
│  { recommendations: [                        │
│      { recipeId, score, reason }             │
│    ]                                         │
│  }                                           │
└───────────────────┬──────────────────────────┘
                    │
                    ▼ (Fallback if ML unavailable)
┌──────────────────────────────────────────────┐
│       Rule-Based Recommendations             │
│  - Filter by dietary preferences             │
│  - Match cuisine from favorites              │
│  - Time-appropriate (breakfast in AM)        │
│  - Exclude recently planned                  │
│  - Sort by popularity                        │
└───────────────────────────────────────────────┘
```

### Event Tracking

**Events to Track**:
- `meal_planned`: User adds recipe to meal plan
- `recipe_favorited`: User favorites recipe
- `recipe_viewed`: User views recipe detail
- `recipe_rated`: User rates recipe (future)
- `shopping_list_generated`: User generates shopping list

**Event Schema**:
```json
{
  "userId": "user-123",
  "recipeId": "recipe-456",
  "eventType": "meal_planned",
  "timestamp": "2025-10-14T10:30:00Z",
  "context": {
    "mealType": "dinner",
    "date": "2025-10-15"
  }
}
```

**Event Flow**:
1. User action triggers event
2. API logs event to database (async)
3. Background job syncs events to AWS Personalize (batch, every 6 hours)
4. Personalize retrains model (weekly)

### Nutrition Analysis

**Service**: `nutrition.service.ts`

**Features**:
1. **Daily/Weekly Summaries**:
   - Aggregate nutrition from all meals in plan
   - Calculate daily totals (calories, protein, carbs, fat)
   - Weekly averages

2. **Balance Score Algorithm**:
   ```typescript
   function calculateBalanceScore(weeklyNutrition) {
     let score = 100;

     // Consistency: Penalize high variance in daily calories
     const calorieStdDev = calculateStdDev(dailyCalories);
     if (calorieStdDev > 500) score -= 20;

     // Macro balance: Ideal 30% protein, 40% carbs, 30% fat
     const macroScore = scoreMacroDistribution(avgMacros);
     score = score * 0.6 + macroScore * 0.4;

     // Variety: Unique meal count
     const uniqueMeals = new Set(recipeIds).size;
     if (uniqueMeals < 7) score -= 10;

     return Math.max(0, Math.min(100, score));
   }
   ```

3. **Actionable Insights**:
   - NLP-based suggestions (rule-based initially)
   - Examples: "Add more vegetables on Thursday", "Great protein balance!"

---

## Notification Service Architecture

### Email Service

```
┌──────────────────┐
│   Trigger Event  │ (User registers, meal plan reminder)
└────┬─────────────┘
     │
     ▼
┌──────────────────┐
│  Email Queue     │ (Redis Queue or AWS SQS)
│  Job: { userId,  │
│         template,│
│         data }   │
└────┬─────────────┘
     │
     ▼
┌──────────────────┐
│  Email Worker    │ (Background job processor)
│                  │ - Fetch user email
│                  │ - Render template
│                  │ - Send via SendGrid/SES
└────┬─────────────┘
     │
     ▼
┌──────────────────┐
│  SendGrid / SES  │ (Email delivery service)
└────┬─────────────┘
     │
     ▼
┌──────────────────┐
│  User Inbox      │
└──────────────────┘
```

**Email Templates**:
- Welcome email (`welcome.html`)
- Meal plan reminder (`meal-plan-reminder.html`)
- Shopping list ready (`shopping-list-ready.html`)
- Weekly summary (`weekly-summary.html`)
- Password reset (`password-reset.html`)

**Template Engine**: Handlebars or EJS

**Example**:
```html
<!-- welcome.html -->
<html>
  <body>
    <h1>Welcome to Meal Planner, {{userName}}!</h1>
    <p>Start planning your meals today.</p>
    <a href="{{dashboardUrl}}">Go to Dashboard</a>
  </body>
</html>
```

**Email Scheduling**:
- Meal plan reminders: Sunday 9 AM (user's timezone)
- Weekly summary: Saturday 6 PM
- Use cron jobs (node-cron) or AWS EventBridge

---

## Scalability Architecture

### Horizontal Scaling

**Auto-Scaling Group Configuration**:
```yaml
MinSize: 2         # Always at least 2 instances (HA)
MaxSize: 20        # Scale up to 20 instances
DesiredCapacity: 2 # Start with 2

ScaleUpPolicy:
  MetricType: CPUUtilization
  TargetValue: 70%
  ScaleUpCooldown: 60s

ScaleDownPolicy:
  MetricType: CPUUtilization
  TargetValue: 30%
  ScaleDownCooldown: 300s
```

**Load Balancer**:
- Algorithm: Round robin
- Health checks: `GET /health` every 30s
- Unhealthy threshold: 2 failures
- Drain connections: 30s before terminating instance

### Database Scaling

**Read Replicas**:
```
┌──────────────────┐
│   Primary DB     │ (All writes)
│   (RW)           │
└────┬─────────────┘
     │
     │ Async replication
     │
     ├─────────────┬─────────────┐
     ▼             ▼             ▼
┌──────────┐  ┌──────────┐  ┌──────────┐
│ Replica 1│  │ Replica 2│  │ Replica 3│
│   (RO)   │  │   (RO)   │  │   (RO)   │
└──────────┘  └──────────┘  └──────────┘
     │             │             │
     └─────────────┴─────────────┘
              │ (Read queries)
              ▼
     ┌──────────────────┐
     │   API Servers    │
     └──────────────────┘
```

**Query Routing**:
```typescript
// Write queries → Primary
await db.primary.query('INSERT INTO users ...');

// Read queries → Replicas (random selection)
const replica = db.replicas[Math.floor(Math.random() * db.replicas.length)];
await replica.query('SELECT * FROM users WHERE id = ?');
```

**Connection Pooling**:
- PgBouncer as middleware
- Pool size: 100 connections per instance
- Max 2000 connections total (Primary), 1000 per replica

### Cache Scaling

**Redis Cluster**:
- 3 shards (master + replica per shard)
- Total: 6 nodes (3 masters, 3 replicas)
- Sharding: Hash-based (consistent hashing)
- Failover: Automatic (replica promoted to master)

---

## High Availability & Disaster Recovery

### Multi-AZ Deployment

```
┌─────────────────────────────────────────────────────┐
│                   AWS Region (us-east-1)            │
│                                                     │
│  ┌──────────────────┐        ┌──────────────────┐ │
│  │   AZ-1a          │        │   AZ-1b          │ │
│  │                  │        │                  │ │
│  │ ┌─────────────┐  │        │ ┌─────────────┐ │ │
│  │ │API Server 1 │  │        │ │API Server 2 │ │ │
│  │ └─────────────┘  │        │ └─────────────┘ │ │
│  │                  │        │                  │ │
│  │ ┌─────────────┐  │        │ ┌─────────────┐ │ │
│  │ │ DB Primary  │◄─┼────────┼─┤ DB Replica  │ │ │
│  │ └─────────────┘  │ Sync   │ └─────────────┘ │ │
│  │                  │        │                  │ │
│  │ ┌─────────────┐  │        │ ┌─────────────┐ │ │
│  │ │Redis Master │◄─┼────────┼─┤Redis Replica│ │ │
│  │ └─────────────┘  │ Sync   │ └─────────────┘ │ │
│  └──────────────────┘        └──────────────────┘ │
└─────────────────────────────────────────────────────┘
```

**Failover Scenarios**:

1. **API Server Failure**:
   - Health check detects failure
   - ALB stops routing to unhealthy instance
   - Auto-scaling replaces instance
   - Downtime: ~30 seconds

2. **Database Primary Failure**:
   - RDS detects failure
   - Promotes replica to primary
   - Updates DNS (CNAME)
   - Downtime: ~60 seconds

3. **Availability Zone Failure**:
   - All services in AZ-1a fail
   - ALB routes all traffic to AZ-1b
   - Auto-scaling launches new instances in AZ-1b
   - Downtime: ~2 minutes

### Backup & Recovery

**Database Backups**:
- Automated daily snapshots (RDS)
- Retention: 30 days
- Point-in-time recovery (PITR): Any time within 30 days
- Manual snapshots before major migrations

**Application State**:
- Stateless API servers (no local state)
- All state in database or Redis
- Redis persistence: RDB snapshots (hourly), AOF log

**Disaster Recovery Plan**:
1. **Recovery Time Objective (RTO)**: 1 hour
2. **Recovery Point Objective (RPO)**: 1 hour (max data loss)
3. **Steps**:
   - Restore database from latest snapshot
   - Replay transaction logs (PITR)
   - Deploy API servers from latest Docker image
   - Restore Redis from snapshot
   - Update DNS if needed

---

## Security Architecture

### Network Security

```
┌──────────────────────────────────────────────┐
│              VPC (10.0.0.0/16)               │
│                                              │
│  ┌──────────────────────────────────────┐   │
│  │  Public Subnet (10.0.1.0/24)         │   │
│  │  - ALB                               │   │
│  │  - NAT Gateway                       │   │
│  └────────────┬─────────────────────────┘   │
│               │                              │
│  ┌────────────┴─────────────────────────┐   │
│  │  Private Subnet (10.0.2.0/24)        │   │
│  │  - API Servers (ECS Tasks)           │   │
│  │  - No direct internet access         │   │
│  └────────────┬─────────────────────────┘   │
│               │                              │
│  ┌────────────┴─────────────────────────┐   │
│  │  DB Subnet (10.0.3.0/24)             │   │
│  │  - RDS PostgreSQL                    │   │
│  │  - ElastiCache Redis                 │   │
│  │  - No internet access                │   │
│  └──────────────────────────────────────┘   │
└──────────────────────────────────────────────┘
```

**Security Groups**:
- **ALB SG**: Inbound 443 (HTTPS) from 0.0.0.0/0, Outbound to API SG
- **API SG**: Inbound from ALB SG only, Outbound to DB SG, Redis SG, Internet (NAT)
- **DB SG**: Inbound 5432 (PostgreSQL) from API SG only
- **Redis SG**: Inbound 6379 (Redis) from API SG only

### Secrets Management

**AWS Secrets Manager**:
- Database credentials
- JWT secret
- SendGrid API key
- AWS access keys (for S3)

**Environment Variables**:
```bash
# Never committed to Git
DB_HOST=<from Secrets Manager>
DB_PASSWORD=<from Secrets Manager>
JWT_SECRET=<from Secrets Manager>
SENDGRID_API_KEY=<from Secrets Manager>
```

**Rotation**:
- Database passwords: Rotated quarterly (automated by RDS)
- JWT secret: Rotated yearly (manual, requires re-login all users)
- API keys: Rotated quarterly

---

## Monitoring & Observability

### Metrics

**Application Metrics** (CloudWatch Custom Metrics):
- Request rate (req/sec)
- Response time (P50, P95, P99)
- Error rate (%)
- Cache hit rate (%)
- Database query time (ms)
- Active users (gauge)

**Infrastructure Metrics** (CloudWatch):
- CPU utilization (%)
- Memory utilization (%)
- Network I/O (MB/s)
- Disk I/O (IOPS)

**Business Metrics**:
- Meals planned (count/day)
- Recipes favorited (count/day)
- Shopping lists generated (count/day)
- User registrations (count/day)

### Logging

**Structured Logs** (JSON):
```json
{
  "timestamp": "2025-10-14T10:30:00Z",
  "level": "info",
  "service": "api",
  "requestId": "req-123",
  "userId": "user-456",
  "method": "POST",
  "path": "/api/v1/meal-plans/plan-001/meals",
  "statusCode": 201,
  "duration": 145,
  "message": "Meal added successfully"
}
```

**Log Aggregation**:
- CloudWatch Logs for all services
- Retention: 90 days (production), 30 days (staging)
- Dashboards: Real-time log search, filters, alerts

### Alerting

**Alert Rules**:
| Metric | Threshold | Action |
|--------|-----------|--------|
| Error rate | > 1% for 5 min | PagerDuty alert |
| API P95 response time | > 500ms for 5 min | Slack notification |
| Database CPU | > 80% for 10 min | PagerDuty alert |
| Disk space | < 20% | Email notification |
| Failed health checks | 2 consecutive | Auto-scale, PagerDuty |

**On-Call Rotation**:
- PagerDuty integration
- 24/7 coverage
- Escalation policy: Level 1 → Level 2 (30 min) → Manager (1 hour)

### Distributed Tracing

**AWS X-Ray**:
- Trace requests across services
- Identify bottlenecks (slow DB queries, external API calls)
- Visualize service map

**Example Trace**:
```
POST /api/v1/meal-plans/plan-001/meals
├─ Auth Middleware (5ms)
├─ MealPlanController (145ms)
│  ├─ RecipeRepository.findById (45ms)
│  │  └─ Database Query (40ms)
│  ├─ MealPlanRepository.update (85ms)
│  │  └─ Database Query (80ms)
│  └─ Cache Invalidation (5ms)
└─ Response (200ms total)
```

---

## Conclusion

This architecture provides a solid foundation for the Meal Planner backend:

- **Scalable**: From 1K to 1M users without major re-architecture
- **Reliable**: 99.9% uptime, multi-AZ, automated failover
- **Performant**: < 200ms API responses, aggressive caching
- **Secure**: Encryption, authentication, regular audits
- **Observable**: Comprehensive monitoring, logging, alerting
- **Maintainable**: Modular code, IaC, automated deployments

The modular monolith approach allows rapid development while keeping the door open for future microservices extraction if needed.

---

**Next Steps**: Proceed to Database Design (DATABASE_DESIGN.md) for detailed schema specifications.
