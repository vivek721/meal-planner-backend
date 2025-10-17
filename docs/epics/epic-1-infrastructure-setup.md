# Epic 1: Infrastructure Setup
## Backend Infrastructure & DevOps Foundation

---

**Epic ID:** EPIC-1
**Priority:** P0 (Blocker)
**Estimated Effort:** 40 hours
**Sprint:** Week 1
**Owner:** DevOps Engineer
**Status:** Not Started

---

## Overview

Establish the complete cloud infrastructure, development environment, and CI/CD pipeline required for backend development and deployment. This epic is foundational and must be completed before any application code can be developed or deployed.

## Goals

1. Provision production-ready AWS infrastructure
2. Set up local development environment with Docker
3. Implement automated CI/CD pipeline
4. Configure monitoring and logging infrastructure
5. Establish security best practices from day one

## User Stories

### US-1.1: As a DevOps Engineer, I need AWS infrastructure provisioned

**Acceptance Criteria:**
- AWS account created with billing alerts configured
- VPC with public and private subnets across 2 availability zones
- Security groups configured for ALB, API servers, database, and Redis
- IAM roles and policies created following least privilege principle
- All infrastructure defined in Terraform/CloudFormation

**Technical Details:**
- VPC CIDR: 10.0.0.0/16
- Public subnet: 10.0.1.0/24 (ALB)
- Private subnet: 10.0.2.0/24 (API servers)
- DB subnet: 10.0.3.0/24 (RDS, ElastiCache)

---

### US-1.2: As a Developer, I need a PostgreSQL database ready for development

**Acceptance Criteria:**
- RDS PostgreSQL 15 provisioned (db.t3.small, Multi-AZ)
- Database schema deployed (users, recipes, meal_plans, etc.)
- Connection pooling configured (PgBouncer)
- Automated daily backups enabled (30-day retention)
- PITR (Point-in-Time Recovery) enabled

**Technical Details:**
- Instance: db.t3.small (2 vCPU, 2 GB RAM)
- Storage: 100 GB GP3 SSD (scalable to 1 TB)
- Backups: Daily snapshots at 3 AM UTC
- Encryption: AES-256 at rest

---

### US-1.3: As a Developer, I need Redis cache for session management

**Acceptance Criteria:**
- ElastiCache Redis 7 provisioned (cache.t3.micro)
- Redis cluster configured with automatic failover
- Redis connection tested from API server
- Persistence enabled (RDB snapshots + AOF log)

**Technical Details:**
- Instance: cache.t3.micro (2 vCPU, 0.5 GB RAM)
- Node count: 2 (primary + replica)
- Snapshot retention: 7 days
- Encryption: In-transit and at-rest

---

### US-1.4: As a Developer, I need S3 storage for images

**Acceptance Criteria:**
- S3 bucket created: `meal-planner-images-production`
- Bucket structure defined: `/recipes/{id}/`, `/users/{id}/`, `/temp/`
- CloudFront distribution configured with S3 origin
- Lifecycle policy: Delete `/temp/` files after 24 hours
- CORS policy configured for frontend uploads

**Technical Details:**
- Bucket: meal-planner-images-production
- Region: us-east-1
- Versioning: Disabled
- CDN: CloudFront with edge caching (1 week TTL)

---

### US-1.5: As a Developer, I need Docker environment for local development

**Acceptance Criteria:**
- Dockerfile created for Node.js API server
- docker-compose.yml includes: API, PostgreSQL, Redis, Nginx
- All services start with `docker-compose up`
- Hot reload working for code changes
- Environment variables loaded from .env file

**Technical Details:**
```yaml
services:
  api:
    build: .
    ports:
      - "3000:3000"
    volumes:
      - ./src:/app/src
    depends_on:
      - db
      - redis

  db:
    image: postgres:15-alpine
    environment:
      POSTGRES_DB: mealplanner
      POSTGRES_USER: admin
      POSTGRES_PASSWORD: dev_password
    volumes:
      - postgres_data:/var/lib/postgresql/data

  redis:
    image: redis:7-alpine
    volumes:
      - redis_data:/data
```

---

### US-1.6: As a Developer, I need CI/CD pipeline for automated deployments

**Acceptance Criteria:**
- GitHub Actions workflow created: `.github/workflows/deploy.yml`
- Pipeline stages: Lint → Test → Build → Deploy
- Staging environment deploys on merge to `develop` branch
- Production deploys on merge to `main` branch (manual approval)
- Rollback mechanism implemented

**Pipeline Stages:**
1. **Lint**: ESLint, Prettier checks
2. **Test**: Run unit and integration tests (Jest)
3. **Build**: Build Docker image, tag with commit SHA
4. **Push**: Push image to ECR
5. **Deploy**: Update ECS service with new image
6. **Verify**: Run smoke tests, rollback on failure

---

### US-1.7: As a DevOps Engineer, I need monitoring and logging configured

**Acceptance Criteria:**
- CloudWatch log groups created for API, database, Redis
- CloudWatch alarms configured (CPU, memory, errors)
- Sentry integrated for error tracking
- CloudWatch dashboard created with key metrics
- PagerDuty integration for critical alerts

**Alarms:**
- API error rate > 1% for 5 minutes
- Database CPU > 80% for 10 minutes
- Redis memory > 90%
- ALB 5xx errors > 10 in 5 minutes

---

### US-1.8: As a Security Engineer, I need secrets management configured

**Acceptance Criteria:**
- AWS Secrets Manager configured
- Database credentials stored in Secrets Manager
- JWT secret generated and stored
- SendGrid API key stored
- Automatic secret rotation enabled for database password

**Secrets:**
- `production/db/password`
- `production/jwt/secret`
- `production/sendgrid/api-key`
- `production/aws/access-keys`

---

## Technical Requirements

### AWS Services Required

| Service | Configuration | Purpose |
|---------|--------------|---------|
| **VPC** | 10.0.0.0/16, 3 subnets, 2 AZs | Network isolation |
| **RDS PostgreSQL** | db.t3.small, Multi-AZ | Primary database |
| **ElastiCache Redis** | cache.t3.micro, 2 nodes | Session cache |
| **S3** | meal-planner-images-production | Image storage |
| **CloudFront** | CDN for S3 | Global content delivery |
| **ALB** | Application Load Balancer | Traffic distribution |
| **ECS Fargate** | 2 tasks, 0.5 vCPU, 1 GB RAM | API containers |
| **ECR** | Docker registry | Container images |
| **Secrets Manager** | 5 secrets | Credentials management |
| **CloudWatch** | Logs, metrics, alarms | Monitoring |

### Estimated Monthly Cost (Year 1)

```
ECS Fargate (2 tasks × 0.5 vCPU)    $50
RDS PostgreSQL (db.t3.small)         $30
ElastiCache Redis (cache.t3.micro)   $15
S3 + CloudFront                      $20
Data Transfer                        $10
────────────────────────────────────────
Total                               ~$125/month
```

---

## Database Schema Deployment

### Migration Files

```sql
-- migrations/001_create_users.sql
CREATE TABLE users (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  email VARCHAR(255) NOT NULL UNIQUE,
  password_hash VARCHAR(255) NOT NULL,
  name VARCHAR(255) NOT NULL,
  role VARCHAR(50) NOT NULL DEFAULT 'user',
  preferences JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_users_email ON users(email) WHERE deleted_at IS NULL;
CREATE INDEX idx_users_created_at ON users(created_at);
```

### Seed Data

```sql
-- seeds/001_admin_user.sql
INSERT INTO users (email, password_hash, name, role)
VALUES (
  'admin@mealplanner.com',
  '$2b$12$...',  -- bcrypt hash of 'AdminPassword123!'
  'System Administrator',
  'admin'
);

-- seeds/002_sample_recipes.sql
-- Insert 10 sample recipes for testing
```

---

## CI/CD Pipeline Configuration

### GitHub Actions Workflow

```yaml
name: Deploy to AWS

on:
  push:
    branches:
      - develop  # Staging
      - main     # Production

jobs:
  lint:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-node@v3
        with:
          node-version: '20'
      - run: npm ci
      - run: npm run lint

  test:
    runs-on: ubuntu-latest
    needs: lint
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-node@v3
      - run: npm ci
      - run: npm test
      - name: Upload coverage
        uses: codecov/codecov-action@v3

  build:
    runs-on: ubuntu-latest
    needs: test
    steps:
      - uses: actions/checkout@v3
      - name: Configure AWS credentials
        uses: aws-actions/configure-aws-credentials@v2
        with:
          aws-access-key-id: ${{ secrets.AWS_ACCESS_KEY_ID }}
          aws-secret-access-key: ${{ secrets.AWS_SECRET_ACCESS_KEY }}
          aws-region: us-east-1

      - name: Build Docker image
        run: |
          docker build -t meal-planner-api:${{ github.sha }} .

      - name: Push to ECR
        run: |
          aws ecr get-login-password | docker login --username AWS --password-stdin ${{ secrets.ECR_REGISTRY }}
          docker tag meal-planner-api:${{ github.sha }} ${{ secrets.ECR_REGISTRY }}/meal-planner-api:${{ github.sha }}
          docker push ${{ secrets.ECR_REGISTRY }}/meal-planner-api:${{ github.sha }}

  deploy:
    runs-on: ubuntu-latest
    needs: build
    steps:
      - name: Deploy to ECS
        run: |
          aws ecs update-service \
            --cluster meal-planner-cluster \
            --service meal-planner-api \
            --force-new-deployment

      - name: Wait for deployment
        run: |
          aws ecs wait services-stable \
            --cluster meal-planner-cluster \
            --services meal-planner-api

      - name: Run smoke tests
        run: |
          curl -f https://api.mealplanner.com/health || exit 1
```

---

## Monitoring Configuration

### CloudWatch Alarms

```json
{
  "AlarmName": "HighAPIErrorRate",
  "MetricName": "5XXError",
  "Namespace": "AWS/ApplicationELB",
  "Statistic": "Sum",
  "Period": 300,
  "EvaluationPeriods": 1,
  "Threshold": 10,
  "ComparisonOperator": "GreaterThanThreshold",
  "AlarmActions": ["arn:aws:sns:us-east-1:123456789012:critical-alerts"]
}
```

### CloudWatch Dashboard

Metrics to display:
- API request rate (requests/minute)
- API error rate (%)
- P50, P95, P99 response times
- Database connections (active/max)
- Redis memory usage (%)
- Cache hit rate (%)

---

## Acceptance Criteria

### Definition of Done

- [ ] All AWS infrastructure provisioned and accessible
- [ ] Database schema deployed with sample data
- [ ] Redis cache operational
- [ ] S3 bucket and CloudFront configured
- [ ] Docker environment working locally (`docker-compose up`)
- [ ] CI/CD pipeline successfully deploys to staging
- [ ] Health check endpoint (`GET /health`) returns 200 OK
- [ ] Monitoring dashboard shows all metrics
- [ ] Secrets stored in AWS Secrets Manager
- [ ] Documentation updated (README, runbooks)

### Testing Requirements

- [ ] API server accessible at `https://api-staging.mealplanner.com`
- [ ] Database connection test passes
- [ ] Redis connection test passes
- [ ] S3 upload test successful
- [ ] CloudFront serves images correctly
- [ ] CI/CD pipeline deploys without errors
- [ ] Rollback procedure tested and documented

---

## Dependencies

### External Dependencies

- AWS account with admin access
- Domain name registered (e.g., mealplanner.com)
- SSL certificate (AWS Certificate Manager)
- GitHub account for CI/CD

### Internal Dependencies

- None (this is the first epic)

---

## Risks & Mitigation

### Risk 1: AWS Cost Overruns

**Impact:** Medium | **Probability:** Medium

**Mitigation:**
- Set up billing alerts at $50, $100, $150
- Use AWS Cost Explorer to track daily spending
- Right-size instances (start small, scale up)
- Implement auto-scaling max limits

---

### Risk 2: Database Migration Failures

**Impact:** High | **Probability:** Low

**Mitigation:**
- Test migrations in staging first
- Always create backup before migration
- Have rollback SQL scripts ready
- Use migration tool with transaction support (Prisma, Flyway)

---

### Risk 3: CI/CD Pipeline Complexity

**Impact:** Medium | **Probability:** Medium

**Mitigation:**
- Start with simple pipeline, add complexity incrementally
- Thorough documentation of each step
- Manual approval for production deployments
- Automated rollback on health check failure

---

## Deliverables

### Infrastructure as Code

- `terraform/vpc.tf`: VPC, subnets, security groups
- `terraform/rds.tf`: PostgreSQL database
- `terraform/elasticache.tf`: Redis cache
- `terraform/s3.tf`: S3 bucket, CloudFront
- `terraform/ecs.tf`: ECS cluster, task definitions
- `terraform/iam.tf`: IAM roles, policies

### Docker Configuration

- `Dockerfile`: Multi-stage build for API server
- `docker-compose.yml`: Local development environment
- `.dockerignore`: Exclude unnecessary files

### CI/CD Pipeline

- `.github/workflows/deploy.yml`: GitHub Actions workflow
- `scripts/deploy.sh`: Deployment script
- `scripts/rollback.sh`: Rollback script

### Documentation

- `docs/infrastructure/AWS_SETUP.md`: Infrastructure guide
- `docs/infrastructure/DOCKER_SETUP.md`: Docker guide
- `docs/infrastructure/CI_CD.md`: Pipeline documentation
- `docs/runbooks/DEPLOYMENT.md`: Deployment runbook
- `docs/runbooks/ROLLBACK.md`: Rollback runbook

---

## Timeline

### Week 1 Breakdown

| Day | Tasks | Hours |
|-----|-------|-------|
| **Monday** | AWS account setup, VPC configuration | 8 |
| **Tuesday** | RDS, ElastiCache provisioning | 8 |
| **Wednesday** | S3, CloudFront, security groups | 8 |
| **Thursday** | Docker setup, local environment | 8 |
| **Friday** | CI/CD pipeline, monitoring, documentation | 8 |

**Total:** 40 hours

---

## Success Metrics

- Infrastructure provisioned in < 5 days
- Zero manual configuration (100% IaC)
- CI/CD pipeline success rate > 95%
- Deployment time < 10 minutes
- Zero security vulnerabilities in infrastructure scan

---

## Post-Epic Review

### What Went Well

- [To be filled after completion]

### What Could Be Improved

- [To be filled after completion]

### Action Items for Next Epic

- [To be filled after completion]

---

**Epic Status:** Not Started
**Last Updated:** 2025-10-14
**Next Review:** End of Week 1

This epic provides the foundation for all backend development. Once complete, developers can begin implementing authentication (Epic 2).
