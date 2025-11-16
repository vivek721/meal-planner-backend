# Backend Architecture Document
## AI-Powered Meal Planner

---

> **Note**: This document has been updated for the Golang implementation (v2.0).
> The backend was migrated from Node.js/TypeScript to Golang on October 16, 2025.

**Version:** 2.0
**Last Updated:** 2025-10-17
**Status:** Production Implementation

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
| **Runtime** | Go 1.21+ | Performance, concurrency, static typing, simple deployment |
| **Framework** | Gin | Fast, minimalist, excellent routing, middleware support |
| **Database** | PostgreSQL 15 | ACID, JSON support, full-text search, reliability |
| **ORM** | GORM | Full-featured, migrations, relationships, hooks |
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
│  │  Go Binary    │  │  Go Binary    │  │  Go Binary    │           │
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
backend/
├── cmd/
│   └── server/
│       └── main.go                 # Application entry point
│
├── internal/
│   ├── auth/
│   │   ├── handler.go              # HTTP handlers (Gin)
│   │   ├── service.go              # Business logic
│   │   ├── repository.go           # Database layer (GORM)
│   │   ├── middleware.go           # JWT validation
│   │   ├── routes.go               # Route registration
│   │   └── models.go               # Domain models
│   │
│   ├── user/
│   │   ├── handler.go
│   │   ├── service.go
│   │   ├── repository.go
│   │   ├── routes.go
│   │   └── models.go
│   │
│   ├── recipe/
│   │   ├── handler.go
│   │   ├── service.go
│   │   ├── repository.go
│   │   ├── search.go               # Search logic
│   │   ├── routes.go
│   │   └── models.go
│   │
│   ├── mealplan/
│   │   ├── handler.go
│   │   ├── service.go
│   │   ├── repository.go
│   │   ├── routes.go
│   │   └── models.go
│   │
│   ├── shopping/
│   │   ├── handler.go
│   │   ├── service.go
│   │   ├── repository.go
│   │   ├── consolidator.go         # Ingredient consolidation
│   │   ├── categorizer.go          # Categorization logic
│   │   ├── routes.go
│   │   └── models.go
│   │
│   ├── ai/
│   │   ├── handler.go
│   │   ├── recommendation.go       # Recommendation service
│   │   ├── nutrition.go            # Nutrition analysis
│   │   ├── substitution.go         # Ingredient substitution
│   │   ├── routes.go
│   │   └── models.go
│   │
│   └── notification/
│       ├── handler.go
│       ├── email.go                # Email service
│       ├── template.go             # Template rendering
│       ├── routes.go
│       └── models.go
│
├── pkg/
│   ├── database/
│   │   ├── postgres.go             # PostgreSQL connection
│   │   └── migrations/             # Database migrations
│   │
│   ├── cache/
│   │   ├── redis.go                # Redis client
│   │   └── cache.go                # Cache service
│   │
│   ├── storage/
│   │   ├── s3.go                   # S3 client
│   │   └── upload.go               # Upload service
│   │
│   ├── middleware/
│   │   ├── auth.go                 # JWT verification
│   │   ├── error.go                # Error handling
│   │   ├── logger.go               # Request logging
│   │   ├── ratelimit.go            # Rate limiting
│   │   └── cors.go                 # CORS middleware
│   │
│   ├── utils/
│   │   ├── logger.go               # Structured logging
│   │   ├── errors.go               # Custom errors
│   │   ├── response.go             # Standard responses
│   │   └── validator.go            # Input validation
│   │
│   └── config/
│       └── config.go               # Configuration management
│
├── go.mod                          # Go module dependencies
├── go.sum                          # Dependency checksums
├── Makefile                        # Build and run commands
└── Dockerfile                      # Container definition
```

### Layer Responsibilities

**Handler Layer** (Gin Handlers):
- Handle HTTP requests/responses
- Input validation (using validator)
- Call service layer
- Transform data for response

**Service Layer**:
- Business logic
- Orchestrate multiple repositories
- Transaction management
- Call external services

**Repository Layer**:
- Database queries (using GORM)
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

### Middleware Stack (Gin Example)

```go
// Example route with middleware
func SetupRoutes(r *gin.Engine, authMiddleware *middleware.AuthMiddleware) {
    v1 := r.Group("/api/v1")

    // Public routes
    auth := v1.Group("/auth")
    {
        auth.POST("/register", authHandler.Register)
        auth.POST("/login", authHandler.Login)
    }

    // Protected routes (user must be authenticated)
    protected := v1.Group("")
    protected.Use(authMiddleware.RequireAuth())
    {
        // User routes (user can only access own data)
        protected.GET("/users/:userId/meal-plans",
            authMiddleware.ValidateUserAccess(),
            mealPlanHandler.GetUserMealPlans)

        // Admin-only routes
        adminRoutes := protected.Group("")
        adminRoutes.Use(authMiddleware.RequireRole("admin"))
        {
            adminRoutes.POST("/recipes", recipeHandler.CreateRecipe)
            adminRoutes.PATCH("/recipes/:id", recipeHandler.UpdateRecipe)
        }
    }
}
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
│   API Server     │ 4. Gin app receives request
│   (Fargate)      │    Middleware chain executes:
└────┬─────────────┘
     │
     ├─> Auth Middleware:
     │   - Verify JWT signature (dgrijalva/jwt-go)
     │   - Check expiration
     │   - Extract userId from token
     │   - Attach user to Gin context
     │
     ├─> Authorization Middleware:
     │   - Ensure user owns meal plan (planId belongs to userId)
     │
     ├─> Validation Middleware:
     │   - Validate request body schema (go-playground/validator)
     │   - Check date, mealType, recipeId valid
     │
     ▼
┌──────────────────┐
│  MealPlanHandler │ 5. Call service layer
└────┬─────────────┘
     │
     ▼
┌──────────────────┐
│  MealPlanService │ 6. Business logic:
└────┬─────────────┘    - Fetch meal plan from DB
     │                  - Check recipe exists
     ├─────────────────> RecipeRepository.FindByID(recipeId)
     │                  - Add meal to plan
     │                  - Invalidate cache
     ▼
┌──────────────────┐
│   PostgreSQL     │ 7. Update meal_plans table (GORM):
│   (Primary DB)   │    db.Model(&mealPlan).
└────┬─────────────┘       Update("meals", updatedMeals)
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
└────┬─────────────┘     (using GORM Create)
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
│  - sync.Map or go-cache library               │
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
```go
func (s *MealPlanService) AddMealToSlot(planID, date, mealType, recipeID string) error {
    // 1. Update database
    err := s.repo.UpdateMealPlan(planID, updatedMeals)
    if err != nil {
        return err
    }

    // 2. Invalidate cache
    cacheKey := fmt.Sprintf("meal-plan:%s:%s", userID, weekStart)
    s.cache.Del(ctx, cacheKey)

    // 3. Return updated data
    return s.GetMealPlan(planID) // Will cache on read
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

### Image Upload Flow (Go Implementation)

```go
// Handler for presigned URL generation
func (h *RecipeHandler) GetUploadURL(c *gin.Context) {
    var req struct {
        Filename    string `json:"filename" binding:"required"`
        ContentType string `json:"contentType" binding:"required"`
    }

    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    // Generate presigned URL (15-minute expiration)
    s3Client := s3.NewFromConfig(cfg)
    presignClient := s3.NewPresignClient(s3Client)

    request, err := presignClient.PresignPutObject(ctx, &s3.PutObjectInput{
        Bucket:      aws.String("meal-planner-images"),
        Key:         aws.String(fmt.Sprintf("recipes/%s/temp/%s", recipeID, req.Filename)),
        ContentType: aws.String(req.ContentType),
    }, s3.WithPresignExpires(15*time.Minute))

    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate upload URL"})
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "uploadUrl": request.URL,
        "key":       request.Key,
    })
}
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

**Search Query (GORM)**:
```go
type RecipeRepository struct {
    db *gorm.DB
}

func (r *RecipeRepository) Search(query string, filters RecipeFilters) ([]Recipe, error) {
    var recipes []Recipe

    db := r.db.Model(&Recipe{})

    // Full-text search
    if query != "" {
        db = db.Where("search_vector @@ to_tsquery('english', ?)", query)
        db = db.Order("ts_rank(search_vector, to_tsquery('english', ?)) DESC", query)
    }

    // Apply filters
    if filters.Category != "" {
        db = db.Where("category = ?", filters.Category)
    }

    if len(filters.Tags) > 0 {
        db = db.Where("tags @> ?", pq.Array(filters.Tags))
    }

    // Execute query
    err := db.Limit(20).Offset(filters.Page * 20).Find(&recipes).Error
    return recipes, err
}
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
- Sync: App-level (write to ES on recipe create/update)

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

### Event Tracking (Go Implementation)

```go
type ActivityService struct {
    repo *ActivityRepository
}

type Activity struct {
    ID         string    `gorm:"primaryKey"`
    UserID     string    `gorm:"index"`
    EventType  string    `gorm:"index"`
    RecipeID   *string
    Details    string
    Timestamp  time.Time `gorm:"autoCreateTime"`
}

func (s *ActivityService) TrackMealPlanned(userID, recipeID, mealType, date string) error {
    activity := &Activity{
        UserID:    userID,
        EventType: "meal_planned",
        RecipeID:  &recipeID,
        Details:   fmt.Sprintf("Added %s to %s on %s", recipeID, mealType, date),
    }
    return s.repo.Create(activity)
}
```

### Nutrition Analysis

**Service**: Nutrition Service (Go)

**Features**:
1. **Daily/Weekly Summaries**:
   - Aggregate nutrition from all meals in plan
   - Calculate daily totals (calories, protein, carbs, fat)
   - Weekly averages

2. **Balance Score Algorithm**:
   ```go
   func CalculateBalanceScore(weeklyNutrition WeeklyNutrition) float64 {
       score := 100.0

       // Consistency: Penalize high variance in daily calories
       calorieStdDev := calculateStdDev(weeklyNutrition.DailyCalories)
       if calorieStdDev > 500 {
           score -= 20
       }

       // Macro balance: Ideal 30% protein, 40% carbs, 30% fat
       macroScore := scoreMacroDistribution(weeklyNutrition.AvgMacros)
       score = score*0.6 + macroScore*0.4

       // Variety: Unique meal count
       uniqueMeals := len(weeklyNutrition.UniqueMeals)
       if uniqueMeals < 7 {
           score -= 10
       }

       return math.Max(0, math.Min(100, score))
   }
   ```

3. **Actionable Insights**:
   - NLP-based suggestions (rule-based initially)
   - Examples: "Add more vegetables on Thursday", "Great protein balance!"

---

## Notification Service Architecture

### Email Service (Go)

```go
type EmailService struct {
    client *sendgrid.Client
}

type EmailTemplate struct {
    Name     string
    Subject  string
    Template string
}

func (s *EmailService) SendWelcomeEmail(user User) error {
    tmpl, err := template.ParseFiles("templates/welcome.html")
    if err != nil {
        return err
    }

    var body bytes.Buffer
    err = tmpl.Execute(&body, map[string]interface{}{
        "UserName":     user.Name,
        "DashboardURL": "https://app.mealplanner.com/dashboard",
    })

    if err != nil {
        return err
    }

    message := mail.NewSingleEmail(
        mail.NewEmail("Meal Planner", "noreply@mealplanner.com"),
        "Welcome to Meal Planner!",
        mail.NewEmail(user.Name, user.Email),
        body.String(),
        body.String(),
    )

    _, err = s.client.Send(message)
    return err
}
```

**Email Templates**:
- Welcome email (`welcome.html`)
- Meal plan reminder (`meal-plan-reminder.html`)
- Shopping list ready (`shopping-list-ready.html`)
- Weekly summary (`weekly-summary.html`)
- Password reset (`password-reset.html`)

**Template Engine**: Go `html/template`

**Email Scheduling**:
- Meal plan reminders: Sunday 9 AM (user's timezone)
- Weekly summary: Saturday 6 PM
- Use Go cron library (robfig/cron)

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

**Query Routing (GORM)**:
```go
// Primary DB for writes
primaryDB := gorm.Open(postgres.Open(primaryDSN))

// Read replicas
replicaDBs := []*gorm.DB{
    gorm.Open(postgres.Open(replica1DSN)),
    gorm.Open(postgres.Open(replica2DSN)),
}

// Write query → Primary
primaryDB.Create(&user)

// Read query → Random replica
replica := replicaDBs[rand.Intn(len(replicaDBs))]
replica.Find(&users)
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
│  │ │ (Go Binary) │  │        │ │ (Go Binary) │ │ │
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

**Environment Variables (Go)**:
```go
type Config struct {
    DBHost         string `env:"DB_HOST,required"`
    DBPassword     string // Loaded from Secrets Manager
    JWTSecret      string // Loaded from Secrets Manager
    SendGridAPIKey string // Loaded from Secrets Manager
}

func LoadConfig() (*Config, error) {
    // Load from environment
    cfg := &Config{}
    if err := env.Parse(cfg); err != nil {
        return nil, err
    }

    // Load secrets from AWS Secrets Manager
    secretsClient := secretsmanager.NewFromConfig(awsCfg)

    dbPass, err := getSecret(secretsClient, "production/db/password")
    if err != nil {
        return nil, err
    }
    cfg.DBPassword = dbPass

    // Load other secrets...

    return cfg, nil
}
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

### Logging (Go Structured Logging)

**Example with Logrus**:
```go
import log "github.com/sirupsen/logrus"

func init() {
    log.SetFormatter(&log.JSONFormatter{})
    log.SetOutput(os.Stdout)
    log.SetLevel(log.InfoLevel)
}

func (h *MealPlanHandler) AddMeal(c *gin.Context) {
    start := time.Now()

    // Process request...

    log.WithFields(log.Fields{
        "timestamp":  time.Now().Format(time.RFC3339),
        "service":    "api",
        "requestId":  c.GetString("requestId"),
        "userId":     c.GetString("userId"),
        "method":     c.Request.Method,
        "path":       c.Request.URL.Path,
        "statusCode": c.Writer.Status(),
        "duration":   time.Since(start).Milliseconds(),
    }).Info("Meal added successfully")
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

**AWS X-Ray (Go SDK)**:
```go
import (
    "github.com/aws/aws-xray-sdk-go/xray"
    "github.com/aws/aws-xray-sdk-go/xraygin"
)

func main() {
    r := gin.Default()

    // Add X-Ray middleware
    r.Use(xraygin.Middleware("meal-planner-api"))

    // Your routes...
}

func (s *MealPlanService) AddMeal(ctx context.Context, planID string) error {
    // Create subsegment for tracing
    _, seg := xray.BeginSubsegment(ctx, "AddMeal")
    defer seg.Close(nil)

    // Your business logic...
}
```

**Example Trace**:
```
POST /api/v1/meal-plans/plan-001/meals
├─ Auth Middleware (5ms)
├─ MealPlanHandler (145ms)
│  ├─ RecipeRepository.FindByID (45ms)
│  │  └─ Database Query (40ms)
│  ├─ MealPlanRepository.Update (85ms)
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

**Go Advantages**:
- **Performance**: 10x faster than Node.js for CPU-bound tasks
- **Concurrency**: Goroutines handle thousands of concurrent requests efficiently
- **Simplicity**: Single binary deployment, no runtime dependencies
- **Type Safety**: Compile-time error catching
- **Memory Efficiency**: Lower memory footprint than Node.js

---

**Next Steps**: Proceed to Database Design (DATABASE_DESIGN.md) for detailed schema specifications.
