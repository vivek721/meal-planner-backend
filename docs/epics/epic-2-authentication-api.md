# Epic 2: Authentication API
## User Authentication & Authorization System

---

**Epic ID:** EPIC-2
**Priority:** P0 (Critical)
**Estimated Effort:** 40 hours
**Sprint:** Week 2
**Owner:** Senior Backend Developer
**Status:** Not Started
**Dependencies:** Epic 1 (Infrastructure Setup)

---

## Overview

Implement a secure, production-ready authentication system with JWT-based session management, password security, and role-based access control. This epic replaces the frontend's localStorage mock authentication with a real backend service.

## Goals

1. Implement user registration with secure password hashing
2. Build JWT-based login/logout system
3. Create token refresh mechanism
4. Implement password reset flow
5. Add role-based authorization middleware
6. Ensure frontend API contract compliance

## User Stories

### US-2.1: As a new user, I can register an account

**Acceptance Criteria:**
- POST `/api/v1/auth/register` endpoint implemented
- Email validation (format, uniqueness)
- Password strength validation (min 8 chars, 1 uppercase, 1 number, 1 special)
- Password hashed with bcrypt (12 rounds minimum)
- User record created in database
- JWT access and refresh tokens returned
- Welcome email sent (async job)

**Request Example:**
```json
{
  "email": "sarah@example.com",
  "password": "SecurePassword123!",
  "name": "Sarah Johnson"
}
```

**Response:**
```json
{
  "data": {
    "user": {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "email": "sarah@example.com",
      "name": "Sarah Johnson",
      "role": "user"
    },
    "tokens": {
      "accessToken": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
      "refreshToken": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
      "expiresIn": 3600
    }
  }
}
```

---

### US-2.2: As a registered user, I can login to my account

**Acceptance Criteria:**
- POST `/api/v1/auth/login` endpoint implemented
- Email and password validated
- Password compared using bcrypt.compare()
- Rate limiting: max 5 attempts per 15 minutes per IP
- JWT tokens generated and returned
- User's `last_login_at` timestamp updated
- Login activity logged

**Business Logic:**
1. Find user by email
2. Check if account is soft-deleted (return 401 if deleted)
3. Compare password hash
4. Generate access token (1 hour expiry)
5. Generate refresh token (7 days expiry)
6. Store refresh token hash in database
7. Update last_login_at
8. Return tokens

**Error Handling:**
- 401 for invalid credentials
- 429 after 5 failed attempts
- Generic "Invalid email or password" message (no email enumeration)

---

### US-2.3: As a logged-in user, I can refresh my access token

**Acceptance Criteria:**
- POST `/api/v1/auth/refresh` endpoint implemented
- Refresh token validated (signature, expiration, not blacklisted)
- New access token generated
- Old refresh token optionally rotated (security best practice)
- Refresh token reuse detected and prevented

**Token Rotation Logic:**
1. Verify refresh token signature
2. Check if token is in blacklist (Redis)
3. Generate new access token
4. Optionally generate new refresh token
5. Blacklist old refresh token
6. Return new tokens

---

### US-2.4: As a logged-in user, I can logout

**Acceptance Criteria:**
- POST `/api/v1/auth/logout` endpoint implemented
- Access token added to blacklist (Redis)
- Refresh token invalidated in database
- Session cleared
- 204 No Content response

**Blacklist Implementation:**
- Store token hash in Redis
- TTL = remaining time until token expiry
- Check blacklist in auth middleware

---

### US-2.5: As a user, I can reset my password if forgotten

**Acceptance Criteria:**
- POST `/api/v1/auth/forgot-password` endpoint
- Email sent with reset link (6-digit code or UUID token)
- Reset token stored in database with 1-hour expiry
- POST `/api/v1/auth/reset-password` endpoint
- Token validated and password updated
- All active sessions invalidated after password reset

**Password Reset Flow:**
1. User requests reset (email)
2. Generate reset token (UUID)
3. Store token in `users.password_reset_token` with expiry
4. Send email with reset link: `https://app.mealplanner.com/reset-password?token=...`
5. User clicks link, enters new password
6. Validate token (exists, not expired)
7. Hash new password
8. Update user password
9. Clear reset token
10. Invalidate all refresh tokens

---

### US-2.6: As a logged-in user, I can view my profile

**Acceptance Criteria:**
- GET `/api/v1/auth/me` endpoint implemented
- Returns current user data from JWT payload
- Includes user ID, email, name, role, preferences
- Requires valid JWT token in Authorization header

**Response:**
```json
{
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "email": "sarah@example.com",
    "name": "Sarah Johnson",
    "role": "user",
    "preferences": {
      "dietary": ["Vegan"],
      "allergies": [],
      "householdSize": 2
    },
    "createdAt": "2025-09-15T10:00:00Z",
    "lastLoginAt": "2025-10-14T10:30:00Z"
  }
}
```

---

### US-2.7: As a system, I enforce role-based access control

**Acceptance Criteria:**
- `authenticate` middleware validates JWT on protected routes
- `authorize(role)` middleware checks user role
- Admin endpoints require `role: admin`
- User endpoints require `role: user` or higher
- 403 Forbidden returned for insufficient permissions

**Middleware Implementation:**
```typescript
// authenticate.middleware.ts
export const authenticate = async (req, res, next) => {
  const token = req.headers.authorization?.replace('Bearer ', '');

  if (!token) {
    return res.status(401).json({ error: { code: 'UNAUTHORIZED' } });
  }

  // Check if token is blacklisted
  const isBlacklisted = await redis.get(`blacklist:${hashToken(token)}`);
  if (isBlacklisted) {
    return res.status(401).json({ error: { code: 'UNAUTHORIZED' } });
  }

  try {
    const payload = jwt.verify(token, process.env.JWT_SECRET);
    req.user = payload;
    next();
  } catch (err) {
    return res.status(401).json({ error: { code: 'UNAUTHORIZED' } });
  }
};

// authorize.middleware.ts
export const authorize = (...roles: string[]) => {
  return (req, res, next) => {
    if (!req.user || !roles.includes(req.user.role)) {
      return res.status(403).json({ error: { code: 'FORBIDDEN' } });
    }
    next();
  };
};
```

---

### US-2.8: As a developer, I have comprehensive auth tests

**Acceptance Criteria:**
- Unit tests for password hashing, JWT generation
- Integration tests for all auth endpoints
- Test coverage > 90% for auth module
- Tests include: happy path, validation errors, edge cases
- Tests run in CI/CD pipeline

**Test Cases:**
- Registration: success, duplicate email, weak password
- Login: success, invalid credentials, rate limiting
- Refresh: success, expired token, blacklisted token
- Logout: success, already logged out
- Password reset: request, reset, expired token
- Authorization: valid token, invalid token, wrong role

---

## Technical Requirements

### Database Tables

**users** (extends existing schema):
```sql
ALTER TABLE users
ADD COLUMN password_reset_token VARCHAR(255),
ADD COLUMN password_reset_expires TIMESTAMPTZ,
ADD COLUMN email_verification_token VARCHAR(255),
ADD COLUMN email_verified BOOLEAN DEFAULT FALSE;

CREATE INDEX idx_users_password_reset_token
ON users(password_reset_token)
WHERE password_reset_token IS NOT NULL;
```

**sessions** (token blacklist):
```sql
CREATE TABLE sessions (
  token_hash VARCHAR(255) PRIMARY KEY,
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  expires_at TIMESTAMPTZ NOT NULL,
  invalidated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_sessions_user_id ON sessions(user_id);
CREATE INDEX idx_sessions_expires_at ON sessions(expires_at);
```

### Dependencies

**NPM Packages:**
```json
{
  "bcryptjs": "^2.4.3",
  "jsonwebtoken": "^9.0.2",
  "zod": "^3.22.4",
  "express-rate-limit": "^7.1.5",
  "express-validator": "^7.0.1"
}
```

### Environment Variables

```bash
JWT_SECRET=<random-256-bit-secret>
JWT_ACCESS_EXPIRY=1h
JWT_REFRESH_EXPIRY=7d
BCRYPT_ROUNDS=12
RATE_LIMIT_MAX=5
RATE_LIMIT_WINDOW_MS=900000  # 15 minutes
```

---

## API Endpoints

### Endpoint Summary

| Method | Endpoint | Auth | Description |
|--------|----------|------|-------------|
| POST | `/auth/register` | None | Register new user |
| POST | `/auth/login` | None | Login user |
| POST | `/auth/refresh` | None | Refresh access token |
| POST | `/auth/logout` | Required | Logout user |
| POST | `/auth/forgot-password` | None | Request password reset |
| POST | `/auth/reset-password` | None | Reset password |
| GET | `/auth/me` | Required | Get current user |

---

## Security Considerations

### Password Security

1. **Hashing Algorithm:** bcrypt with 12 rounds minimum
2. **Validation:**
   - Min 8 characters
   - At least 1 uppercase letter
   - At least 1 number
   - At least 1 special character
3. **No Storage:** Never log or store plain-text passwords
4. **Breach Detection:** Optional integration with Have I Been Pwned API

### JWT Security

1. **Algorithm:** HS256 (HMAC with SHA-256)
2. **Secret:** 256-bit random secret, stored in AWS Secrets Manager
3. **Expiry:** Access token 1 hour, refresh token 7 days
4. **Rotation:** Refresh tokens rotated on use (security best practice)
5. **Blacklist:** Expired tokens stored in Redis until natural expiry

### Rate Limiting

```typescript
import rateLimit from 'express-rate-limit';

const loginLimiter = rateLimit({
  windowMs: 15 * 60 * 1000, // 15 minutes
  max: 5, // 5 requests per window
  message: { error: { code: 'RATE_LIMIT_EXCEEDED' } },
  standardHeaders: true,
  legacyHeaders: false,
});

router.post('/auth/login', loginLimiter, authController.login);
```

### Input Validation

```typescript
import { z } from 'zod';

const registerSchema = z.object({
  email: z.string().email(),
  password: z.string()
    .min(8)
    .regex(/[A-Z]/, 'Must contain uppercase')
    .regex(/[0-9]/, 'Must contain number')
    .regex(/[^A-Za-z0-9]/, 'Must contain special character'),
  name: z.string().min(2).max(255),
});
```

---

## Testing Requirements

### Unit Tests

```typescript
describe('AuthService', () => {
  describe('hashPassword', () => {
    it('should hash password with bcrypt', async () => {
      const hash = await authService.hashPassword('Password123!');
      expect(hash).toMatch(/^\$2[aby]\$.{56}$/);
    });
  });

  describe('generateTokens', () => {
    it('should generate valid JWT tokens', () => {
      const tokens = authService.generateTokens({ userId: '123', email: 'test@example.com', role: 'user' });
      expect(tokens.accessToken).toBeDefined();
      expect(tokens.refreshToken).toBeDefined();

      const payload = jwt.verify(tokens.accessToken, process.env.JWT_SECRET);
      expect(payload.userId).toBe('123');
    });
  });
});
```

### Integration Tests

```typescript
describe('POST /auth/register', () => {
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
  });

  it('should return 409 for duplicate email', async () => {
    // Register first user
    await authService.register({
      email: 'existing@example.com',
      password: 'Password123!',
      name: 'Existing User',
    });

    // Try to register with same email
    const res = await request(app)
      .post('/api/v1/auth/register')
      .send({
        email: 'existing@example.com',
        password: 'Password123!',
        name: 'Duplicate User',
      });

    expect(res.status).toBe(409);
    expect(res.body.error.code).toBe('CONFLICT');
  });
});

describe('POST /auth/login', () => {
  beforeEach(async () => {
    await authService.register({
      email: 'test@example.com',
      password: 'Password123!',
      name: 'Test User',
    });
  });

  it('should login with valid credentials', async () => {
    const res = await request(app)
      .post('/api/v1/auth/login')
      .send({
        email: 'test@example.com',
        password: 'Password123!',
      });

    expect(res.status).toBe(200);
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
    // Make 5 failed login attempts
    for (let i = 0; i < 5; i++) {
      await request(app)
        .post('/api/v1/auth/login')
        .send({
          email: 'test@example.com',
          password: 'WrongPassword',
        });
    }

    // 6th attempt should be rate limited
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
```

---

## Acceptance Criteria

### Definition of Done

- [ ] All 6 auth endpoints implemented and tested
- [ ] Password hashing with bcrypt (12 rounds)
- [ ] JWT tokens generated with correct expiry
- [ ] Token refresh mechanism working
- [ ] Token blacklist implemented in Redis
- [ ] Rate limiting active on login endpoint
- [ ] Input validation on all endpoints
- [ ] Role-based authorization middleware
- [ ] Test coverage > 90%
- [ ] API documentation updated
- [ ] Frontend integration tested (mock service replaced)

### Testing Checklist

- [ ] Register: success, duplicate email, weak password
- [ ] Login: success, invalid credentials, rate limiting
- [ ] Refresh: success, expired token, reused token
- [ ] Logout: success, token blacklisted
- [ ] Password reset: request, reset, expired token
- [ ] Authorization: valid JWT, invalid JWT, missing JWT
- [ ] Role-based access: user role, admin role, forbidden

---

## Dependencies

### Upstream Dependencies (Blockers)

- Epic 1: Infrastructure Setup (database, Redis, Docker)

### Downstream Dependencies (Unblocks)

- Epic 3: Recipe Service (requires authentication)
- Epic 4: Meal Planning API (requires authentication)
- All user-specific features

---

## Risks & Mitigation

### Risk 1: JWT Secret Compromise

**Impact:** Critical | **Probability:** Low

**Mitigation:**
- Store secret in AWS Secrets Manager (never in code)
- Rotate secret quarterly
- Use strong 256-bit random secret
- Monitor for unusual token activity

---

### Risk 2: Password Reset Token Abuse

**Impact:** High | **Probability:** Medium

**Mitigation:**
- 1-hour expiry on reset tokens
- Single-use tokens (invalidated after use)
- Rate limit password reset requests (1 per 15 minutes per email)
- Log all password reset attempts

---

### Risk 3: Token Blacklist Growth (Redis Memory)

**Impact:** Medium | **Probability:** Medium

**Mitigation:**
- TTL on blacklisted tokens (max 7 days)
- Automatic cleanup of expired tokens
- Monitor Redis memory usage
- Scale Redis if needed (cache.t3.small → cache.t3.medium)

---

## Deliverables

### Code

- `src/modules/auth/auth.controller.ts`: HTTP request handlers
- `src/modules/auth/auth.service.ts`: Business logic
- `src/modules/auth/auth.repository.ts`: Database queries
- `src/modules/auth/auth.validator.ts`: Input validation schemas
- `src/modules/auth/auth.routes.ts`: Route definitions
- `src/common/middleware/authenticate.ts`: JWT verification
- `src/common/middleware/authorize.ts`: Role checking
- `src/common/utils/jwt.ts`: Token generation/verification
- `src/common/utils/crypto.ts`: Password hashing

### Tests

- `src/modules/auth/__tests__/auth.service.test.ts`
- `src/modules/auth/__tests__/auth.controller.test.ts`
- `src/modules/auth/__tests__/auth.integration.test.ts`

### Documentation

- `docs/api/authentication.md`: Endpoint documentation
- `docs/security/JWT_IMPLEMENTATION.md`: Token security guide
- `docs/runbooks/PASSWORD_RESET.md`: Manual intervention guide

---

## Timeline

### Week 2 Breakdown

| Day | Tasks | Hours |
|-----|-------|-------|
| **Monday** | Register, login endpoints, password hashing | 8 |
| **Tuesday** | JWT generation, token refresh, logout | 8 |
| **Wednesday** | Password reset flow, email integration | 8 |
| **Thursday** | Authorization middleware, role-based access | 8 |
| **Friday** | Testing, bug fixes, documentation | 8 |

**Total:** 40 hours

---

## Success Metrics

- Registration success rate > 95%
- Login response time P95 < 200ms
- Password reset email delivery > 99%
- Zero authentication bypasses (security testing)
- Test coverage > 90%

---

## Frontend Integration

### Migration from Mock to Real API

**Before (Mock):**
```typescript
// MockAuthService (localStorage)
login(email, password) {
  const user = users.find(u => u.email === email);
  if (user && user.password === password) {
    localStorage.setItem('token', 'mock-jwt-token');
    return { user, token: 'mock-jwt-token' };
  }
  throw new Error('Invalid credentials');
}
```

**After (Real API):**
```typescript
// ApiAuthService (HTTP)
async login(email, password) {
  const response = await fetch('https://api.mealplanner.com/api/v1/auth/login', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ email, password }),
  });

  if (!response.ok) {
    throw new Error('Invalid credentials');
  }

  const data = await response.json();
  localStorage.setItem('accessToken', data.tokens.accessToken);
  localStorage.setItem('refreshToken', data.tokens.refreshToken);

  return { user: data.user, token: data.tokens.accessToken };
}
```

**No Frontend Code Changes Required!** Service factory automatically swaps implementations based on `VITE_USE_MOCK_SERVICES` env variable.

---

**Epic Status:** Not Started
**Last Updated:** 2025-10-14
**Next Review:** End of Week 2

This epic establishes secure authentication, enabling all user-specific features in subsequent epics.
