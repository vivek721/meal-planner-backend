# Technology Stack
## AI-Powered Meal Planner Backend

---

**Version:** 1.0
**Last Updated:** 2025-10-14

---

## Overview

This document provides detailed justification for all technology choices in the backend stack, including alternatives considered and decision rationale.

---

## Core Technologies

### 1. Runtime: Node.js 20 LTS

**Purpose:** JavaScript runtime for backend services

**Why Node.js:**
- **JavaScript Consistency:** Same language as React frontend, easier to find full-stack developers
- **Async I/O:** Non-blocking, perfect for I/O-heavy operations (database, API calls)
- **Mature Ecosystem:** 2M+ npm packages, battle-tested libraries
- **Performance:** V8 engine, comparable to Go/Java for web services
- **Long-Term Support:** LTS version ensures stability for 30+ months

**Alternatives Considered:**
- **Python + FastAPI:** Better for ML (but we use AWS Personalize), slower for I/O
- **Go:** Faster, but smaller ecosystem, steeper learning curve
- **Java + Spring Boot:** Enterprise-grade, but heavier, slower development

**Configuration:**
```json
{
  "engines": {
    "node": ">=20.0.0",
    "npm": ">=10.0.0"
  }
}
```

---

### 2. Framework: Express.js 4.x

**Purpose:** Web application framework

**Why Express:**
- **Battle-Tested:** 15+ years in production, powers Netflix, Uber
- **Minimalist:** Unopinionated, flexible, easy to customize
- **Middleware Ecosystem:** Extensive plugins for auth, validation, logging
- **Performance:** Handles 10K+ req/sec on modest hardware
- **Community:** Largest Node.js framework community

**Alternatives Considered:**
- **Fastify:** Faster (2x Express), but smaller ecosystem, less mature
- **NestJS:** TypeScript-first, more opinionated, heavier framework
- **Koa:** Lighter than Express, but smaller community

**Decision:** Express for familiarity and ecosystem. Consider Fastify for v2 if performance bottleneck.

**Example Setup:**
```typescript
import express from 'express';
import helmet from 'helmet';
import cors from 'cors';

const app = express();

app.use(helmet());
app.use(cors({ origin: process.env.FRONTEND_URL }));
app.use(express.json());
app.use(express.urlencoded({ extended: true }));

export default app;
```

---

### 3. Database: PostgreSQL 15

**Purpose:** Primary relational database

**Why PostgreSQL:**
- **ACID Compliance:** Transactions, data integrity, reliability
- **JSON Support:** JSONB for flexible schemas (preferences, meals, ingredients)
- **Full-Text Search:** Built-in tsvector, GIN indexes (no Elasticsearch initially)
- **Performance:** Handles millions of rows, excellent query planner
- **Open Source:** No licensing costs, huge community

**Alternatives Considered:**
- **MongoDB:** NoSQL, flexible schema, but lacks ACID, complex queries harder
- **MySQL:** Popular, but weaker JSON support, no full-text ranking
- **Aurora PostgreSQL:** AWS-managed, expensive initially ($200+/month vs $30)

**Decision:** PostgreSQL for ACID + JSON flexibility. Migrate to Aurora if needed at scale.

**Configuration (RDS):**
- Instance: db.t3.small (2 vCPU, 2 GB RAM)
- Storage: 100 GB GP3 SSD (auto-scaling to 1 TB)
- Multi-AZ: Yes (high availability)
- Backups: Automated daily, 30-day retention, PITR enabled

---

### 4. Cache: Redis 7

**Purpose:** In-memory cache and session store

**Why Redis:**
- **Speed:** Microsecond latency, perfect for caching
- **Versatility:** Cache, sessions, rate limiting, job queues (Bull)
- **Data Structures:** Strings, hashes, lists, sets (flexible use cases)
- **Persistence:** RDB snapshots + AOF log (durability)
- **Replication:** Primary-replica for high availability

**Alternatives Considered:**
- **Memcached:** Simpler, faster, but no persistence, fewer data structures
- **DynamoDB:** AWS-managed, expensive for caching use case
- **Application-level cache:** LRU cache, but not shared across servers

**Configuration (ElastiCache):**
- Instance: cache.t3.micro (2 vCPU, 0.5 GB RAM)
- Nodes: 2 (primary + replica, auto-failover)
- Encryption: In-transit and at-rest

**Cache Strategy:**
```typescript
// Cache recipe details (1 hour TTL)
const cacheKey = `recipe:${id}`;
const cached = await redis.get(cacheKey);

if (cached) {
  return JSON.parse(cached);
}

const recipe = await db.recipes.findById(id);
await redis.setex(cacheKey, 3600, JSON.stringify(recipe));
return recipe;
```

---

### 5. Cloud Provider: AWS

**Purpose:** Infrastructure hosting

**Why AWS:**
- **Market Leader:** 32% cloud market share, most mature platform
- **Service Breadth:** RDS, ElastiCache, S3, ECS, Personalize (all-in-one)
- **Reliability:** 99.99% SLA, global infrastructure
- **Pricing:** Competitive, free tier for 12 months
- **Ecosystem:** Largest community, best documentation

**Alternatives Considered:**
- **GCP:** Good for ML (but AWS Personalize sufficient), smaller ecosystem
- **Azure:** Enterprise focus, less popular for startups
- **DigitalOcean:** Simple, cheap, but limited managed services

**Key AWS Services:**
| Service | Purpose | Monthly Cost (Year 1) |
|---------|---------|----------------------|
| ECS Fargate | API containers | $50 |
| RDS PostgreSQL | Database | $30 |
| ElastiCache Redis | Cache | $15 |
| S3 + CloudFront | Image storage/CDN | $20 |
| Secrets Manager | Credentials | $5 |
| CloudWatch | Monitoring/logs | $5 |
| **Total** | | **$125** |

---

### 6. Storage: AWS S3 + CloudFront

**Purpose:** Image storage and delivery

**Why S3 + CloudFront:**
- **Durability:** 99.999999999% (11 nines)
- **Scalability:** Unlimited storage, auto-scaling
- **CDN Integration:** CloudFront for global low-latency delivery
- **Cost-Effective:** $0.023/GB storage, $0.085/GB transfer (first 10 TB)

**Bucket Structure:**
```
meal-planner-images-production/
├── recipes/
│   └── {recipeId}/
│       ├── original/image.jpg
│       ├── large/image_1200x800.webp
│       ├── medium/image_600x400.webp
│       └── thumbnail/image_200x200.webp
└── temp/ (24-hour lifecycle policy)
```

---

### 7. ORM: Prisma

**Purpose:** Database ORM and migrations

**Why Prisma:**
- **Type-Safe:** Auto-generated TypeScript types from schema
- **Developer Experience:** Intuitive API, excellent autocomplete
- **Migrations:** Built-in migration tool, version control
- **Performance:** Optimized queries, connection pooling

**Alternatives Considered:**
- **TypeORM:** Mature, Active Record pattern, but less type-safe
- **Sequelize:** Popular, but callback-heavy, no TypeScript types
- **Knex.js:** Query builder, flexible, but manual typing

**Schema Example:**
```prisma
model User {
  id            String    @id @default(uuid())
  email         String    @unique
  passwordHash  String    @map("password_hash")
  name          String
  role          String    @default("user")
  preferences   Json      @default("{}")
  createdAt     DateTime  @default(now()) @map("created_at")
  updatedAt     DateTime  @updatedAt @map("updated_at")
  deletedAt     DateTime? @map("deleted_at")

  mealPlans     MealPlan[]
  favorites     Favorite[]

  @@map("users")
}
```

---

### 8. Authentication: JWT (jsonwebtoken)

**Purpose:** Stateless authentication

**Why JWT:**
- **Stateless:** No server-side session storage, scalable
- **Self-Contained:** All user info in token payload
- **Standard:** RFC 7519, widely adopted
- **Flexible:** Works across domains, mobile apps

**Alternatives Considered:**
- **Session-Based:** Requires session store (Redis), not stateless
- **OAuth 2.0 Only:** Adds complexity, requires provider setup

**Token Structure:**
```typescript
{
  "userId": "550e8400-e29b-41d4-a716-446655440000",
  "email": "user@example.com",
  "role": "user",
  "iat": 1697280000,
  "exp": 1697283600  // 1 hour expiry
}
```

**Security:**
- Algorithm: HS256 (HMAC SHA-256)
- Secret: 256-bit random (stored in AWS Secrets Manager)
- Access token expiry: 1 hour
- Refresh token expiry: 7 days
- Blacklist on logout (Redis)

---

### 9. Validation: Zod

**Purpose:** Runtime type validation

**Why Zod:**
- **TypeScript-First:** Infers TypeScript types from schemas
- **Composable:** Build complex schemas from simple ones
- **Error Messages:** Clear, user-friendly validation errors
- **Performance:** Fast, minimal overhead

**Alternatives Considered:**
- **Joi:** Popular, but doesn't infer TypeScript types
- **Yup:** Good, but Zod better TypeScript integration
- **express-validator:** Works, but less type-safe

**Example:**
```typescript
import { z } from 'zod';

const registerSchema = z.object({
  email: z.string().email(),
  password: z.string().min(8).regex(/[A-Z]/).regex(/[0-9]/),
  name: z.string().min(2).max(255),
});

type RegisterInput = z.infer<typeof registerSchema>;
```

---

### 10. Testing: Jest + Supertest

**Purpose:** Unit and integration testing

**Why Jest:**
- **All-in-One:** Test runner, assertion library, mocking, coverage
- **Fast:** Parallel test execution
- **Snapshot Testing:** UI component testing
- **Popular:** Used by Facebook, Airbnb

**Why Supertest:**
- **HTTP Testing:** Test Express endpoints easily
- **Integration:** Works seamlessly with Jest

**Example:**
```typescript
describe('POST /auth/register', () => {
  it('should register new user', async () => {
    const res = await request(app)
      .post('/api/v1/auth/register')
      .send({
        email: 'test@example.com',
        password: 'Password123!',
        name: 'Test User',
      });

    expect(res.status).toBe(201);
    expect(res.body.data.user.email).toBe('test@example.com');
  });
});
```

---

### 11. Deployment: Docker + ECS Fargate

**Purpose:** Containerization and orchestration

**Why Docker:**
- **Consistency:** Same environment dev → production
- **Isolation:** Dependencies packaged, no conflicts
- **Portability:** Run anywhere (AWS, GCP, on-prem)

**Why ECS Fargate:**
- **Serverless:** No EC2 instances to manage
- **Auto-Scaling:** Scale to zero, pay per use
- **Integration:** Native AWS (ALB, CloudWatch, ECR)
- **Cost:** Cheaper than Kubernetes for small scale

**Alternatives Considered:**
- **Kubernetes (EKS):** Powerful, but overkill for initial scale, complex
- **EC2 + Docker:** Manual management, less auto-scaling

**Dockerfile:**
```dockerfile
FROM node:20-alpine AS build
WORKDIR /app
COPY package*.json ./
RUN npm ci --only=production
COPY . .
RUN npm run build

FROM node:20-alpine
WORKDIR /app
COPY --from=build /app/dist ./dist
COPY --from=build /app/node_modules ./node_modules
EXPOSE 3000
CMD ["node", "dist/server.js"]
```

---

### 12. CI/CD: GitHub Actions

**Purpose:** Automated testing and deployment

**Why GitHub Actions:**
- **Integrated:** Built into GitHub, no third-party setup
- **Free:** 2,000 minutes/month for private repos
- **Flexible:** YAML workflows, easy to customize
- **Marketplace:** Pre-built actions for AWS, Docker, testing

**Alternatives Considered:**
- **GitLab CI:** Good, but requires GitLab
- **CircleCI:** Popular, but costs $$ after free tier
- **Jenkins:** Powerful, but requires self-hosting

**Workflow:**
```yaml
name: Deploy
on:
  push:
    branches: [main]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-node@v3
      - run: npm ci
      - run: npm test
  deploy:
    needs: test
    runs-on: ubuntu-latest
    steps:
      - name: Deploy to ECS
        run: |
          aws ecs update-service --cluster meal-planner --service api --force-new-deployment
```

---

### 13. Monitoring: CloudWatch + Sentry

**Purpose:** Observability and error tracking

**Why CloudWatch:**
- **Native AWS:** Logs, metrics, alarms integrated
- **Cost-Effective:** $5/month for basic monitoring
- **Dashboards:** Real-time visualization

**Why Sentry:**
- **Error Tracking:** Stack traces, user context, breadcrumbs
- **Alerting:** Slack/email notifications
- **Performance:** APM for slow transactions

**Configuration:**
```typescript
import * as Sentry from '@sentry/node';

Sentry.init({
  dsn: process.env.SENTRY_DSN,
  environment: process.env.NODE_ENV,
  tracesSampleRate: 0.1, // 10% of transactions
});

app.use(Sentry.Handlers.requestHandler());
app.use(Sentry.Handlers.errorHandler());
```

---

### 14. Email: SendGrid

**Purpose:** Transactional email delivery

**Why SendGrid:**
- **Reliability:** 99% deliverability rate
- **Templates:** Drag-and-drop email builder
- **Analytics:** Open rates, click rates, bounces
- **Free Tier:** 100 emails/day free

**Alternatives Considered:**
- **AWS SES:** Cheaper ($0.10/1K emails), but less features
- **Mailgun:** Similar to SendGrid, slightly more expensive

**Configuration:**
```typescript
import sgMail from '@sendgrid/mail';

sgMail.setApiKey(process.env.SENDGRID_API_KEY);

await sgMail.send({
  to: user.email,
  from: 'noreply@mealplanner.com',
  subject: 'Welcome!',
  html: '<h1>Welcome to Meal Planner</h1>',
});
```

---

### 15. ML: AWS Personalize

**Purpose:** AI meal recommendations

**Why AWS Personalize:**
- **Managed:** No ML ops, automatic retraining
- **Proven:** Same tech as Amazon.com recommendations
- **Real-Time:** < 100ms inference
- **Scalable:** Handles millions of users

**Alternatives Considered:**
- **Custom ML (Python):** Flexible, but requires ML expertise, ops overhead
- **TensorFlow Recommenders:** Open-source, but manual deployment
- **Collaborative Filtering (custom):** Simple, but less accurate

**Decision:** AWS Personalize for MVP, migrate to custom if needed for advanced features.

---

## NPM Dependencies

**Production:**
```json
{
  "dependencies": {
    "@prisma/client": "^5.7.0",
    "@sentry/node": "^7.80.0",
    "bcryptjs": "^2.4.3",
    "bull": "^4.11.5",
    "cors": "^2.8.5",
    "dotenv": "^16.3.1",
    "express": "^4.18.2",
    "express-rate-limit": "^7.1.5",
    "helmet": "^7.1.0",
    "ioredis": "^5.3.2",
    "jsonwebtoken": "^9.0.2",
    "winston": "^3.11.0",
    "zod": "^3.22.4",
    "@sendgrid/mail": "^8.1.0",
    "aws-sdk": "^2.1495.0"
  },
  "devDependencies": {
    "@types/node": "^20.10.0",
    "@types/express": "^4.17.21",
    "@types/jest": "^29.5.10",
    "jest": "^29.7.0",
    "supertest": "^6.3.3",
    "ts-jest": "^29.1.1",
    "typescript": "^5.3.2",
    "prisma": "^5.7.0"
  }
}
```

---

## Summary

| Category | Technology | Reason |
|----------|-----------|--------|
| Runtime | Node.js 20 | JavaScript consistency, async I/O |
| Framework | Express.js | Battle-tested, flexible |
| Database | PostgreSQL 15 | ACID + JSON support |
| Cache | Redis 7 | Fast, versatile |
| Cloud | AWS | Comprehensive services |
| Storage | S3 + CloudFront | Scalable, durable |
| ORM | Prisma | Type-safe, great DX |
| Auth | JWT | Stateless, scalable |
| Validation | Zod | TypeScript-first |
| Testing | Jest + Supertest | Comprehensive testing |
| Deployment | Docker + ECS | Containerized, auto-scaling |
| CI/CD | GitHub Actions | Integrated, free |
| Monitoring | CloudWatch + Sentry | Observability |
| Email | SendGrid | Reliable delivery |
| ML | AWS Personalize | Managed ML |

**Total Monthly Cost (Year 1):** ~$125/month infrastructure + $0 for most SaaS (free tiers)

---

**Document Version:** 1.0
**Last Updated:** 2025-10-14
**Status:** Approved

This tech stack provides a solid foundation for rapid development, scalability, and maintainability.
