# API Specification
## AI-Powered Meal Planner - Backend API

---

**Version:** 1.0
**Last Updated:** 2025-10-14
**Base URL:** `https://api.mealplanner.com/api/v1`
**Protocol:** HTTPS only

---

## Table of Contents

1. [API Overview](#api-overview)
2. [Authentication](#authentication)
3. [Standard Response Formats](#standard-response-formats)
4. [Error Handling](#error-handling)
5. [Pagination](#pagination)
6. [Rate Limiting](#rate-limiting)
7. [API Endpoints](#api-endpoints)
   - [Authentication Module](#authentication-module)
   - [Recipe Module](#recipe-module)
   - [Meal Planning Module](#meal-planning-module)
   - [Shopping List Module](#shopping-list-module)
   - [User Preferences Module](#user-preferences-module)
   - [AI/ML Module](#aiml-module)
   - [Admin Module](#admin-module)

---

## API Overview

### Versioning Strategy

**URL Path Versioning**: `/api/v1/`, `/api/v2/`

- Major version in URL path
- Breaking changes trigger new version
- Old versions supported for 12 months minimum
- Deprecation warnings in response headers

### Environments

| Environment | Base URL | Purpose |
|-------------|----------|---------|
| **Development** | `http://localhost:3000/api/v1` | Local development |
| **Staging** | `https://api-staging.mealplanner.com/api/v1` | Testing, QA |
| **Production** | `https://api.mealplanner.com/api/v1` | Live application |

### Content Types

- **Request**: `Content-Type: application/json`
- **Response**: `Content-Type: application/json`
- **File Upload**: `Content-Type: multipart/form-data`

---

## Authentication

### JWT Bearer Token

All authenticated endpoints require a JWT token in the `Authorization` header:

```http
Authorization: Bearer <access_token>
```

### Token Types

**Access Token**:
- Expiration: 1 hour
- Used for all API requests
- Stored in localStorage or httpOnly cookie

**Refresh Token**:
- Expiration: 7 days
- Used only for `/auth/refresh` endpoint
- Stored securely (httpOnly cookie recommended)

### Token Payload

```json
{
  "userId": "550e8400-e29b-41d4-a716-446655440000",
  "email": "user@example.com",
  "role": "user",
  "iat": 1697280000,
  "exp": 1697283600
}
```

---

## Standard Response Formats

### Success Response

```json
{
  "data": {
    "id": "123",
    "name": "Resource data"
  },
  "meta": {
    "timestamp": "2025-10-14T10:30:00Z",
    "requestId": "req-abc123"
  }
}
```

### List Response (with pagination)

```json
{
  "data": [
    { "id": "1", "name": "Item 1" },
    { "id": "2", "name": "Item 2" }
  ],
  "pagination": {
    "page": 1,
    "limit": 20,
    "total": 245,
    "totalPages": 13,
    "hasNext": true,
    "hasPrev": false
  },
  "meta": {
    "timestamp": "2025-10-14T10:30:00Z",
    "requestId": "req-abc123"
  }
}
```

---

## Error Handling

### Error Response Format

```json
{
  "error": {
    "code": "ERROR_CODE",
    "message": "Human-readable error message",
    "details": {
      "field": "email",
      "reason": "Email already exists"
    }
  },
  "meta": {
    "timestamp": "2025-10-14T10:30:00Z",
    "requestId": "req-abc123"
  }
}
```

### Standard Error Codes

| HTTP Status | Error Code | Description |
|-------------|-----------|-------------|
| 400 | `VALIDATION_ERROR` | Invalid request data |
| 401 | `UNAUTHORIZED` | Missing or invalid authentication token |
| 403 | `FORBIDDEN` | Insufficient permissions |
| 404 | `RESOURCE_NOT_FOUND` | Resource doesn't exist |
| 409 | `CONFLICT` | Resource already exists |
| 429 | `RATE_LIMIT_EXCEEDED` | Too many requests |
| 500 | `INTERNAL_ERROR` | Server error |
| 503 | `SERVICE_UNAVAILABLE` | Temporary service outage |

### Validation Error Example

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

---

## Pagination

### Query Parameters

```
GET /api/v1/recipes?page=1&limit=20
```

- `page`: Page number (default: 1, min: 1)
- `limit`: Items per page (default: 20, max: 100)

### Filtering

```
GET /api/v1/recipes?category=Dinner&dietary=Vegan&maxPrepTime=30
```

### Sorting

```
GET /api/v1/recipes?sort=createdAt:desc,rating:desc
```

- Format: `field:direction`
- Direction: `asc` or `desc`
- Multiple fields: comma-separated

---

## Rate Limiting

### Limits

| Endpoint Type | Rate Limit | Window |
|--------------|-----------|--------|
| **Authentication** | 10 requests | 15 minutes |
| **General API** | 100 requests | 1 minute |
| **Search** | 30 requests | 1 minute |
| **AI/ML** | 10 requests | 1 minute |

### Rate Limit Headers

```http
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 75
X-RateLimit-Reset: 1697280120
```

### Rate Limit Exceeded Response

```json
{
  "error": {
    "code": "RATE_LIMIT_EXCEEDED",
    "message": "Too many requests. Please try again later.",
    "details": {
      "retryAfter": 45
    }
  }
}
```

---

## API Endpoints

---

## Authentication Module

Base Path: `/api/v1/auth`

---

### POST /auth/register

Register a new user account.

**Authentication**: None (public)

**Request Body**:
```json
{
  "email": "sarah@example.com",
  "password": "SecurePassword123!",
  "name": "Sarah Johnson"
}
```

**Validation Rules**:
- `email`: Valid email format, unique
- `password`: Min 8 characters, 1 uppercase, 1 number, 1 special char
- `name`: Min 2 characters, max 255

**Success Response** (201 Created):
```json
{
  "data": {
    "user": {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "email": "sarah@example.com",
      "name": "Sarah Johnson",
      "role": "user",
      "createdAt": "2025-10-14T10:30:00Z"
    },
    "tokens": {
      "accessToken": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
      "refreshToken": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
      "expiresIn": 3600
    }
  }
}
```

**Error Responses**:

400 - Validation Error:
```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Invalid input data",
    "details": {
      "email": "Email is required",
      "password": "Password must contain at least 1 uppercase letter"
    }
  }
}
```

409 - Email Already Exists:
```json
{
  "error": {
    "code": "CONFLICT",
    "message": "Email already exists",
    "details": {
      "field": "email"
    }
  }
}
```

---

### POST /auth/login

Login user and return JWT tokens.

**Authentication**: None (public)

**Request Body**:
```json
{
  "email": "sarah@example.com",
  "password": "SecurePassword123!"
}
```

**Success Response** (200 OK):
```json
{
  "data": {
    "user": {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "email": "sarah@example.com",
      "name": "Sarah Johnson",
      "role": "user",
      "preferences": {
        "dietary": ["Vegan"],
        "allergies": [],
        "householdSize": 2,
        "onboardingCompleted": true
      },
      "lastLoginAt": "2025-10-14T10:30:00Z"
    },
    "tokens": {
      "accessToken": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
      "refreshToken": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
      "expiresIn": 3600
    }
  }
}
```

**Error Responses**:

401 - Invalid Credentials:
```json
{
  "error": {
    "code": "UNAUTHORIZED",
    "message": "Invalid email or password"
  }
}
```

429 - Rate Limit (after 5 failed attempts):
```json
{
  "error": {
    "code": "RATE_LIMIT_EXCEEDED",
    "message": "Too many login attempts. Please try again in 15 minutes.",
    "details": {
      "retryAfter": 900
    }
  }
}
```

---

### POST /auth/refresh

Refresh access token using refresh token.

**Authentication**: None (uses refresh token)

**Request Body**:
```json
{
  "refreshToken": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
}
```

**Success Response** (200 OK):
```json
{
  "data": {
    "accessToken": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "expiresIn": 3600
  }
}
```

**Error Responses**:

401 - Invalid Refresh Token:
```json
{
  "error": {
    "code": "UNAUTHORIZED",
    "message": "Invalid or expired refresh token"
  }
}
```

---

### POST /auth/logout

Logout user and invalidate tokens.

**Authentication**: Required (Bearer token)

**Request Body**:
```json
{
  "refreshToken": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
}
```

**Success Response** (204 No Content)

**Error Responses**:

401 - Unauthorized:
```json
{
  "error": {
    "code": "UNAUTHORIZED",
    "message": "Missing or invalid authentication token"
  }
}
```

---

### POST /auth/forgot-password

Request password reset email.

**Authentication**: None (public)

**Request Body**:
```json
{
  "email": "sarah@example.com"
}
```

**Success Response** (200 OK):
```json
{
  "data": {
    "message": "If the email exists, a password reset link has been sent."
  }
}
```

Note: Always returns success even if email doesn't exist (security best practice).

---

### POST /auth/reset-password

Reset password using reset token.

**Authentication**: None (uses reset token)

**Request Body**:
```json
{
  "token": "reset-token-from-email",
  "newPassword": "NewSecurePassword123!"
}
```

**Success Response** (200 OK):
```json
{
  "data": {
    "message": "Password reset successful. You can now login with your new password."
  }
}
```

**Error Responses**:

400 - Invalid or Expired Token:
```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Invalid or expired reset token"
  }
}
```

---

### GET /auth/me

Get current user profile.

**Authentication**: Required (Bearer token)

**Success Response** (200 OK):
```json
{
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "email": "sarah@example.com",
    "name": "Sarah Johnson",
    "role": "user",
    "avatarUrl": "https://cdn.mealplanner.com/users/550e8400.../avatar.webp",
    "bio": "Food lover and home cook",
    "preferences": {
      "dietary": ["Vegan", "Gluten-Free"],
      "allergies": ["Peanuts"],
      "householdSize": 2,
      "onboardingCompleted": true
    },
    "createdAt": "2025-09-15T10:00:00Z",
    "lastLoginAt": "2025-10-14T10:30:00Z"
  }
}
```

---

## Recipe Module

Base Path: `/api/v1/recipes`

---

### GET /recipes

List recipes with filtering, sorting, and pagination.

**Authentication**: Required (Bearer token)

**Query Parameters**:
- `page`: Page number (default: 1)
- `limit`: Items per page (default: 20, max: 100)
- `category`: Filter by category (Breakfast, Lunch, Dinner, Snacks, Desserts)
- `dietary`: Filter by dietary tag (Vegan, Keto, Gluten-Free, etc.)
- `cuisine`: Filter by cuisine (Italian, Mexican, Chinese, etc.)
- `maxPrepTime`: Max prep time in minutes
- `maxCookTime`: Max cook time in minutes
- `difficulty`: Filter by difficulty (Easy, Medium, Hard)
- `q`: Search query (full-text search)
- `sort`: Sort order (e.g., `rating:desc,createdAt:desc`)

**Request Example**:
```http
GET /api/v1/recipes?category=Dinner&dietary=Vegan&maxPrepTime=30&page=1&limit=20&sort=rating:desc
```

**Success Response** (200 OK):
```json
{
  "data": [
    {
      "id": "recipe-001",
      "name": "Vegan Buddha Bowl",
      "description": "Nutritious and colorful vegan bowl",
      "category": "Dinner",
      "cuisine": "Fusion",
      "prepTime": 20,
      "cookTime": 15,
      "totalTime": 35,
      "servings": 2,
      "difficulty": "Easy",
      "imageUrl": "https://cdn.mealplanner.com/recipes/recipe-001/medium.webp",
      "rating": 4.8,
      "reviewCount": 156,
      "tags": ["Vegan", "Gluten-Free", "High-Protein"],
      "nutrition": {
        "calories": 420,
        "protein": 18,
        "carbohydrates": 52,
        "fat": 16,
        "fiber": 12
      },
      "isPublic": true,
      "isFeatured": true,
      "createdAt": "2025-08-15T10:00:00Z"
    }
  ],
  "pagination": {
    "page": 1,
    "limit": 20,
    "total": 45,
    "totalPages": 3,
    "hasNext": true,
    "hasPrev": false
  }
}
```

---

### GET /recipes/:id

Get recipe details by ID.

**Authentication**: Required (Bearer token)

**Path Parameters**:
- `id`: Recipe UUID

**Success Response** (200 OK):
```json
{
  "data": {
    "id": "recipe-001",
    "name": "Chicken Tacos",
    "description": "Delicious and easy chicken tacos with fresh toppings",
    "category": "Dinner",
    "cuisine": "Mexican",
    "prepTime": 15,
    "cookTime": 20,
    "totalTime": 35,
    "servings": 4,
    "difficulty": "Easy",
    "ingredients": [
      {
        "name": "Chicken breast",
        "quantity": "1",
        "unit": "lb",
        "category": "Meat"
      },
      {
        "name": "Taco seasoning",
        "quantity": "2",
        "unit": "tbsp",
        "category": "Pantry"
      },
      {
        "name": "Tortillas",
        "quantity": "8",
        "unit": "",
        "category": "Bakery"
      },
      {
        "name": "Lettuce",
        "quantity": "1",
        "unit": "cup",
        "category": "Produce"
      },
      {
        "name": "Tomatoes",
        "quantity": "2",
        "unit": "",
        "category": "Produce"
      },
      {
        "name": "Cheese",
        "quantity": "1",
        "unit": "cup",
        "category": "Dairy"
      }
    ],
    "instructions": [
      "Season chicken breast with taco seasoning on both sides",
      "Heat skillet over medium-high heat and cook chicken for 8-10 minutes per side",
      "Remove chicken and let rest for 5 minutes, then shred with two forks",
      "Warm tortillas in microwave or on stovetop",
      "Assemble tacos with chicken, lettuce, tomatoes, and cheese",
      "Serve immediately with optional toppings like sour cream and salsa"
    ],
    "nutrition": {
      "calories": 350,
      "protein": 28,
      "carbohydrates": 35,
      "fat": 10,
      "fiber": 5,
      "sodium": 680,
      "sugar": 3
    },
    "tags": ["Mexican", "Quick", "Family-Friendly", "High-Protein"],
    "imageUrl": "https://cdn.mealplanner.com/recipes/recipe-001/medium.webp",
    "imageUrls": {
      "original": "https://cdn.mealplanner.com/recipes/recipe-001/original.jpg",
      "large": "https://cdn.mealplanner.com/recipes/recipe-001/large.webp",
      "medium": "https://cdn.mealplanner.com/recipes/recipe-001/medium.webp",
      "thumbnail": "https://cdn.mealplanner.com/recipes/recipe-001/thumbnail.webp"
    },
    "rating": 4.5,
    "reviewCount": 89,
    "isPublic": true,
    "isFeatured": false,
    "createdById": "admin-001",
    "createdAt": "2025-08-01T10:00:00Z",
    "updatedAt": "2025-10-10T14:20:00Z"
  }
}
```

**Error Responses**:

404 - Recipe Not Found:
```json
{
  "error": {
    "code": "RESOURCE_NOT_FOUND",
    "message": "Recipe not found"
  }
}
```

---

### POST /recipes

Create a new recipe (admin only).

**Authentication**: Required (Bearer token, admin role)

**Request Body**:
```json
{
  "name": "Vegan Stir Fry",
  "description": "Quick and healthy vegetable stir fry",
  "category": "Dinner",
  "cuisine": "Asian",
  "prepTime": 15,
  "cookTime": 10,
  "servings": 4,
  "difficulty": "Easy",
  "ingredients": [
    {
      "name": "Mixed vegetables",
      "quantity": "4",
      "unit": "cups",
      "category": "Produce"
    },
    {
      "name": "Soy sauce",
      "quantity": "3",
      "unit": "tbsp",
      "category": "Pantry"
    },
    {
      "name": "Garlic",
      "quantity": "3",
      "unit": "cloves",
      "category": "Produce"
    },
    {
      "name": "Ginger",
      "quantity": "1",
      "unit": "tbsp",
      "category": "Produce"
    }
  ],
  "instructions": [
    "Heat oil in large wok or skillet",
    "Add garlic and ginger, cook for 30 seconds",
    "Add vegetables and stir fry for 5-7 minutes",
    "Add soy sauce and toss to coat",
    "Serve over rice or noodles"
  ],
  "nutrition": {
    "calories": 180,
    "protein": 8,
    "carbohydrates": 28,
    "fat": 5,
    "fiber": 6
  },
  "tags": ["Vegan", "Quick", "Healthy", "Asian"],
  "imageUrl": "https://cdn.mealplanner.com/recipes/recipe-new/medium.webp"
}
```

**Success Response** (201 Created):
```json
{
  "data": {
    "id": "recipe-new-uuid",
    "name": "Vegan Stir Fry",
    "description": "Quick and healthy vegetable stir fry",
    "category": "Dinner",
    "cuisine": "Asian",
    "prepTime": 15,
    "cookTime": 10,
    "totalTime": 25,
    "servings": 4,
    "difficulty": "Easy",
    "ingredients": [...],
    "instructions": [...],
    "nutrition": {...},
    "tags": ["Vegan", "Quick", "Healthy", "Asian"],
    "imageUrl": "https://cdn.mealplanner.com/recipes/recipe-new/medium.webp",
    "rating": 0.0,
    "reviewCount": 0,
    "isPublic": true,
    "isFeatured": false,
    "createdById": "admin-001",
    "createdAt": "2025-10-14T10:30:00Z",
    "updatedAt": "2025-10-14T10:30:00Z"
  }
}
```

**Error Responses**:

403 - Forbidden (not admin):
```json
{
  "error": {
    "code": "FORBIDDEN",
    "message": "Insufficient permissions. Admin role required."
  }
}
```

---

### PUT /recipes/:id

Update a recipe (admin only).

**Authentication**: Required (Bearer token, admin role)

**Path Parameters**:
- `id`: Recipe UUID

**Request Body** (partial update supported):
```json
{
  "name": "Updated Recipe Name",
  "prepTime": 20,
  "tags": ["Vegan", "Quick", "Easy"]
}
```

**Success Response** (200 OK):
```json
{
  "data": {
    "id": "recipe-001",
    "name": "Updated Recipe Name",
    "prepTime": 20,
    "updatedAt": "2025-10-14T10:30:00Z",
    ...
  }
}
```

---

### DELETE /recipes/:id

Delete a recipe (admin only).

**Authentication**: Required (Bearer token, admin role)

**Path Parameters**:
- `id`: Recipe UUID

**Success Response** (204 No Content)

**Error Responses**:

404 - Recipe Not Found:
```json
{
  "error": {
    "code": "RESOURCE_NOT_FOUND",
    "message": "Recipe not found"
  }
}
```

---

### GET /recipes/search

Search recipes with advanced full-text search.

**Authentication**: Required (Bearer token)

**Query Parameters**:
- `q`: Search query (required)
- `category`: Filter by category
- `dietary`: Filter by dietary tag
- `page`: Page number
- `limit`: Items per page

**Request Example**:
```http
GET /api/v1/recipes/search?q=chicken%20tacos&category=Dinner&page=1&limit=10
```

**Success Response** (200 OK):
```json
{
  "data": [
    {
      "id": "recipe-001",
      "name": "Chicken Tacos",
      "description": "Delicious and easy chicken tacos",
      "imageUrl": "https://cdn.mealplanner.com/recipes/recipe-001/medium.webp",
      "category": "Dinner",
      "prepTime": 15,
      "cookTime": 20,
      "rating": 4.5,
      "relevanceScore": 0.95
    }
  ],
  "pagination": {
    "page": 1,
    "limit": 10,
    "total": 8,
    "totalPages": 1,
    "hasNext": false,
    "hasPrev": false
  }
}
```

---

### GET /recipes/categories

Get list of all recipe categories.

**Authentication**: Required (Bearer token)

**Success Response** (200 OK):
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

### POST /recipes/:id/favorite

Add recipe to user's favorites.

**Authentication**: Required (Bearer token)

**Path Parameters**:
- `id`: Recipe UUID

**Success Response** (200 OK):
```json
{
  "data": {
    "recipeId": "recipe-001",
    "userId": "user-001",
    "favoritedAt": "2025-10-14T10:30:00Z"
  }
}
```

**Error Responses**:

409 - Already Favorited:
```json
{
  "error": {
    "code": "CONFLICT",
    "message": "Recipe already in favorites"
  }
}
```

---

### DELETE /recipes/:id/favorite

Remove recipe from user's favorites.

**Authentication**: Required (Bearer token)

**Path Parameters**:
- `id`: Recipe UUID

**Success Response** (204 No Content)

---

### GET /recipes/favorites

Get user's favorited recipes.

**Authentication**: Required (Bearer token)

**Query Parameters**:
- `page`: Page number
- `limit`: Items per page

**Success Response** (200 OK):
```json
{
  "data": [
    {
      "id": "recipe-001",
      "name": "Chicken Tacos",
      "imageUrl": "https://cdn.mealplanner.com/recipes/recipe-001/medium.webp",
      "category": "Dinner",
      "prepTime": 15,
      "cookTime": 20,
      "rating": 4.5,
      "favoritedAt": "2025-10-10T14:20:00Z"
    }
  ],
  "pagination": {
    "page": 1,
    "limit": 20,
    "total": 12,
    "totalPages": 1,
    "hasNext": false,
    "hasPrev": false
  }
}
```

---

## Meal Planning Module

Base Path: `/api/v1/meal-plans`

---

### GET /meal-plans

List user's meal plans.

**Authentication**: Required (Bearer token)

**Query Parameters**:
- `weekStart`: Filter by week start date (YYYY-MM-DD, must be Sunday)
- `page`: Page number
- `limit`: Items per page

**Request Example**:
```http
GET /api/v1/meal-plans?weekStart=2025-10-13
```

**Success Response** (200 OK):
```json
{
  "data": [
    {
      "id": "plan-001",
      "userId": "user-001",
      "weekStart": "2025-10-13",
      "meals": {
        "sunday": {
          "breakfast": "recipe-001",
          "lunch": null,
          "dinner": "recipe-002",
          "snacks": []
        },
        "monday": {
          "breakfast": "recipe-003",
          "lunch": "recipe-004",
          "dinner": "recipe-005",
          "snacks": ["recipe-006"]
        },
        "tuesday": {
          "breakfast": null,
          "lunch": null,
          "dinner": "recipe-007",
          "snacks": []
        },
        "wednesday": {
          "breakfast": "recipe-008",
          "lunch": "recipe-009",
          "dinner": null,
          "snacks": []
        },
        "thursday": {
          "breakfast": null,
          "lunch": "recipe-010",
          "dinner": "recipe-011",
          "snacks": []
        },
        "friday": {
          "breakfast": "recipe-012",
          "lunch": null,
          "dinner": "recipe-013",
          "snacks": []
        },
        "saturday": {
          "breakfast": null,
          "lunch": "recipe-014",
          "dinner": null,
          "snacks": ["recipe-015"]
        }
      },
      "isActive": true,
      "createdAt": "2025-10-13T08:00:00Z",
      "updatedAt": "2025-10-14T10:30:00Z"
    }
  ],
  "pagination": {
    "page": 1,
    "limit": 20,
    "total": 1,
    "totalPages": 1,
    "hasNext": false,
    "hasPrev": false
  }
}
```

---

### GET /meal-plans/:id

Get meal plan details with populated recipe data.

**Authentication**: Required (Bearer token)

**Path Parameters**:
- `id`: Meal plan UUID

**Success Response** (200 OK):
```json
{
  "data": {
    "id": "plan-001",
    "userId": "user-001",
    "weekStart": "2025-10-13",
    "meals": {
      "monday": {
        "breakfast": {
          "id": "recipe-003",
          "name": "Overnight Oats",
          "imageUrl": "https://cdn.mealplanner.com/recipes/recipe-003/thumbnail.webp",
          "prepTime": 10,
          "nutrition": {
            "calories": 320,
            "protein": 12
          }
        },
        "lunch": {
          "id": "recipe-004",
          "name": "Caesar Salad",
          "imageUrl": "https://cdn.mealplanner.com/recipes/recipe-004/thumbnail.webp",
          "prepTime": 15,
          "nutrition": {
            "calories": 280,
            "protein": 18
          }
        },
        "dinner": {
          "id": "recipe-005",
          "name": "Grilled Salmon",
          "imageUrl": "https://cdn.mealplanner.com/recipes/recipe-005/thumbnail.webp",
          "prepTime": 10,
          "cookTime": 15,
          "nutrition": {
            "calories": 420,
            "protein": 35
          }
        },
        "snacks": []
      },
      "tuesday": {
        "breakfast": null,
        "lunch": null,
        "dinner": {
          "id": "recipe-007",
          "name": "Chicken Tacos",
          "imageUrl": "https://cdn.mealplanner.com/recipes/recipe-007/thumbnail.webp",
          "prepTime": 15,
          "cookTime": 20,
          "nutrition": {
            "calories": 350,
            "protein": 28
          }
        },
        "snacks": []
      }
    },
    "summary": {
      "totalMeals": 15,
      "avgCaloriesPerDay": 1850,
      "avgProteinPerDay": 95
    },
    "isActive": true,
    "createdAt": "2025-10-13T08:00:00Z",
    "updatedAt": "2025-10-14T10:30:00Z"
  }
}
```

---

### POST /meal-plans

Create a new meal plan for a week.

**Authentication**: Required (Bearer token)

**Request Body**:
```json
{
  "weekStart": "2025-10-13",
  "meals": {}
}
```

**Success Response** (201 Created):
```json
{
  "data": {
    "id": "plan-new-uuid",
    "userId": "user-001",
    "weekStart": "2025-10-13",
    "meals": {},
    "isActive": true,
    "createdAt": "2025-10-14T10:30:00Z",
    "updatedAt": "2025-10-14T10:30:00Z"
  }
}
```

**Error Responses**:

409 - Meal Plan Already Exists:
```json
{
  "error": {
    "code": "CONFLICT",
    "message": "Meal plan for this week already exists"
  }
}
```

---

### PUT /meal-plans/:id

Update meal plan.

**Authentication**: Required (Bearer token)

**Path Parameters**:
- `id`: Meal plan UUID

**Request Body**:
```json
{
  "meals": {
    "monday": {
      "breakfast": "recipe-001",
      "lunch": null,
      "dinner": "recipe-002",
      "snacks": []
    }
  }
}
```

**Success Response** (200 OK):
```json
{
  "data": {
    "id": "plan-001",
    "userId": "user-001",
    "weekStart": "2025-10-13",
    "meals": {...},
    "updatedAt": "2025-10-14T10:35:00Z"
  }
}
```

---

### DELETE /meal-plans/:id

Delete meal plan.

**Authentication**: Required (Bearer token)

**Path Parameters**:
- `id`: Meal plan UUID

**Success Response** (204 No Content)

---

### POST /meal-plans/:id/meals

Add meal to a specific slot.

**Authentication**: Required (Bearer token)

**Path Parameters**:
- `id`: Meal plan UUID

**Request Body**:
```json
{
  "day": "monday",
  "mealType": "dinner",
  "recipeId": "recipe-001"
}
```

**Validation Rules**:
- `day`: One of: sunday, monday, tuesday, wednesday, thursday, friday, saturday
- `mealType`: One of: breakfast, lunch, dinner, snacks
- `recipeId`: Valid recipe UUID

**Success Response** (200 OK):
```json
{
  "data": {
    "id": "plan-001",
    "userId": "user-001",
    "weekStart": "2025-10-13",
    "meals": {
      "monday": {
        "breakfast": null,
        "lunch": null,
        "dinner": "recipe-001",
        "snacks": []
      }
    },
    "updatedAt": "2025-10-14T10:40:00Z"
  }
}
```

---

### DELETE /meal-plans/:id/meals/:mealId

Remove meal from a specific slot.

**Authentication**: Required (Bearer token)

**Path Parameters**:
- `id`: Meal plan UUID
- `mealId`: Not used (kept for API consistency)

**Request Body**:
```json
{
  "day": "monday",
  "mealType": "dinner"
}
```

**Success Response** (200 OK):
```json
{
  "data": {
    "id": "plan-001",
    "meals": {
      "monday": {
        "breakfast": null,
        "lunch": null,
        "dinner": null,
        "snacks": []
      }
    },
    "updatedAt": "2025-10-14T10:45:00Z"
  }
}
```

---

### POST /meal-plans/:id/copy-day

Copy meals from one day to another.

**Authentication**: Required (Bearer token)

**Path Parameters**:
- `id`: Meal plan UUID

**Request Body**:
```json
{
  "sourceDay": "monday",
  "targetDay": "tuesday"
}
```

**Success Response** (200 OK):
```json
{
  "data": {
    "id": "plan-001",
    "meals": {
      "monday": {
        "breakfast": "recipe-003",
        "lunch": "recipe-004",
        "dinner": "recipe-005",
        "snacks": []
      },
      "tuesday": {
        "breakfast": "recipe-003",
        "lunch": "recipe-004",
        "dinner": "recipe-005",
        "snacks": []
      }
    },
    "updatedAt": "2025-10-14T10:50:00Z"
  }
}
```

---

## Shopping List Module

Base Path: `/api/v1/shopping-lists`

---

### GET /shopping-lists

List user's shopping lists.

**Authentication**: Required (Bearer token)

**Query Parameters**:
- `page`: Page number
- `limit`: Items per page

**Success Response** (200 OK):
```json
{
  "data": [
    {
      "id": "list-001",
      "userId": "user-001",
      "mealPlanId": "plan-001",
      "weekStart": "2025-10-13",
      "checkedCount": 5,
      "totalCount": 23,
      "shareId": null,
      "createdAt": "2025-10-13T09:00:00Z",
      "updatedAt": "2025-10-14T10:30:00Z"
    }
  ],
  "pagination": {
    "page": 1,
    "limit": 20,
    "total": 1,
    "totalPages": 1
  }
}
```

---

### GET /shopping-lists/:id

Get shopping list details with all items.

**Authentication**: Required (Bearer token)

**Path Parameters**:
- `id`: Shopping list UUID

**Success Response** (200 OK):
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
        "recipeIds": ["recipe-001", "recipe-005"],
        "addedManually": false
      },
      {
        "id": "item-002",
        "name": "Eggs",
        "quantity": "12",
        "unit": "",
        "category": "Dairy & Eggs",
        "checked": true,
        "recipeIds": ["recipe-003", "recipe-008"],
        "addedManually": false
      },
      {
        "id": "item-003",
        "name": "Olive oil",
        "quantity": "1",
        "unit": "bottle",
        "category": "Pantry",
        "checked": false,
        "recipeIds": [],
        "addedManually": true
      }
    ],
    "checkedCount": 1,
    "totalCount": 3,
    "shareId": "abc123xyz",
    "createdAt": "2025-10-13T09:00:00Z",
    "updatedAt": "2025-10-14T10:30:00Z"
  }
}
```

---

### POST /shopping-lists/generate

Generate shopping list from meal plan.

**Authentication**: Required (Bearer token)

**Request Body**:
```json
{
  "mealPlanId": "plan-001"
}
```

**Success Response** (201 Created):
```json
{
  "data": {
    "id": "list-new-uuid",
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
        "recipeIds": ["recipe-001", "recipe-005"],
        "addedManually": false
      }
    ],
    "checkedCount": 0,
    "totalCount": 23,
    "shareId": null,
    "createdAt": "2025-10-14T11:00:00Z",
    "updatedAt": "2025-10-14T11:00:00Z"
  }
}
```

---

### PUT /shopping-lists/:id

Update shopping list (e.g., re-generate).

**Authentication**: Required (Bearer token)

**Path Parameters**:
- `id`: Shopping list UUID

**Request Body**:
```json
{
  "regenerate": true
}
```

**Success Response** (200 OK):
```json
{
  "data": {
    "id": "list-001",
    "items": [...],
    "updatedAt": "2025-10-14T11:05:00Z"
  }
}
```

---

### DELETE /shopping-lists/:id

Delete shopping list.

**Authentication**: Required (Bearer token)

**Path Parameters**:
- `id`: Shopping list UUID

**Success Response** (204 No Content)

---

### PATCH /shopping-lists/:id/items/:itemId

Update shopping list item (check/uncheck, change quantity).

**Authentication**: Required (Bearer token)

**Path Parameters**:
- `id`: Shopping list UUID
- `itemId`: Item ID (from items array)

**Request Body**:
```json
{
  "checked": true,
  "quantity": "3",
  "unit": "lbs"
}
```

**Success Response** (200 OK):
```json
{
  "data": {
    "id": "list-001",
    "items": [
      {
        "id": "item-001",
        "name": "Chicken breast",
        "quantity": "3",
        "unit": "lbs",
        "category": "Meat & Seafood",
        "checked": true,
        "recipeIds": ["recipe-001", "recipe-005"],
        "addedManually": false
      }
    ],
    "checkedCount": 6,
    "totalCount": 23,
    "updatedAt": "2025-10-14T11:10:00Z"
  }
}
```

---

### POST /shopping-lists/:id/items

Add manual item to shopping list.

**Authentication**: Required (Bearer token)

**Path Parameters**:
- `id`: Shopping list UUID

**Request Body**:
```json
{
  "name": "Milk",
  "quantity": "1",
  "unit": "gallon",
  "category": "Dairy & Eggs"
}
```

**Success Response** (201 Created):
```json
{
  "data": {
    "id": "list-001",
    "items": [
      ...,
      {
        "id": "item-new-uuid",
        "name": "Milk",
        "quantity": "1",
        "unit": "gallon",
        "category": "Dairy & Eggs",
        "checked": false,
        "recipeIds": [],
        "addedManually": true
      }
    ],
    "totalCount": 24,
    "updatedAt": "2025-10-14T11:15:00Z"
  }
}
```

---

### DELETE /shopping-lists/:id/items/:itemId

Delete item from shopping list.

**Authentication**: Required (Bearer token)

**Path Parameters**:
- `id`: Shopping list UUID
- `itemId`: Item ID

**Success Response** (200 OK):
```json
{
  "data": {
    "id": "list-001",
    "items": [...],
    "totalCount": 22,
    "updatedAt": "2025-10-14T11:20:00Z"
  }
}
```

---

## User Preferences Module

Base Path: `/api/v1/users`

---

### GET /users/preferences

Get current user's preferences.

**Authentication**: Required (Bearer token)

**Success Response** (200 OK):
```json
{
  "data": {
    "dietary": ["Vegan", "Gluten-Free"],
    "allergies": ["Peanuts", "Shellfish"],
    "householdSize": 2,
    "onboardingCompleted": true,
    "notificationPreferences": {
      "email": {
        "weeklyReminder": true,
        "shoppingListReady": true,
        "newRecipeSuggestions": false
      },
      "push": {
        "mealPrepReminders": false
      }
    }
  }
}
```

---

### PUT /users/preferences

Update user preferences.

**Authentication**: Required (Bearer token)

**Request Body**:
```json
{
  "dietary": ["Vegan", "Gluten-Free", "Low-Carb"],
  "allergies": ["Peanuts"],
  "householdSize": 3,
  "onboardingCompleted": true
}
```

**Success Response** (200 OK):
```json
{
  "data": {
    "dietary": ["Vegan", "Gluten-Free", "Low-Carb"],
    "allergies": ["Peanuts"],
    "householdSize": 3,
    "onboardingCompleted": true,
    "updatedAt": "2025-10-14T11:25:00Z"
  }
}
```

---

### PUT /users/profile

Update user profile (name, avatar, bio).

**Authentication**: Required (Bearer token)

**Request Body**:
```json
{
  "name": "Sarah Marie Johnson",
  "bio": "Food blogger and nutrition enthusiast",
  "avatarUrl": "https://cdn.mealplanner.com/users/user-001/avatar.webp"
}
```

**Success Response** (200 OK):
```json
{
  "data": {
    "id": "user-001",
    "email": "sarah@example.com",
    "name": "Sarah Marie Johnson",
    "bio": "Food blogger and nutrition enthusiast",
    "avatarUrl": "https://cdn.mealplanner.com/users/user-001/avatar.webp",
    "updatedAt": "2025-10-14T11:30:00Z"
  }
}
```

---

### PUT /users/password

Change user password.

**Authentication**: Required (Bearer token)

**Request Body**:
```json
{
  "currentPassword": "OldPassword123!",
  "newPassword": "NewSecurePassword456!"
}
```

**Success Response** (200 OK):
```json
{
  "data": {
    "message": "Password updated successfully"
  }
}
```

**Error Responses**:

401 - Incorrect Current Password:
```json
{
  "error": {
    "code": "UNAUTHORIZED",
    "message": "Current password is incorrect"
  }
}
```

---

### DELETE /users/account

Delete user account (GDPR compliance).

**Authentication**: Required (Bearer token)

**Request Body**:
```json
{
  "password": "CurrentPassword123!",
  "confirmDeletion": true
}
```

**Success Response** (204 No Content)

Note: This performs a soft delete. User data is anonymized after 30 days.

---

## AI/ML Module

Base Path: `/api/v1/ai`

---

### POST /ai/suggestions

Get personalized meal suggestions.

**Authentication**: Required (Bearer token)

**Request Body**:
```json
{
  "mealType": "dinner",
  "date": "2025-10-15",
  "count": 5,
  "excludeRecipeIds": ["recipe-001", "recipe-002"]
}
```

**Success Response** (200 OK):
```json
{
  "data": {
    "suggestions": [
      {
        "recipeId": "recipe-010",
        "recipe": {
          "id": "recipe-010",
          "name": "Thai Green Curry",
          "imageUrl": "https://cdn.mealplanner.com/recipes/recipe-010/medium.webp",
          "category": "Dinner",
          "prepTime": 20,
          "cookTime": 25,
          "rating": 4.7,
          "tags": ["Vegan", "Thai", "Spicy"]
        },
        "score": 0.92,
        "reason": "Based on your preference for Asian cuisine and vegan meals"
      },
      {
        "recipeId": "recipe-015",
        "recipe": {
          "id": "recipe-015",
          "name": "Mediterranean Bowl",
          "imageUrl": "https://cdn.mealplanner.com/recipes/recipe-015/medium.webp",
          "category": "Dinner",
          "prepTime": 15,
          "cookTime": 0,
          "rating": 4.5,
          "tags": ["Vegan", "Mediterranean", "Quick"]
        },
        "score": 0.88,
        "reason": "Similar to recipes you've favorited recently"
      }
    ],
    "generatedAt": "2025-10-14T11:35:00Z"
  }
}
```

**Error Responses**:

503 - AI Service Unavailable:
```json
{
  "error": {
    "code": "SERVICE_UNAVAILABLE",
    "message": "AI service temporarily unavailable. Please try again later."
  }
}
```

---

### POST /ai/nutrition-analysis

Analyze nutrition balance for a meal plan.

**Authentication**: Required (Bearer token)

**Request Body**:
```json
{
  "mealPlanId": "plan-001"
}
```

**Success Response** (200 OK):
```json
{
  "data": {
    "summary": {
      "avgCaloriesPerDay": 1850,
      "avgProteinPerDay": 95,
      "avgCarbsPerDay": 185,
      "avgFatPerDay": 62
    },
    "dailyBreakdown": {
      "monday": {
        "calories": 1920,
        "protein": 98,
        "carbohydrates": 190,
        "fat": 65
      },
      "tuesday": {
        "calories": 1780,
        "protein": 92,
        "carbohydrates": 180,
        "fat": 59
      }
    },
    "balanceScore": 85,
    "insights": [
      {
        "type": "positive",
        "message": "Great protein balance across the week!"
      },
      {
        "type": "suggestion",
        "message": "Consider adding more vegetables on Thursday and Friday"
      },
      {
        "type": "warning",
        "message": "Calorie variance is high. Try to keep daily intake more consistent"
      }
    ],
    "macroDistribution": {
      "protein": 21,
      "carbohydrates": 40,
      "fat": 30
    },
    "variety": {
      "uniqueMeals": 18,
      "repeatedMeals": 3,
      "varietyScore": 92
    },
    "analyzedAt": "2025-10-14T11:40:00Z"
  }
}
```

---

### POST /ai/substitutions

Get ingredient substitution suggestions.

**Authentication**: Required (Bearer token)

**Request Body**:
```json
{
  "ingredient": "chicken breast",
  "reason": "dietary",
  "dietary": ["Vegan"]
}
```

**Success Response** (200 OK):
```json
{
  "data": {
    "original": "chicken breast",
    "substitutions": [
      {
        "ingredient": "extra-firm tofu",
        "ratio": "1:1",
        "reason": "High protein, similar texture when pressed and marinated",
        "nutritionImpact": {
          "calories": -30,
          "protein": -8,
          "fat": +2
        }
      },
      {
        "ingredient": "tempeh",
        "ratio": "1:1",
        "reason": "Nutty flavor, excellent protein source",
        "nutritionImpact": {
          "calories": -20,
          "protein": -5,
          "fat": +3
        }
      },
      {
        "ingredient": "seitan",
        "ratio": "1:1",
        "reason": "Very high protein, meat-like texture",
        "nutritionImpact": {
          "calories": -10,
          "protein": -2,
          "fat": +1
        }
      }
    ],
    "generatedAt": "2025-10-14T11:45:00Z"
  }
}
```

---

## Admin Module

Base Path: `/api/v1/admin`

---

### GET /admin/users

List all users (admin only).

**Authentication**: Required (Bearer token, admin role)

**Query Parameters**:
- `page`: Page number
- `limit`: Items per page
- `role`: Filter by role (user, admin, moderator)
- `search`: Search by email or name

**Success Response** (200 OK):
```json
{
  "data": [
    {
      "id": "user-001",
      "email": "sarah@example.com",
      "name": "Sarah Johnson",
      "role": "user",
      "createdAt": "2025-09-15T10:00:00Z",
      "lastLoginAt": "2025-10-14T09:30:00Z",
      "mealPlansCount": 8,
      "favoritesCount": 23
    }
  ],
  "pagination": {
    "page": 1,
    "limit": 50,
    "total": 10234,
    "totalPages": 205
  }
}
```

---

### GET /admin/analytics

Get dashboard analytics (admin only).

**Authentication**: Required (Bearer token, admin role)

**Success Response** (200 OK):
```json
{
  "data": {
    "users": {
      "total": 10234,
      "active": 4567,
      "newThisWeek": 156,
      "newThisMonth": 678
    },
    "recipes": {
      "total": 1234,
      "published": 1156,
      "drafts": 78
    },
    "mealPlans": {
      "totalCreated": 45678,
      "activeThisWeek": 3456
    },
    "shoppingLists": {
      "totalGenerated": 23456,
      "generatedThisWeek": 1234
    },
    "engagement": {
      "avgMealsPerWeek": 12.5,
      "avgRecipesPerPlan": 8.3,
      "favoriteRate": 0.23
    },
    "topRecipes": [
      {
        "id": "recipe-001",
        "name": "Chicken Tacos",
        "timesUsed": 2345,
        "rating": 4.5
      }
    ],
    "generatedAt": "2025-10-14T11:50:00Z"
  }
}
```

---

### GET /admin/health

Get system health metrics (admin only).

**Authentication**: Required (Bearer token, admin role)

**Success Response** (200 OK):
```json
{
  "data": {
    "api": {
      "status": "healthy",
      "uptime": 345678,
      "version": "1.0.0"
    },
    "database": {
      "status": "healthy",
      "connections": 45,
      "maxConnections": 100,
      "avgQueryTime": 23
    },
    "cache": {
      "status": "healthy",
      "hitRate": 0.87,
      "memoryUsed": "1.2 GB",
      "memoryTotal": "4 GB"
    },
    "storage": {
      "status": "healthy",
      "totalSize": "234 GB",
      "usedSize": "45 GB"
    },
    "aiService": {
      "status": "healthy",
      "avgResponseTime": 1250
    },
    "checkedAt": "2025-10-14T11:55:00Z"
  }
}
```

---

## Appendix

### HTTP Status Code Summary

- **200 OK**: Successful GET, PUT, PATCH
- **201 Created**: Successful POST (resource created)
- **204 No Content**: Successful DELETE (no response body)
- **400 Bad Request**: Validation error
- **401 Unauthorized**: Missing or invalid auth token
- **403 Forbidden**: Insufficient permissions
- **404 Not Found**: Resource doesn't exist
- **409 Conflict**: Resource already exists
- **429 Too Many Requests**: Rate limit exceeded
- **500 Internal Server Error**: Server error
- **503 Service Unavailable**: Temporary service outage

### Common Headers

**Request Headers**:
```http
Authorization: Bearer <token>
Content-Type: application/json
Accept: application/json
X-Client-Version: 1.0.0
```

**Response Headers**:
```http
Content-Type: application/json
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 75
X-RateLimit-Reset: 1697280120
X-Request-ID: req-abc123
```

---

**Document Version:** 1.0
**Last Updated:** 2025-10-14
**Status:** Complete

This API specification provides comprehensive documentation for all backend endpoints. For implementation details, refer to the Architecture and Database Design documents.
