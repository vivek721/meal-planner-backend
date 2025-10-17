# Epic 8: Testing & Production Launch
## Final Testing, Optimization & Deployment

---

**Epic ID:** EPIC-8
**Priority:** P0 (Critical)
**Estimated Effort:** 40 hours
**Sprint:** Week 8
**Owner:** Senior Backend Developer + DevOps Engineer
**Status:** Not Started
**Dependencies:** All previous epics (1-7)

---

## Overview

Conduct comprehensive testing, performance optimization, security audit, and production deployment. Ensure backend is production-ready with 99.9% uptime target.

## Goals

1. Achieve 80%+ test coverage across all modules
2. Conduct load testing (10K+ concurrent users)
3. Perform security audit (OWASP Top 10)
4. Optimize performance (API P95 < 200ms)
5. Deploy to production
6. Monitor and validate production deployment

## User Stories

### US-8.1: Achieve comprehensive test coverage

**Acceptance Criteria:**
- Unit test coverage > 85%
- Integration test coverage > 80%
- All critical paths tested
- Edge cases covered
- Error scenarios tested

**Test Categories:**
- **Unit Tests:** Service layer business logic
- **Integration Tests:** API endpoints with database
- **E2E Tests:** Full user flows (register → plan meal → generate list)

**Coverage Report:**
```bash
npm run test:coverage

----------------------|---------|----------|---------|---------|
File                  | % Stmts | % Branch | % Funcs | % Lines |
----------------------|---------|----------|---------|---------|
All files            |   87.45 |    82.31 |   89.12 |   87.45 |
 auth/               |   92.15 |    88.50 |   95.00 |   92.15 |
  auth.service.ts    |   95.30 |    90.00 |   100   |   95.30 |
  auth.controller.ts |   88.50 |    85.00 |   90.00 |   88.50 |
 recipe/             |   85.20 |    80.15 |   87.50 |   85.20 |
  recipe.service.ts  |   90.00 |    85.00 |   92.00 |   90.00 |
  recipe.search.ts   |   78.50 |    72.30 |   80.00 |   78.50 |
...
```

---

### US-8.2: Conduct load testing

**Tool:** k6 (open-source load testing)

**Scenarios:**
1. **Steady State:** 1,000 concurrent users for 10 minutes
2. **Spike Test:** Ramp to 5,000 users in 1 minute
3. **Stress Test:** Gradually increase to 10,000 users until failure

**k6 Script:**
```javascript
import http from 'k6/http';
import { check, sleep } from 'k6';

export const options = {
  stages: [
    { duration: '2m', target: 100 },    // Warm up
    { duration: '5m', target: 1000 },   // Steady state
    { duration: '2m', target: 5000 },   // Spike
    { duration: '5m', target: 5000 },   // Sustained spike
    { duration: '2m', target: 0 },      // Cool down
  ],
  thresholds: {
    http_req_duration: ['p(95)<200'],   // 95% of requests < 200ms
    http_req_failed: ['rate<0.01'],     // Error rate < 1%
  },
};

export default function () {
  // Login
  const loginRes = http.post('https://api.mealplanner.com/api/v1/auth/login', JSON.stringify({
    email: 'test@example.com',
    password: 'Password123!',
  }), {
    headers: { 'Content-Type': 'application/json' },
  });

  check(loginRes, {
    'login successful': (r) => r.status === 200,
  });

  const token = loginRes.json('data.tokens.accessToken');

  // Browse recipes
  const recipesRes = http.get('https://api.mealplanner.com/api/v1/recipes?page=1&limit=20', {
    headers: { Authorization: `Bearer ${token}` },
  });

  check(recipesRes, {
    'recipes loaded': (r) => r.status === 200,
    'response time OK': (r) => r.timings.duration < 200,
  });

  sleep(1);
}
```

**Success Criteria:**
- P95 response time < 200ms
- Error rate < 0.1%
- Throughput > 1,000 req/sec
- Zero database connection exhaustion

---

### US-8.3: Perform security audit

**OWASP Top 10 Checklist:**

1. **Injection (SQL, NoSQL)**
   - [ ] All queries use parameterized statements
   - [ ] No string concatenation in SQL
   - [ ] Input validation on all endpoints

2. **Broken Authentication**
   - [ ] JWT tokens properly secured
   - [ ] Password hashing with bcrypt (12+ rounds)
   - [ ] Token blacklist on logout
   - [ ] Session timeout enforced

3. **Sensitive Data Exposure**
   - [ ] HTTPS only (TLS 1.3)
   - [ ] Passwords never logged
   - [ ] Database encrypted at rest
   - [ ] Secrets in AWS Secrets Manager

4. **XML External Entities (XXE)**
   - [ ] No XML parsing (JSON-only API)

5. **Broken Access Control**
   - [ ] Authorization checks on all endpoints
   - [ ] User can only access own data
   - [ ] Admin role required for admin endpoints

6. **Security Misconfiguration**
   - [ ] Security headers (Helmet.js)
   - [ ] CORS properly configured
   - [ ] Default credentials changed
   - [ ] Error messages don't leak info

7. **Cross-Site Scripting (XSS)**
   - [ ] Input sanitization
   - [ ] Output encoding
   - [ ] Content Security Policy headers

8. **Insecure Deserialization**
   - [ ] Validate all JSON inputs
   - [ ] Use Zod schemas

9. **Using Components with Known Vulnerabilities**
   - [ ] `npm audit` shows zero high/critical
   - [ ] Dependencies up to date
   - [ ] Automated vulnerability scanning (Snyk)

10. **Insufficient Logging & Monitoring**
    - [ ] All errors logged
    - [ ] Authentication failures logged
    - [ ] CloudWatch alarms configured
    - [ ] Sentry error tracking active

**Security Testing Tools:**
- **OWASP ZAP:** Automated vulnerability scanning
- **npm audit:** Dependency vulnerabilities
- **Snyk:** Continuous security monitoring
- **SSL Labs:** HTTPS configuration testing

---

### US-8.4: Optimize performance

**Database Optimization:**
```sql
-- Analyze slow queries
EXPLAIN ANALYZE
SELECT r.* FROM recipes r
WHERE r.search_vector @@ to_tsquery('chicken')
ORDER BY r.rating DESC
LIMIT 20;

-- Add missing indexes
CREATE INDEX IF NOT EXISTS idx_recipes_rating_desc ON recipes(rating DESC);

-- Update statistics
ANALYZE recipes;

-- Vacuum to reclaim space
VACUUM ANALYZE recipes;
```

**Redis Caching Audit:**
```typescript
// Check cache hit rates
const stats = await redis.info('stats');
console.log('Cache hit rate:', stats.keyspace_hits / (stats.keyspace_hits + stats.keyspace_misses));

// Target: > 80% hit rate
```

**Query Optimization:**
```typescript
// Before: N+1 query problem
const mealPlan = await db.mealPlans.findById(id);
for (const recipeId of extractRecipeIds(mealPlan.meals)) {
  const recipe = await db.recipes.findById(recipeId); // N queries!
}

// After: Single query with IN clause
const recipeIds = extractRecipeIds(mealPlan.meals);
const recipes = await db.recipes.findMany({
  where: { id: { in: recipeIds } },
});
```

---

### US-8.5: Deploy to production

**Pre-Deployment Checklist:**
- [ ] All tests passing (unit, integration, e2e)
- [ ] Load testing completed successfully
- [ ] Security audit passed (zero high/critical)
- [ ] Database migrations tested in staging
- [ ] Environment variables configured
- [ ] Secrets stored in AWS Secrets Manager
- [ ] Monitoring & alerting active
- [ ] Backup & rollback procedures documented
- [ ] Stakeholder approval received

**Deployment Steps:**
1. **Create Production Database Snapshot**
   ```bash
   aws rds create-db-snapshot \
     --db-instance-identifier meal-planner-prod \
     --db-snapshot-identifier pre-launch-2025-10-14
   ```

2. **Run Database Migrations**
   ```bash
   npm run migrate:production
   ```

3. **Deploy Docker Image to ECS**
   ```bash
   # Build and tag
   docker build -t meal-planner-api:1.0.0 .
   docker tag meal-planner-api:1.0.0 $ECR_REGISTRY/meal-planner-api:1.0.0

   # Push to ECR
   aws ecr get-login-password | docker login --username AWS --password-stdin $ECR_REGISTRY
   docker push $ECR_REGISTRY/meal-planner-api:1.0.0

   # Update ECS service
   aws ecs update-service \
     --cluster meal-planner-cluster \
     --service meal-planner-api \
     --force-new-deployment
   ```

4. **Wait for Deployment to Stabilize**
   ```bash
   aws ecs wait services-stable \
     --cluster meal-planner-cluster \
     --services meal-planner-api
   ```

5. **Run Smoke Tests**
   ```bash
   curl -f https://api.mealplanner.com/health || exit 1
   curl -H "Authorization: Bearer $TEST_TOKEN" https://api.mealplanner.com/api/v1/recipes | jq '.data | length'
   ```

6. **Monitor for 1 Hour**
   - Watch CloudWatch metrics (error rate, latency)
   - Monitor Sentry for errors
   - Check database connections
   - Verify cache hit rates

---

### US-8.6: Post-deployment validation

**Validation Checklist:**
- [ ] Health check endpoint returns 200
- [ ] All API endpoints accessible
- [ ] Frontend integration working
- [ ] User registration successful
- [ ] Login flow working
- [ ] Recipe browsing functional
- [ ] Meal planning operational
- [ ] Shopping list generation working
- [ ] AI recommendations returning results
- [ ] Emails sending successfully
- [ ] Error rate < 0.1%
- [ ] P95 response time < 200ms
- [ ] Database queries < 50ms
- [ ] Cache hit rate > 80%

**Production Smoke Test Script:**
```bash
#!/bin/bash

API_URL="https://api.mealplanner.com/api/v1"

# 1. Health check
echo "Testing health endpoint..."
curl -f $API_URL/health || { echo "Health check failed"; exit 1; }

# 2. Register test user
echo "Testing registration..."
REGISTER_RESPONSE=$(curl -s -X POST $API_URL/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email":"smoketest@example.com","password":"Test123!","name":"Smoke Test"}')

TOKEN=$(echo $REGISTER_RESPONSE | jq -r '.data.tokens.accessToken')

# 3. Browse recipes
echo "Testing recipe browsing..."
curl -f -H "Authorization: Bearer $TOKEN" $API_URL/recipes || { echo "Recipe browse failed"; exit 1; }

# 4. Create meal plan
echo "Testing meal plan creation..."
curl -f -X POST -H "Authorization: Bearer $TOKEN" $API_URL/meal-plans \
  -H "Content-Type: application/json" \
  -d '{"weekStart":"2025-10-13"}' || { echo "Meal plan creation failed"; exit 1; }

echo "All smoke tests passed!"
```

---

## Deliverables

### Test Suites

- `tests/unit/` - 200+ unit tests
- `tests/integration/` - 100+ integration tests
- `tests/e2e/` - 20+ end-to-end tests
- `tests/load/` - k6 load test scripts

### Documentation

- `docs/testing/TEST_COVERAGE_REPORT.md`
- `docs/testing/LOAD_TEST_RESULTS.md`
- `docs/security/SECURITY_AUDIT_REPORT.md`
- `docs/deployment/PRODUCTION_LAUNCH_CHECKLIST.md`
- `docs/runbooks/ROLLBACK_PROCEDURE.md`
- `docs/runbooks/INCIDENT_RESPONSE.md`

### Scripts

- `scripts/smoke-test.sh` - Production validation
- `scripts/load-test.sh` - k6 load testing
- `scripts/security-scan.sh` - OWASP ZAP scan
- `scripts/performance-benchmark.sh` - API benchmarking

---

## Acceptance Criteria

### Testing

- [ ] Unit test coverage > 85%
- [ ] Integration test coverage > 80%
- [ ] All critical paths tested
- [ ] E2E tests passing

### Performance

- [ ] API P95 response time < 200ms
- [ ] Database query P95 < 50ms
- [ ] Load test supports 10K concurrent users
- [ ] Cache hit rate > 80%
- [ ] Throughput > 1,000 req/sec

### Security

- [ ] Zero high/critical vulnerabilities (npm audit)
- [ ] OWASP Top 10 compliance
- [ ] SSL Labs A+ rating
- [ ] All secrets in AWS Secrets Manager
- [ ] Security headers configured (Helmet.js)

### Deployment

- [ ] Successful production deployment
- [ ] Zero errors in first hour post-launch
- [ ] Health checks passing
- [ ] Monitoring & alerting operational
- [ ] Backup & rollback tested

---

## Timeline

| Day | Tasks | Hours |
|-----|-------|-------|
| **Monday** | Write missing tests, achieve 80%+ coverage | 8 |
| **Tuesday** | Load testing with k6, performance optimization | 8 |
| **Wednesday** | Security audit, vulnerability fixes | 8 |
| **Thursday** | Production deployment prep, final testing | 8 |
| **Friday** | Production deployment, monitoring, validation | 8 |

**Total:** 40 hours

---

## Success Metrics

- Production deployment successful: ✅
- Zero critical errors in first 24 hours: ✅
- API uptime > 99.9% in first week: ✅
- User registrations working: ✅
- Frontend integration seamless: ✅

---

## Rollback Plan

If critical issues detected post-deployment:

1. **Immediate Actions** (< 5 minutes)
   - Revert ECS service to previous task definition
   - Monitor error rates
   - Notify team via Slack/PagerDuty

2. **Database Rollback** (if schema changed)
   - Restore from pre-deployment snapshot
   - Replay transaction logs if needed

3. **Communication**
   - Update status page
   - Notify users via email (if widespread impact)
   - Document incident for post-mortem

4. **Post-Incident Review**
   - Root cause analysis
   - Update deployment procedures
   - Add monitoring/tests to prevent recurrence

---

**Epic Status:** Not Started
**Last Updated:** 2025-10-14
**Next Review:** End of Week 8

This epic marks the culmination of backend development and transition to production operations.
