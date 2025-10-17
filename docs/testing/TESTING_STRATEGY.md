# Testing Strategy
## Comprehensive Backend Testing Approach

---

**Version:** 1.0
**Last Updated:** 2025-10-14
**Coverage Target:** 80%+

---

## Testing Pyramid

```
           /\
          /  \
         / E2E\ (10%)
        /------\
       /        \
      / Integration\ (30%)
     /------------\
    /              \
   /   Unit Tests   \ (60%)
  /__________________\
```

**Distribution:**
- Unit Tests: 60% (fast, isolated, many)
- Integration Tests: 30% (API endpoints with DB)
- E2E Tests: 10% (full user flows)

---

## Unit Testing

### Tools
- **Jest:** Test runner, assertions, mocking
- **Coverage:** Istanbul (built into Jest)

### Configuration

```javascript
// jest.config.js
module.exports = {
  preset: 'ts-jest',
  testEnvironment: 'node',
  roots: ['<rootDir>/src'],
  testMatch: ['**/__tests__/**/*.test.ts'],
  collectCoverageFrom: [
    'src/**/*.{ts,js}',
    '!src/**/*.d.ts',
    '!src/**/__tests__/**',
  ],
  coverageThreshold: {
    global: {
      branches: 80,
      functions: 80,
      lines: 80,
      statements: 80,
    },
  },
};
```

### Example Tests

**Service Layer:**
```typescript
// src/modules/auth/__tests__/auth.service.test.ts
import { AuthService } from '../auth.service';
import bcrypt from 'bcryptjs';
import jwt from 'jsonwebtoken';

describe('AuthService', () => {
  let authService: AuthService;

  beforeEach(() => {
    authService = new AuthService();
  });

  describe('hashPassword', () => {
    it('should hash password with bcrypt', async () => {
      const password = 'Password123!';
      const hash = await authService.hashPassword(password);

      expect(hash).toBeDefined();
      expect(hash).not.toBe(password);
      expect(hash).toMatch(/^\$2[aby]\$.{56}$/);
    });

    it('should generate different hashes for same password', async () => {
      const password = 'Password123!';
      const hash1 = await authService.hashPassword(password);
      const hash2 = await authService.hashPassword(password);

      expect(hash1).not.toBe(hash2);
    });
  });

  describe('verifyPassword', () => {
    it('should verify correct password', async () => {
      const password = 'Password123!';
      const hash = await bcrypt.hash(password, 12);

      const isValid = await authService.verifyPassword(password, hash);

      expect(isValid).toBe(true);
    });

    it('should reject incorrect password', async () => {
      const password = 'Password123!';
      const hash = await bcrypt.hash(password, 12);

      const isValid = await authService.verifyPassword('WrongPassword', hash);

      expect(isValid).toBe(false);
    });
  });

  describe('generateTokens', () => {
    it('should generate valid JWT tokens', () => {
      const payload = {
        userId: '123',
        email: 'test@example.com',
        role: 'user',
      };

      const tokens = authService.generateTokens(payload);

      expect(tokens.accessToken).toBeDefined();
      expect(tokens.refreshToken).toBeDefined();
      expect(tokens.expiresIn).toBe(3600);

      const decoded = jwt.verify(tokens.accessToken, process.env.JWT_SECRET);
      expect(decoded.userId).toBe('123');
    });
  });
});
```

**Repository Layer:**
```typescript
// src/modules/user/__tests__/user.repository.test.ts
import { UserRepository } from '../user.repository';
import { PrismaClient } from '@prisma/client';

describe('UserRepository', () => {
  let repository: UserRepository;
  let prisma: PrismaClient;

  beforeAll(async () => {
    prisma = new PrismaClient();
    repository = new UserRepository(prisma);
  });

  afterEach(async () => {
    await prisma.user.deleteMany();
  });

  afterAll(async () => {
    await prisma.$disconnect();
  });

  describe('create', () => {
    it('should create new user', async () => {
      const user = await repository.create({
        email: 'test@example.com',
        passwordHash: 'hashed_password',
        name: 'Test User',
      });

      expect(user.id).toBeDefined();
      expect(user.email).toBe('test@example.com');
      expect(user.role).toBe('user');
    });

    it('should throw error for duplicate email', async () => {
      await repository.create({
        email: 'duplicate@example.com',
        passwordHash: 'hash',
        name: 'User 1',
      });

      await expect(
        repository.create({
          email: 'duplicate@example.com',
          passwordHash: 'hash',
          name: 'User 2',
        })
      ).rejects.toThrow();
    });
  });
});
```

---

## Integration Testing

### Tools
- **Supertest:** HTTP assertions
- **Test Database:** Separate PostgreSQL instance

### Setup

```typescript
// tests/setup.ts
import { PrismaClient } from '@prisma/client';

export const prisma = new PrismaClient({
  datasources: {
    db: {
      url: process.env.DATABASE_URL_TEST,
    },
  },
});

export async function resetDatabase() {
  await prisma.$executeRaw`TRUNCATE TABLE users CASCADE`;
  await prisma.$executeRaw`TRUNCATE TABLE recipes CASCADE`;
  await prisma.$executeRaw`TRUNCATE TABLE meal_plans CASCADE`;
  // ... other tables
}

beforeEach(async () => {
  await resetDatabase();
});

afterAll(async () => {
  await prisma.$disconnect();
});
```

### Example Tests

**Authentication Endpoints:**
```typescript
// tests/integration/auth.test.ts
import request from 'supertest';
import app from '../../src/app';
import { resetDatabase } from '../setup';

describe('Authentication API', () => {
  beforeEach(async () => {
    await resetDatabase();
  });

  describe('POST /api/v1/auth/register', () => {
    it('should register new user', async () => {
      const res = await request(app)
        .post('/api/v1/auth/register')
        .send({
          email: 'newuser@example.com',
          password: 'SecurePassword123!',
          name: 'New User',
        });

      expect(res.status).toBe(201);
      expect(res.body.data.user.email).toBe('newuser@example.com');
      expect(res.body.data.tokens.accessToken).toBeDefined();
      expect(res.body.data.user.password).toBeUndefined(); // Never return password
    });

    it('should return 409 for duplicate email', async () => {
      await request(app)
        .post('/api/v1/auth/register')
        .send({
          email: 'existing@example.com',
          password: 'Password123!',
          name: 'User 1',
        });

      const res = await request(app)
        .post('/api/v1/auth/register')
        .send({
          email: 'existing@example.com',
          password: 'Password123!',
          name: 'User 2',
        });

      expect(res.status).toBe(409);
      expect(res.body.error.code).toBe('CONFLICT');
    });

    it('should validate password strength', async () => {
      const res = await request(app)
        .post('/api/v1/auth/register')
        .send({
          email: 'weak@example.com',
          password: 'weak',
          name: 'Weak User',
        });

      expect(res.status).toBe(400);
      expect(res.body.error.code).toBe('VALIDATION_ERROR');
    });
  });

  describe('POST /api/v1/auth/login', () => {
    let testUser;

    beforeEach(async () => {
      const res = await request(app)
        .post('/api/v1/auth/register')
        .send({
          email: 'test@example.com',
          password: 'Password123!',
          name: 'Test User',
        });

      testUser = res.body.data.user;
    });

    it('should login with valid credentials', async () => {
      const res = await request(app)
        .post('/api/v1/auth/login')
        .send({
          email: 'test@example.com',
          password: 'Password123!',
        });

      expect(res.status).toBe(200);
      expect(res.body.data.user.id).toBe(testUser.id);
      expect(res.body.data.tokens.accessToken).toBeDefined();
    });

    it('should return 401 for invalid password', async () => {
      const res = await request(app)
        .post('/api/v1/auth/login')
        .send({
          email: 'test@example.com',
          password: 'WrongPassword123!',
        });

      expect(res.status).toBe(401);
      expect(res.body.error.code).toBe('UNAUTHORIZED');
    });

    it('should rate limit after 5 failed attempts', async () => {
      for (let i = 0; i < 5; i++) {
        await request(app)
          .post('/api/v1/auth/login')
          .send({
            email: 'test@example.com',
            password: 'WrongPassword',
          });
      }

      const res = await request(app)
        .post('/api/v1/auth/login')
        .send({
          email: 'test@example.com',
          password: 'WrongPassword',
        });

      expect(res.status).toBe(429);
      expect(res.body.error.code).toBe('RATE_LIMIT_EXCEEDED');
    });
  });
});
```

**Recipe Endpoints:**
```typescript
// tests/integration/recipes.test.ts
describe('Recipe API', () => {
  let authToken;
  let adminToken;

  beforeEach(async () => {
    await resetDatabase();

    // Create regular user
    const userRes = await request(app)
      .post('/api/v1/auth/register')
      .send({
        email: 'user@example.com',
        password: 'Password123!',
        name: 'Regular User',
      });
    authToken = userRes.body.data.tokens.accessToken;

    // Create admin user (manually set role in DB)
    const adminRes = await request(app)
      .post('/api/v1/auth/register')
      .send({
        email: 'admin@example.com',
        password: 'Password123!',
        name: 'Admin User',
      });
    adminToken = adminRes.body.data.tokens.accessToken;

    await prisma.user.update({
      where: { email: 'admin@example.com' },
      data: { role: 'admin' },
    });
  });

  describe('GET /api/v1/recipes', () => {
    it('should return paginated recipes', async () => {
      // Create test recipes
      for (let i = 0; i < 25; i++) {
        await prisma.recipe.create({
          data: {
            name: `Recipe ${i + 1}`,
            category: 'Dinner',
            prepTime: 15,
            cookTime: 20,
            servings: 4,
            difficulty: 'Easy',
            ingredients: [],
            instructions: [],
            nutrition: {},
            tags: [],
          },
        });
      }

      const res = await request(app)
        .get('/api/v1/recipes?page=1&limit=20')
        .set('Authorization', `Bearer ${authToken}`);

      expect(res.status).toBe(200);
      expect(res.body.data).toHaveLength(20);
      expect(res.body.pagination.total).toBe(25);
      expect(res.body.pagination.totalPages).toBe(2);
    });

    it('should filter by category', async () => {
      await prisma.recipe.create({
        data: {
          name: 'Breakfast Recipe',
          category: 'Breakfast',
          prepTime: 10,
          cookTime: 5,
          servings: 2,
          difficulty: 'Easy',
          ingredients: [],
          instructions: [],
        },
      });

      const res = await request(app)
        .get('/api/v1/recipes?category=Breakfast')
        .set('Authorization', `Bearer ${authToken}`);

      expect(res.status).toBe(200);
      expect(res.body.data).toHaveLength(1);
      expect(res.body.data[0].category).toBe('Breakfast');
    });
  });

  describe('POST /api/v1/recipes', () => {
    it('should create recipe (admin only)', async () => {
      const res = await request(app)
        .post('/api/v1/recipes')
        .set('Authorization', `Bearer ${adminToken}`)
        .send({
          name: 'New Recipe',
          category: 'Dinner',
          prepTime: 15,
          cookTime: 20,
          servings: 4,
          difficulty: 'Easy',
          ingredients: [{ name: 'Chicken', quantity: '1', unit: 'lb' }],
          instructions: ['Cook chicken'],
          nutrition: { calories: 350 },
          tags: ['Quick'],
        });

      expect(res.status).toBe(201);
      expect(res.body.data.name).toBe('New Recipe');
    });

    it('should return 403 for non-admin', async () => {
      const res = await request(app)
        .post('/api/v1/recipes')
        .set('Authorization', `Bearer ${authToken}`)
        .send({ name: 'Test Recipe' });

      expect(res.status).toBe(403);
      expect(res.body.error.code).toBe('FORBIDDEN');
    });
  });
});
```

---

## End-to-End Testing

### Tools
- **Playwright** (optional, primarily frontend)
- **Postman Collections** (API E2E)

### Example Flow

```typescript
// tests/e2e/meal-planning-flow.test.ts
describe('Complete Meal Planning Flow', () => {
  it('should complete full user journey', async () => {
    // 1. Register
    const registerRes = await request(app)
      .post('/api/v1/auth/register')
      .send({
        email: 'journey@example.com',
        password: 'Password123!',
        name: 'Journey User',
      });

    const token = registerRes.body.data.tokens.accessToken;
    expect(registerRes.status).toBe(201);

    // 2. Browse recipes
    const recipesRes = await request(app)
      .get('/api/v1/recipes')
      .set('Authorization', `Bearer ${token}`);

    expect(recipesRes.status).toBe(200);
    const recipe = recipesRes.body.data[0];

    // 3. Favorite a recipe
    const favoriteRes = await request(app)
      .post(`/api/v1/recipes/${recipe.id}/favorite`)
      .set('Authorization', `Bearer ${token}`);

    expect(favoriteRes.status).toBe(200);

    // 4. Create meal plan
    const planRes = await request(app)
      .post('/api/v1/meal-plans')
      .set('Authorization', `Bearer ${token}`)
      .send({ weekStart: '2025-10-13' });

    expect(planRes.status).toBe(201);
    const planId = planRes.body.data.id;

    // 5. Add meal to plan
    const addMealRes = await request(app)
      .post(`/api/v1/meal-plans/${planId}/meals`)
      .set('Authorization', `Bearer ${token}`)
      .send({
        day: 'monday',
        mealType: 'dinner',
        recipeId: recipe.id,
      });

    expect(addMealRes.status).toBe(200);

    // 6. Generate shopping list
    const shoppingRes = await request(app)
      .post('/api/v1/shopping-lists/generate')
      .set('Authorization', `Bearer ${token}`)
      .send({ mealPlanId: planId });

    expect(shoppingRes.status).toBe(201);
    expect(shoppingRes.body.data.items.length).toBeGreaterThan(0);

    // 7. Check off item
    const listId = shoppingRes.body.data.id;
    const itemId = shoppingRes.body.data.items[0].id;

    const checkRes = await request(app)
      .patch(`/api/v1/shopping-lists/${listId}/items/${itemId}`)
      .set('Authorization', `Bearer ${token}`)
      .send({ checked: true });

    expect(checkRes.status).toBe(200);
    expect(checkRes.body.data.checkedCount).toBe(1);
  });
});
```

---

## Load Testing

### Tool: k6

**Installation:**
```bash
# macOS
brew install k6

# Docker
docker pull grafana/k6
```

**Test Script:**
```javascript
// tests/load/steady-state.js
import http from 'k6/http';
import { check, sleep } from 'k6';

export const options = {
  stages: [
    { duration: '2m', target: 100 },   // Ramp-up
    { duration: '5m', target: 1000 },  // Steady state
    { duration: '2m', target: 0 },     // Ramp-down
  ],
  thresholds: {
    http_req_duration: ['p(95)<200'],  // P95 < 200ms
    http_req_failed: ['rate<0.01'],    // Error rate < 1%
  },
};

export default function () {
  const BASE_URL = 'https://api.mealplanner.com/api/v1';

  // Login
  const loginRes = http.post(`${BASE_URL}/auth/login`, JSON.stringify({
    email: __ENV.TEST_EMAIL,
    password: __ENV.TEST_PASSWORD,
  }), {
    headers: { 'Content-Type': 'application/json' },
  });

  check(loginRes, {
    'login successful': (r) => r.status === 200,
  });

  const token = loginRes.json('data.tokens.accessToken');

  // Browse recipes
  const recipesRes = http.get(`${BASE_URL}/recipes`, {
    headers: { Authorization: `Bearer ${token}` },
  });

  check(recipesRes, {
    'recipes loaded': (r) => r.status === 200,
    'response time OK': (r) => r.timings.duration < 200,
  });

  sleep(1);
}
```

**Run Load Test:**
```bash
k6 run tests/load/steady-state.js

# With environment variables
k6 run -e TEST_EMAIL=test@example.com -e TEST_PASSWORD=Password123! tests/load/steady-state.js
```

---

## Performance Testing

### Database Query Performance

```typescript
// tests/performance/query-performance.test.ts
describe('Query Performance', () => {
  it('should execute recipe search in < 300ms', async () => {
    // Insert 1000 test recipes
    for (let i = 0; i < 1000; i++) {
      await prisma.recipe.create({
        data: {
          name: `Recipe ${i}`,
          category: 'Dinner',
          prepTime: 15,
          cookTime: 20,
          servings: 4,
          difficulty: 'Easy',
          ingredients: [],
          instructions: [],
        },
      });
    }

    const start = Date.now();

    await request(app)
      .get('/api/v1/recipes/search?q=chicken')
      .set('Authorization', `Bearer ${token}`);

    const duration = Date.now() - start;

    expect(duration).toBeLessThan(300);
  });
});
```

---

## Security Testing

### OWASP ZAP Automated Scan

```bash
docker run -t owasp/zap2docker-stable zap-baseline.py \
  -t https://api-staging.mealplanner.com \
  -r zap-report.html
```

---

## Test Data Management

### Test Fixtures

```typescript
// tests/fixtures/users.ts
export const testUsers = {
  regularUser: {
    email: 'regular@example.com',
    password: 'Password123!',
    name: 'Regular User',
    role: 'user',
  },
  adminUser: {
    email: 'admin@example.com',
    password: 'AdminPassword123!',
    name: 'Admin User',
    role: 'admin',
  },
};

// tests/fixtures/recipes.ts
export const testRecipes = {
  chickenTacos: {
    name: 'Chicken Tacos',
    category: 'Dinner',
    prepTime: 15,
    cookTime: 20,
    servings: 4,
    difficulty: 'Easy',
    ingredients: [
      { name: 'Chicken breast', quantity: '1', unit: 'lb' },
      { name: 'Tortillas', quantity: '8', unit: '' },
    ],
    instructions: ['Cook chicken', 'Assemble tacos'],
    nutrition: { calories: 350, protein: 28 },
    tags: ['Mexican', 'Quick'],
  },
};
```

---

## CI/CD Integration

### GitHub Actions Test Workflow

```yaml
name: Tests

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest

    services:
      postgres:
        image: postgres:15
        env:
          POSTGRES_PASSWORD: test_password
        options: >-
          --health-cmd pg_isready
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5
        ports:
          - 5432:5432

      redis:
        image: redis:7
        ports:
          - 6379:6379

    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-node@v3
        with:
          node-version: '20'

      - name: Install dependencies
        run: npm ci

      - name: Run migrations
        run: npx prisma migrate deploy

      - name: Run tests
        run: npm test
        env:
          DATABASE_URL: postgresql://postgres:test_password@localhost:5432/test
          REDIS_URL: redis://localhost:6379

      - name: Upload coverage
        uses: codecov/codecov-action@v3
        with:
          file: ./coverage/coverage-final.json
```

---

## Testing Checklist

### Before Each PR

- [ ] All tests passing locally
- [ ] Coverage > 80% for new code
- [ ] Integration tests for new endpoints
- [ ] No console.log() statements
- [ ] Test names are descriptive

### Before Deployment

- [ ] Full test suite passing (unit + integration)
- [ ] Load test completed successfully
- [ ] Security scan passed (npm audit)
- [ ] Smoke tests passing in staging
- [ ] Database migrations tested

---

**Document Version:** 1.0
**Last Updated:** 2025-10-14

This testing strategy ensures high-quality, reliable backend code through comprehensive test coverage.
