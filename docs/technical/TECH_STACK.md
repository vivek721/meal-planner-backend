# Technology Stack
## AI-Powered Meal Planner Backend

---

**Version:** 2.0
**Last Updated:** 2025-10-17
**Migration:** Node.js/TypeScript → Golang (October 2025)

---

## Overview

This document provides detailed justification for all technology choices in the backend stack, including alternatives considered and decision rationale.

**Technology Migration Note:**
This backend was initially planned for Node.js + Express + TypeScript but was migrated to **Golang** for superior performance, simpler deployment, and better concurrency. This document reflects the current Golang implementation.

---

## Core Technologies

### 1. Runtime: Go 1.21+

**Purpose:** Primary programming language for backend services

**Why Golang:**
- **Performance:** Compiled language, 10-40x faster than Node.js for CPU-intensive tasks
- **Concurrency:** Built-in goroutines and channels for efficient concurrent operations
- **Simple Deployment:** Single binary, no runtime dependencies
- **Memory Efficiency:** Low memory footprint (~15MB idle vs ~50MB+ for Node.js)
- **Type Safety:** Strong static typing with compile-time and runtime checks
- **Standard Library:** Comprehensive stdlib reduces external dependencies
- **Fast Compilation:** Sub-second builds for rapid development
- **Production Ready:** Used by Google, Uber, Dropbox, Docker, Kubernetes

**Alternatives Considered:**
- **Node.js + TypeScript:** Originally planned, but Golang offers better performance and simpler deployment
- **Python + FastAPI:** Good for ML, but slower and heavier than Go
- **Java + Spring Boot:** Enterprise-grade, but much heavier and slower builds
- **Rust:** Excellent performance, but steeper learning curve and longer compile times

**Configuration:**
```go
// go.mod
module meal-planner-backend

go 1.21

require (
    github.com/gin-gonic/gin v1.9.1
    gorm.io/gorm v1.25.5
    gorm.io/driver/postgres v1.5.4
    github.com/golang-jwt/jwt/v5 v5.2.0
    golang.org/x/crypto v0.17.0
)
```

---

### 2. Framework: Gin Web Framework

**Purpose:** HTTP web framework

**Why Gin:**
- **Performance:** Fastest Go web framework (up to 40x faster than Express.js)
- **Routing:** Radix tree-based router with zero memory allocation
- **Middleware:** Extensive middleware ecosystem
- **JSON Handling:** Built-in JSON validation and binding
- **Error Handling:** Panic recovery and error management
- **Community:** Large ecosystem, actively maintained
- **Production Usage:** Used by major companies worldwide

**Alternatives Considered:**
- **Echo:** Similar performance, slightly different API
- **Fiber:** Express.like API, but uses fasthttp (different from net/http)
- **Chi:** Lightweight, but less feature-rich than Gin
- **Standard net/http:** Minimal, but requires more boilerplate

**Example Setup:**
```go
package main

import (
    "github.com/gin-gonic/gin"
    "github.com/gin-contrib/cors"
)

func main() {
    router := gin.Default()

    // CORS middleware
    config := cors.DefaultConfig()
    config.AllowOrigins = []string{"http://localhost:3000"}
    config.AllowHeaders = []string{"Authorization", "Content-Type"}
    router.Use(cors.New(config))

    // Routes
    router.POST("/api/auth/register", registerHandler)
    router.POST("/api/auth/login", loginHandler)

    router.Run(":3001")
}
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
- **Production Ready:** Battle-tested in production environments worldwide

**Alternatives Considered:**
- **MongoDB:** NoSQL, flexible schema, but lacks ACID, complex queries harder
- **MySQL:** Popular, but weaker JSON support, no full-text ranking
- **Aurora PostgreSQL:** AWS-managed, expensive initially ($200+/month vs $30)

**Decision:** PostgreSQL for ACID + JSON flexibility. Migrate to Aurora if needed at scale.

**Configuration (Local/Docker):**
- Database: `meal_planner`
- User: `postgres`
- Port: 5432
- Connection pooling via GORM

**Configuration (Production):**
- Instance: AWS RDS db.t3.small (2 vCPU, 2 GB RAM)
- Storage: 100 GB GP3 SSD (auto-scaling to 1 TB)
- Multi-AZ: Yes (high availability)
- Backups: Automated daily, 30-day retention, PITR enabled

---

### 4. ORM: GORM

**Purpose:** Database ORM and migrations

**Why GORM:**
- **Feature-Rich:** Associations, hooks, transactions, migrations
- **Developer Experience:** Intuitive, chainable API
- **Type-Safe:** Strong typing with Go structs
- **Auto-Migration:** Automatic schema creation/updates
- **Performance:** Query optimization, connection pooling
- **Community:** Largest Go ORM, actively maintained

**Alternatives Considered:**
- **SQLBoiler:** Code generation approach, complex setup
- **Ent:** Facebook's ORM, powerful but more complex
- **sqlx:** Query builder, flexible but no auto-migrations

**Model Example:**
```go
type User struct {
    ID                      string     `gorm:"primaryKey"`
    Email                   string     `gorm:"unique;not null"`
    PasswordHash            string     `gorm:"column:password_hash;not null"`
    Name                    string
    HasCompletedOnboarding  bool       `gorm:"default:false"`
    CreatedAt               time.Time
    UpdatedAt               time.Time
    DeletedAt               gorm.DeletedAt `gorm:"index"`

    // Login tracking
    LoginAttempts           int        `gorm:"default:0"`
    LastLoginAttempt        *time.Time
    AccountLockedUntil      *time.Time

    // Preferences (embedded)
    PrefTheme               string     `gorm:"default:'light'"`
    PrefNotifications       bool       `gorm:"default:true"`
}
```

---

### 5. Authentication: JWT (golang-jwt)

**Purpose:** Stateless authentication

**Why JWT:**
- **Stateless:** No server-side session storage, scalable
- **Self-Contained:** All user info in token payload
- **Standard:** RFC 7519, widely adopted
- **Flexible:** Works across domains, mobile apps
- **Go Library:** `github.com/golang-jwt/jwt/v5` - official JWT implementation

**Alternatives Considered:**
- **Session-Based:** Requires session store (Redis), not stateless
- **OAuth 2.0 Only:** Adds complexity, requires provider setup

**Token Structure:**
```go
type Claims struct {
    UserID string `json:"userId"`
    Email  string `json:"email"`
    jwt.RegisteredClaims
}

// Token generation
token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
signedToken, _ := token.SignedString([]byte(jwtSecret))
```

**Security:**
- Algorithm: HS256 (HMAC SHA-256)
- Secret: 256-bit random (stored in environment variables)
- Access token expiry: 24 hours (configurable)
- Refresh mechanism: 30-day refresh period
- Token validation on all protected routes

---

### 6. Password Hashing: Bcrypt

**Purpose:** Secure password hashing

**Why Bcrypt:**
- **Industry Standard:** Battle-tested algorithm
- **Adaptive:** Cost factor can be increased over time
- **Salt Included:** Automatic salt generation
- **Resistant:** Protects against rainbow table attacks
- **Go Support:** `golang.org/x/crypto/bcrypt` - official implementation

**Configuration:**
```go
import "golang.org/x/crypto/bcrypt"

const DefaultCost = 12 // ~250ms on modern hardware

func HashPassword(password string) (string, error) {
    bytes, err := bcrypt.GenerateFromPassword(
        []byte(password),
        DefaultCost,
    )
    return string(bytes), err
}

func CheckPassword(password, hash string) error {
    return bcrypt.CompareHashAndPassword(
        []byte(hash),
        []byte(password),
    )
}
```

---

### 7. Validation: Built-in + Custom Validators

**Purpose:** Input validation

**Why Custom + Built-in:**
- **Native Go:** No external dependencies for basic validation
- **Type Safety:** Compile-time checking with struct tags
- **Custom Logic:** Easy to implement custom validators
- **Performance:** Zero allocation for many operations

**Example:**
```go
// Gin binding with validation
type RegisterRequest struct {
    Email    string `json:"email" binding:"required,email"`
    Password string `json:"password" binding:"required,min=8"`
    Name     string `json:"name"`
}

// Custom validator
func ValidatePasswordStrength(password string) error {
    if len(password) < 8 {
        return errors.New("password must be at least 8 characters")
    }
    hasUpper := regexp.MustCompile(`[A-Z]`).MatchString(password)
    hasLower := regexp.MustCompile(`[a-z]`).MatchString(password)
    hasNumber := regexp.MustCompile(`[0-9]`).MatchString(password)
    hasSpecial := regexp.MustCompile(`[!@#$%^&*]`).MatchString(password)

    if !hasUpper || !hasLower || !hasNumber || !hasSpecial {
        return errors.New("password must contain upper, lower, number, and special character")
    }
    return nil
}
```

---

### 8. Development: Air (Hot Reload)

**Purpose:** Live reload during development

**Why Air:**
- **Fast:** Instant reload on file changes
- **Configurable:** Support for custom build commands
- **Go-Specific:** Designed for Go applications
- **Zero Config:** Works out of the box with sensible defaults

**Configuration:**
```toml
# .air.toml
[build]
  cmd = "go build -o ./bin/meal-planner-api ./cmd/server"
  bin = "./bin/meal-planner-api"
  include_ext = ["go"]
  exclude_dir = ["bin", "vendor"]
  delay = 1000
```

---

### 9. Testing: Go Testing Framework

**Purpose:** Unit and integration testing

**Why Go's Built-in Testing:**
- **Native:** No external framework needed
- **Fast:** Parallel test execution
- **Simple:** Minimal syntax, easy to learn
- **Coverage:** Built-in coverage reporting
- **Benchmarking:** Performance testing included

**Example:**
```go
func TestHashPassword(t *testing.T) {
    password := "TestPassword123!"
    hash, err := HashPassword(password)

    if err != nil {
        t.Errorf("HashPassword failed: %v", err)
    }

    if hash == password {
        t.Error("Hash should not equal plain password")
    }

    if err := CheckPassword(password, hash); err != nil {
        t.Error("Password verification failed")
    }
}
```

---

### 10. Deployment: Docker

**Purpose:** Containerization

**Why Docker:**
- **Consistency:** Same environment dev → production
- **Isolation:** Dependencies packaged, no conflicts
- **Portability:** Run anywhere (AWS, GCP, on-prem)
- **Single Binary:** Go compiles to single executable

**Multi-Stage Dockerfile:**
```dockerfile
# Build stage
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY go.* ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o meal-planner-api ./cmd/server

# Production stage
FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/meal-planner-api .
EXPOSE 3001
CMD ["./meal-planner-api"]
```

**Benefits:**
- **Small Image:** ~15-20MB (Alpine + binary)
- **Fast Startup:** < 100ms
- **Secure:** Minimal attack surface
- **Efficient:** Low resource usage

---

### 11. Cloud Provider: AWS (Planned)

**Purpose:** Infrastructure hosting

**Why AWS:**
- **Market Leader:** 32% cloud market share, most mature platform
- **Service Breadth:** RDS, ElastiCache, S3, ECS, Lambda (all-in-one)
- **Reliability:** 99.99% SLA, global infrastructure
- **Pricing:** Competitive, free tier for 12 months
- **Ecosystem:** Largest community, best documentation

**Alternatives Considered:**
- **GCP:** Good for ML, but AWS has better PostgreSQL support
- **Azure:** Enterprise focus, less popular for startups
- **DigitalOcean:** Simple, cheap, but limited managed services
- **Railway/Render:** Simple deployment, good for MVP

**Key AWS Services (Planned):**
| Service | Purpose | Monthly Cost (Est.) |
|---------|---------|---------------------|
| ECS Fargate | API containers | $30-50 |
| RDS PostgreSQL | Database | $30 |
| ElastiCache Redis | Cache (future) | $15 |
| S3 + CloudFront | Image storage/CDN | $10 |
| Secrets Manager | Credentials | $1 |
| CloudWatch | Monitoring/logs | $5 |
| **Total** | | **~$90-110** |

**Docker Deployment Options:**
- **ECS Fargate:** Serverless containers (recommended)
- **Google Cloud Run:** Serverless containers
- **Azure Container Instances:** Serverless containers
- **Railway/Render:** Simple PaaS deployment
- **DigitalOcean App Platform:** Budget-friendly option

---

### 12. Storage: AWS S3 + CloudFront (Planned)

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

### 13. CI/CD: GitHub Actions

**Purpose:** Automated testing and deployment

**Why GitHub Actions:**
- **Integrated:** Built into GitHub, no third-party setup
- **Free:** 2,000 minutes/month for private repos
- **Flexible:** YAML workflows, easy to customize
- **Marketplace:** Pre-built actions for Go, Docker, AWS

**Alternatives Considered:**
- **GitLab CI:** Good, but requires GitLab
- **CircleCI:** Popular, but costs $$ after free tier
- **Jenkins:** Powerful, but requires self-hosting

**Example Workflow:**
```yaml
name: Test and Build
on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: '1.21'
      - run: go mod download
      - run: go test ./...
      - run: go build ./cmd/server

  build-docker:
    needs: test
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: docker/build-push-action@v4
        with:
          push: false
          tags: meal-planner-api:latest
```

---

### 14. Monitoring: Prometheus + Grafana (Planned)

**Purpose:** Observability and metrics

**Why Prometheus + Grafana:**
- **Open Source:** Free, community-driven
- **Go Native:** Excellent Go instrumentation libraries
- **Powerful:** Time-series database, flexible queries
- **Visualization:** Grafana dashboards
- **Alerting:** Built-in alert manager

**Alternatives:**
- **CloudWatch:** AWS-native, simpler but less powerful
- **Datadog:** Commercial, expensive but feature-rich
- **New Relic:** APM focused, expensive

**Go Prometheus Example:**
```go
import (
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
    httpRequestsTotal = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "http_requests_total",
            Help: "Total HTTP requests",
        },
        []string{"method", "endpoint", "status"},
    )
)

func init() {
    prometheus.MustRegister(httpRequestsTotal)
}

// Metrics endpoint
router.GET("/metrics", gin.WrapH(promhttp.Handler()))
```

---

## Go Dependencies

**Core Dependencies:**
```go
require (
    github.com/gin-gonic/gin v1.9.1              // Web framework
    gorm.io/gorm v1.25.5                          // ORM
    gorm.io/driver/postgres v1.5.4                // PostgreSQL driver
    github.com/golang-jwt/jwt/v5 v5.2.0           // JWT authentication
    golang.org/x/crypto v0.17.0                   // Bcrypt, crypto
    github.com/joho/godotenv v1.5.1               // Environment variables
    github.com/gin-contrib/cors v1.4.0            // CORS middleware
)
```

**Development Dependencies:**
```go
require (
    github.com/cosmtrek/air v1.49.0               // Hot reload
    github.com/stretchr/testify v1.8.4            // Testing assertions
)
```

---

## Comparison: Node.js vs Golang

| Aspect | Node.js + Express | Golang + Gin |
|--------|------------------|--------------|
| **Performance** | Moderate (event loop) | High (compiled, goroutines) |
| **Memory Usage** | ~50-100MB | ~15-30MB |
| **Concurrency** | Event loop (single-threaded) | Goroutines (multi-threaded) |
| **Type Safety** | TypeScript (compile-time) | Go (compile + runtime) |
| **Deployment** | Node runtime + dependencies | Single binary |
| **Build Time** | 5-10 seconds | < 1 second |
| **Learning Curve** | Low (JavaScript familiar) | Moderate |
| **Ecosystem** | 2M+ npm packages | Smaller but high-quality |
| **Production Usage** | Very common | Very common |
| **Startup Time** | ~500ms | < 100ms |

---

## Migration Rationale

### Why We Migrated from Node.js to Golang

1. **Performance:** Golang offers 10-40x better performance for CPU-intensive operations
2. **Deployment Simplicity:** Single binary vs Node runtime + node_modules
3. **Resource Efficiency:** Lower memory footprint means lower hosting costs
4. **Concurrency:** Goroutines provide better concurrency than event loop
5. **Type Safety:** Go's static typing catches more errors at compile time
6. **Build Speed:** Sub-second builds vs 5-10 second TypeScript compilation
7. **Production Readiness:** Faster to production with less complexity

### What We Kept

- **PostgreSQL:** Database choice unchanged (GORM works great with Postgres)
- **JWT Authentication:** Same authentication approach
- **REST API Design:** API endpoints and structure unchanged
- **Docker Deployment:** Same containerization strategy
- **AWS Infrastructure:** Cloud provider choice unchanged

### Migration Results

- **Development Speed:** Faster iteration with hot reload and fast builds
- **Code Quality:** Strong typing reduces bugs
- **Deployment:** Simpler with single binary
- **Resource Usage:** Lower hosting costs
- **Performance:** Better response times

---

## Summary

| Category | Technology | Reason |
|----------|-----------|--------|
| **Language** | Go 1.21+ | Performance, concurrency, simple deployment |
| **Framework** | Gin | Fastest Go framework, production-ready |
| **Database** | PostgreSQL 15 | ACID + JSON support |
| **ORM** | GORM | Feature-rich, intuitive API |
| **Auth** | JWT (golang-jwt) | Stateless, scalable |
| **Password** | Bcrypt | Industry standard |
| **Testing** | Go testing | Built-in, fast |
| **Deployment** | Docker | Portable, consistent |
| **CI/CD** | GitHub Actions | Integrated, free |
| **Cloud** | AWS | Comprehensive services |
| **Monitoring** | Prometheus + Grafana | Open source, powerful |

**Total Monthly Cost (Year 1):** ~$90-110/month infrastructure

---

**Document Version:** 2.0
**Last Updated:** 2025-10-17
**Status:** Updated for Golang Implementation

This tech stack provides superior performance, simpler deployment, and excellent developer experience compared to the original Node.js plan.
