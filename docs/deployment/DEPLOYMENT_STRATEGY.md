# Deployment Strategy
## Production Deployment Guide

---

**Version:** 1.0
**Last Updated:** 2025-10-14

---

## Environments

### 1. Development

**Purpose:** Local development
**URL:** `http://localhost:3000`
**Infrastructure:** Docker Compose

**Configuration:**
```yaml
# docker-compose.yml
services:
  api:
    build: .
    ports:
      - "3000:3000"
    environment:
      - NODE_ENV=development
      - DATABASE_URL=postgresql://admin:password@db:5432/mealplanner
      - REDIS_URL=redis://redis:6379
```

---

### 2. Staging

**Purpose:** Testing, QA, integration
**URL:** `https://api-staging.mealplanner.com`
**Infrastructure:** AWS ECS Fargate

**Configuration:**
- **ECS:** 1 task (0.5 vCPU, 1 GB RAM)
- **RDS:** db.t3.micro (1 GB RAM)
- **Redis:** cache.t3.micro (0.5 GB RAM)
- **Auto-Deploy:** On merge to `develop` branch

---

### 3. Production

**Purpose:** Live application
**URL:** `https://api.mealplanner.com`
**Infrastructure:** AWS ECS Fargate (Multi-AZ)

**Configuration:**
- **ECS:** 2-20 tasks (auto-scaling)
- **RDS:** db.t3.small Multi-AZ (2 GB RAM)
- **Redis:** cache.t3.micro with replica
- **Deploy:** Manual approval on merge to `main`

---

## Infrastructure as Code

### Terraform

**Structure:**
```
terraform/
├── main.tf
├── variables.tf
├── outputs.tf
├── modules/
│   ├── vpc/
│   ├── rds/
│   ├── elasticache/
│   ├── ecs/
│   └── s3/
└── environments/
    ├── staging/
    └── production/
```

**Example (VPC):**
```hcl
# terraform/modules/vpc/main.tf
resource "aws_vpc" "main" {
  cidr_block           = var.vpc_cidr
  enable_dns_hostnames = true
  enable_dns_support   = true

  tags = {
    Name        = "${var.environment}-vpc"
    Environment = var.environment
  }
}

resource "aws_subnet" "public" {
  count                   = 2
  vpc_id                  = aws_vpc.main.id
  cidr_block              = cidrsubnet(var.vpc_cidr, 8, count.index)
  availability_zone       = data.aws_availability_zones.available.names[count.index]
  map_public_ip_on_launch = true

  tags = {
    Name = "${var.environment}-public-subnet-${count.index + 1}"
  }
}
```

---

## CI/CD Pipeline

### GitHub Actions Workflow

```yaml
name: Deploy to Production

on:
  push:
    branches: [main]
  workflow_dispatch:

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

      - name: Login to ECR
        id: login-ecr
        uses: aws-actions/amazon-ecr-login@v1

      - name: Build and push Docker image
        env:
          ECR_REGISTRY: ${{ steps.login-ecr.outputs.registry }}
          ECR_REPOSITORY: meal-planner-api
          IMAGE_TAG: ${{ github.sha }}
        run: |
          docker build -t $ECR_REGISTRY/$ECR_REPOSITORY:$IMAGE_TAG .
          docker tag $ECR_REGISTRY/$ECR_REPOSITORY:$IMAGE_TAG $ECR_REGISTRY/$ECR_REPOSITORY:latest
          docker push $ECR_REGISTRY/$ECR_REPOSITORY:$IMAGE_TAG
          docker push $ECR_REGISTRY/$ECR_REPOSITORY:latest

  deploy:
    runs-on: ubuntu-latest
    needs: build
    environment: production
    steps:
      - name: Configure AWS credentials
        uses: aws-actions/configure-aws-credentials@v2
        with:
          aws-access-key-id: ${{ secrets.AWS_ACCESS_KEY_ID }}
          aws-secret-access-key: ${{ secrets.AWS_SECRET_ACCESS_KEY }}
          aws-region: us-east-1

      - name: Deploy to ECS
        run: |
          aws ecs update-service \
            --cluster meal-planner-production \
            --service meal-planner-api \
            --force-new-deployment

      - name: Wait for deployment
        run: |
          aws ecs wait services-stable \
            --cluster meal-planner-production \
            --services meal-planner-api

      - name: Run smoke tests
        run: |
          curl -f https://api.mealplanner.com/health || exit 1
```

---

## Database Migrations

### Prisma Migrations

**Create Migration:**
```bash
npx prisma migrate dev --name add_cuisine_to_recipes
```

**Apply to Production:**
```bash
# 1. Create snapshot first
aws rds create-db-snapshot \
  --db-instance-identifier meal-planner-prod \
  --db-snapshot-identifier pre-migration-$(date +%Y%m%d)

# 2. Run migration
npx prisma migrate deploy

# 3. Verify
npx prisma db pull
```

**Rollback:**
```bash
# Restore from snapshot
aws rds restore-db-instance-from-db-snapshot \
  --db-instance-identifier meal-planner-prod \
  --db-snapshot-identifier pre-migration-20251014
```

---

## Deployment Methods

### 1. Blue-Green Deployment

**Process:**
1. Deploy new version (green) alongside old (blue)
2. Route small % of traffic to green (canary)
3. Monitor metrics (errors, latency)
4. Gradually increase traffic to green
5. Decommission blue after 24 hours

**ECS Task Definition:**
```json
{
  "family": "meal-planner-api",
  "taskRoleArn": "arn:aws:iam::123456789012:role/ecsTaskRole",
  "executionRoleArn": "arn:aws:iam::123456789012:role/ecsExecutionRole",
  "networkMode": "awsvpc",
  "containerDefinitions": [
    {
      "name": "api",
      "image": "123456789012.dkr.ecr.us-east-1.amazonaws.com/meal-planner-api:latest",
      "cpu": 512,
      "memory": 1024,
      "essential": true,
      "portMappings": [
        {
          "containerPort": 3000,
          "protocol": "tcp"
        }
      ],
      "environment": [
        {
          "name": "NODE_ENV",
          "value": "production"
        }
      ],
      "secrets": [
        {
          "name": "DATABASE_URL",
          "valueFrom": "arn:aws:secretsmanager:us-east-1:123456789012:secret:production/db/url"
        }
      ],
      "logConfiguration": {
        "logDriver": "awslogs",
        "options": {
          "awslogs-group": "/ecs/meal-planner-api",
          "awslogs-region": "us-east-1",
          "awslogs-stream-prefix": "ecs"
        }
      }
    }
  ],
  "requiresCompatibilities": ["FARGATE"],
  "cpu": "512",
  "memory": "1024"
}
```

---

### 2. Rolling Deployment

**Process:**
1. Update 1 task at a time
2. Wait for health checks to pass
3. Continue to next task
4. Complete when all tasks updated

**Advantages:**
- Zero downtime
- Gradual rollout
- Easy rollback

---

## Monitoring & Validation

### Health Checks

**ECS Health Check:**
```json
{
  "healthCheck": {
    "command": ["CMD-SHELL", "curl -f http://localhost:3000/health || exit 1"],
    "interval": 30,
    "timeout": 5,
    "retries": 3,
    "startPeriod": 60
  }
}
```

**Application Health Endpoint:**
```typescript
app.get('/health', async (req, res) => {
  const health = {
    status: 'healthy',
    timestamp: new Date().toISOString(),
    uptime: process.uptime(),
    database: 'unknown',
    cache: 'unknown',
  };

  try {
    await prisma.$queryRaw`SELECT 1`;
    health.database = 'healthy';
  } catch {
    health.database = 'unhealthy';
    health.status = 'unhealthy';
  }

  try {
    await redis.ping();
    health.cache = 'healthy';
  } catch {
    health.cache = 'unhealthy';
    health.status = 'degraded';
  }

  const statusCode = health.status === 'healthy' ? 200 : 503;
  res.status(statusCode).json(health);
});
```

---

### Post-Deployment Checks

**Automated Smoke Tests:**
```bash
#!/bin/bash
# scripts/smoke-test.sh

API_URL="https://api.mealplanner.com/api/v1"

echo "1. Health check..."
curl -f $API_URL/health || exit 1

echo "2. Authentication..."
TOKEN=$(curl -s -X POST $API_URL/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","password":"Test123!"}' \
  | jq -r '.data.tokens.accessToken')

echo "3. Recipe browsing..."
curl -f -H "Authorization: Bearer $TOKEN" $API_URL/recipes || exit 1

echo "4. Meal plan creation..."
curl -f -X POST -H "Authorization: Bearer $TOKEN" $API_URL/meal-plans \
  -H "Content-Type: application/json" \
  -d '{"weekStart":"2025-10-13"}' || exit 1

echo "All smoke tests passed!"
```

---

## Rollback Procedures

### Application Rollback

**ECS Rollback (< 5 minutes):**
```bash
# 1. Get previous task definition
PREVIOUS_TASK_DEF=$(aws ecs describe-services \
  --cluster meal-planner-production \
  --services meal-planner-api \
  --query 'services[0].deployments[1].taskDefinition' \
  --output text)

# 2. Update service to previous version
aws ecs update-service \
  --cluster meal-planner-production \
  --service meal-planner-api \
  --task-definition $PREVIOUS_TASK_DEF

# 3. Monitor rollback
aws ecs wait services-stable \
  --cluster meal-planner-production \
  --services meal-planner-api
```

---

### Database Rollback

**Option 1: Restore from Snapshot**
```bash
aws rds restore-db-instance-from-db-snapshot \
  --db-instance-identifier meal-planner-prod \
  --db-snapshot-identifier pre-deployment-20251014
```

**Option 2: Reverse Migration**
```bash
# Manual SQL rollback
psql $DATABASE_URL -f rollback/001_reverse_migration.sql
```

---

## Disaster Recovery

### Backup Strategy

**Database:**
- Automated daily snapshots (RDS)
- 30-day retention
- Point-in-time recovery (PITR)
- Manual snapshot before migrations

**Code:**
- Git version control (GitHub)
- Docker images in ECR (tagged by commit SHA)

**Configuration:**
- Secrets in AWS Secrets Manager
- Infrastructure in Terraform (version controlled)

---

### Recovery Time Objectives

| Component | RTO | RPO |
|-----------|-----|-----|
| Application | 15 minutes | 0 (stateless) |
| Database | 1 hour | 5 minutes |
| Redis Cache | 5 minutes | 0 (rebuilt) |
| S3 Images | N/A | 0 (durable) |

---

## Scaling Strategy

### Auto-Scaling Configuration

```hcl
resource "aws_appautoscaling_target" "ecs_target" {
  max_capacity       = 20
  min_capacity       = 2
  resource_id        = "service/meal-planner-production/meal-planner-api"
  scalable_dimension = "ecs:service:DesiredCount"
  service_namespace  = "ecs"
}

resource "aws_appautoscaling_policy" "cpu_scaling" {
  name               = "cpu-scaling"
  policy_type        = "TargetTrackingScaling"
  resource_id        = aws_appautoscaling_target.ecs_target.resource_id
  scalable_dimension = aws_appautoscaling_target.ecs_target.scalable_dimension
  service_namespace  = aws_appautoscaling_target.ecs_target.service_namespace

  target_tracking_scaling_policy_configuration {
    target_value       = 70.0
    predefined_metric_specification {
      predefined_metric_type = "ECSServiceAverageCPUUtilization"
    }
    scale_in_cooldown  = 300
    scale_out_cooldown = 60
  }
}
```

---

## Deployment Checklist

### Pre-Deployment

- [ ] All tests passing (unit, integration)
- [ ] Code review approved
- [ ] Security scan passed (npm audit)
- [ ] Database migration tested in staging
- [ ] Environment variables configured
- [ ] Secrets stored in Secrets Manager
- [ ] Stakeholder approval received

### Deployment

- [ ] Create database snapshot
- [ ] Run database migrations
- [ ] Deploy new Docker image
- [ ] Wait for ECS stabilization
- [ ] Run smoke tests
- [ ] Monitor error rates (< 0.1%)
- [ ] Verify P95 latency (< 200ms)

### Post-Deployment

- [ ] Monitor for 1 hour
- [ ] Check CloudWatch metrics
- [ ] Review Sentry for new errors
- [ ] Verify frontend integration
- [ ] Update documentation
- [ ] Notify team of successful deployment

---

**Document Version:** 1.0
**Last Updated:** 2025-10-14
