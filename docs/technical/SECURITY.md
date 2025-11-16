# Security Guidelines
## Backend Security Best Practices

---

> **Note**: This document has been updated for the Golang implementation.
> The backend was migrated from Node.js/TypeScript to Golang on October 16, 2025.

**Version:** 2.0
**Last Updated:** 2025-10-17
**Compliance:** OWASP Top 10, GDPR

---

## Authentication & Authorization

### Password Security

**Requirements:**
- Minimum 8 characters
- At least 1 uppercase letter
- At least 1 number
- At least 1 special character

**Hashing:**
```go
import "golang.org/x/crypto/bcrypt"

const SALT_ROUNDS = 12 // Cost factor: 2^12 iterations (~250ms)

func HashPassword(password string) (string, error) {
    bytes, err := bcrypt.GenerateFromPassword([]byte(password), SALT_ROUNDS)
    return string(bytes), err
}

func VerifyPassword(password, hash string) bool {
    err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
    return err == nil
}
```

**Best Practices:**
- Never log passwords (plain-text or hashed)
- Increase bcrypt rounds as hardware improves (every 2-3 years)
- Consider Argon2 for new projects (more secure than bcrypt)

---

### JWT Token Security

**Token Configuration:**
```go
import (
    "github.com/golang-jwt/jwt/v5"
    "time"
)

const (
    AccessTokenExpiry  = time.Hour        // 1 hour
    RefreshTokenExpiry = 7 * 24 * time.Hour // 7 days
)

type TokenClaims struct {
    UserID string `json:"userId"`
    Email  string `json:"email"`
    Role   string `json:"role"`
    jwt.RegisteredClaims
}

func GenerateAccessToken(payload TokenClaims) (string, error) {
    payload.ExpiresAt = jwt.NewNumericDate(time.Now().Add(AccessTokenExpiry))
    payload.IssuedAt = jwt.NewNumericDate(time.Now())

    token := jwt.NewWithClaims(jwt.SigningMethodHS256, payload)
    return token.SignedString([]byte(os.Getenv("JWT_SECRET")))
}

func ValidateToken(tokenString string) (*TokenClaims, error) {
    token, err := jwt.ParseWithClaims(tokenString, &TokenClaims{}, func(token *jwt.Token) (interface{}, error) {
        if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
            return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
        }
        return []byte(os.Getenv("JWT_SECRET")), nil
    })

    if err != nil {
        return nil, err
    }

    if claims, ok := token.Claims.(*TokenClaims); ok && token.Valid {
        return claims, nil
    }

    return nil, fmt.Errorf("invalid token")
}
```

**Security Measures:**
- Secret stored in AWS Secrets Manager or environment variables
- Rotate secret quarterly
- Use HTTPS only (no HTTP)
- Token blacklist on logout (Redis)
- Validate token signature and expiry on every request

---

## Data Encryption

### In Transit (HTTPS/TLS)

**Requirements:**
- TLS 1.3 only (disable TLS 1.2 and below)
- Strong cipher suites (AES-256-GCM)
- HSTS enabled (max-age: 31536000, includeSubDomains)
- SSL/TLS certificate auto-renewed (AWS Certificate Manager)

**Gin Security Middleware:**
```go
import "github.com/gin-contrib/secure"

func SecurityMiddleware() gin.HandlerFunc {
    return secure.New(secure.Config{
        STSSeconds:            31536000,
        STSIncludeSubdomains:  true,
        FrameDeny:             true,
        ContentTypeNosniff:    true,
        BrowserXssFilter:      true,
        ReferrerPolicy:        "no-referrer",
        ContentSecurityPolicy: "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' https://cdn.mealplanner.com",
    })
}

// Usage
r := gin.Default()
r.Use(SecurityMiddleware())
```

---

### At Rest

**Database (RDS):**
- AES-256 encryption enabled
- Encrypted backups
- Encryption keys managed by AWS KMS

**S3 Images:**
- Server-side encryption (SSE-S3 or SSE-KMS)
- Bucket policy enforces encryption

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "DenyUnencryptedObjectUploads",
      "Effect": "Deny",
      "Principal": "*",
      "Action": "s3:PutObject",
      "Resource": "arn:aws:s3:::meal-planner-images-production/*",
      "Condition": {
        "StringNotEquals": {
          "s3:x-amz-server-side-encryption": "AES256"
        }
      }
    }
  ]
}
```

---

## Input Validation & Sanitization

### Prevent SQL Injection

**Always use parameterized queries:**
```go
// ❌ NEVER DO THIS
query := fmt.Sprintf("SELECT * FROM users WHERE email = '%s'", email)

// ✅ ALWAYS DO THIS (GORM)
var user models.User
db.Where("email = ?", email).First(&user)

// ✅ OR THIS (Raw SQL with parameters)
db.Raw("SELECT * FROM users WHERE email = ?", email).Scan(&user)
```

**GORM prevents SQL injection automatically** through parameterized queries.

---

### Prevent XSS (Cross-Site Scripting)

**Sanitize all user inputs:**
```go
import (
    "github.com/go-playground/validator/v10"
    "html"
)

func SanitizeInput(input string) string {
    return html.EscapeString(strings.TrimSpace(input))
}

// Use validator for automatic validation
type RegisterRequest struct {
    Name     string `json:"name" binding:"required,min=2,max=255"`
    Email    string `json:"email" binding:"required,email"`
    Password string `json:"password" binding:"required,min=8,password"`
}

func (h *AuthHandler) Register(c *gin.Context) {
    var req RegisterRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
        return
    }

    // Input is validated and sanitized
    // ...
}
```

**Content Security Policy:**
- Set CSP headers (via secure middleware)
- Disable inline scripts
- Whitelist trusted domains

---

### Prevent CSRF (Cross-Site Request Forgery)

**For state-changing operations:**
```go
import "github.com/gin-contrib/csrf"

// CSRF middleware
r.Use(csrf.Middleware(csrf.Options{
    Secret: os.Getenv("CSRF_SECRET"),
    ErrorFunc: func(c *gin.Context) {
        c.JSON(http.StatusForbidden, gin.H{"error": "CSRF token invalid"})
        c.Abort()
    },
}))

// Protected routes
r.POST("/api/v1/recipes", csrfMiddleware, recipeHandler.Create)
```

**Alternative for JWT:**
- SameSite cookie attribute: `SameSite=Strict`
- Double-submit cookie pattern
- For API-only (no browser cookies), CSRF is less of a concern

---

## Rate Limiting

### API Rate Limits

```go
import (
    "github.com/ulule/limiter/v3"
    "github.com/ulule/limiter/v3/drivers/middleware/gin"
    "github.com/ulule/limiter/v3/drivers/store/memory"
)

// General rate limiter
func RateLimitMiddleware() gin.HandlerFunc {
    rate := limiter.Rate{
        Period: 1 * time.Minute,
        Limit:  100, // 100 requests per minute per IP
    }
    store := memory.NewStore()
    instance := limiter.New(store, rate)
    return ginlimiter.NewMiddleware(instance)
}

// Auth-specific rate limiter
func AuthRateLimitMiddleware() gin.HandlerFunc {
    rate := limiter.Rate{
        Period: 15 * time.Minute,
        Limit:  5, // 5 login attempts per 15 minutes
    }
    store := memory.NewStore()
    instance := limiter.New(store, rate)
    return ginlimiter.NewMiddleware(instance)
}

// Usage
r.Use(RateLimitMiddleware())
r.POST("/api/v1/auth/login", AuthRateLimitMiddleware(), authHandler.Login)
```

**Redis-backed rate limiting** (for distributed systems):
```go
import "github.com/ulule/limiter/v3/drivers/store/redis"

store, err := redis.NewStoreWithOptions(redisClient, limiter.StoreOptions{
    Prefix:   "limiter",
})

instance := limiter.New(store, rate)
```

---

## Secrets Management

### AWS Secrets Manager

**Store:**
- Database credentials
- JWT secret
- SendGrid API key
- Third-party API keys

**Retrieve at Runtime:**
```go
import (
    "context"
    "github.com/aws/aws-sdk-go-v2/config"
    "github.com/aws/aws-sdk-go-v2/service/secretsmanager"
)

func GetSecret(ctx context.Context, secretName string) (string, error) {
    cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion("us-east-1"))
    if err != nil {
        return "", err
    }

    client := secretsmanager.NewFromConfig(cfg)

    result, err := client.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{
        SecretId: &secretName,
    })

    if err != nil {
        return "", err
    }

    return *result.SecretString, nil
}

// Use in startup
func LoadSecrets(ctx context.Context) error {
    jwtSecret, err := GetSecret(ctx, "production/jwt/secret")
    if err != nil {
        return err
    }
    os.Setenv("JWT_SECRET", jwtSecret)

    dbPassword, err := GetSecret(ctx, "production/db/password")
    if err != nil {
        return err
    }
    os.Setenv("DB_PASSWORD", dbPassword)

    return nil
}
```

**Rotation:**
- Database password: Automated quarterly (RDS rotation)
- JWT secret: Manual yearly (requires all users to re-login)
- API keys: Quarterly

---

## Security Headers

### Gin Security Configuration

```go
import "github.com/gin-contrib/secure"

r.Use(secure.New(secure.Config{
    // HSTS
    STSSeconds:           31536000,
    STSIncludeSubdomains: true,
    STSPreload:           true,

    // Content Security Policy
    ContentSecurityPolicy: "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' https://cdn.mealplanner.com",

    // Other headers
    FrameDeny:             true,
    ContentTypeNosniff:    true,
    BrowserXssFilter:      true,
    ReferrerPolicy:        "no-referrer",
    FeaturePolicy:         "geolocation 'none'",
}))
```

---

## CORS Configuration

```go
import "github.com/gin-contrib/cors"

func CORSMiddleware() gin.HandlerFunc {
    return cors.New(cors.Config{
        AllowOrigins:     []string{"https://app.mealplanner.com", "http://localhost:3000"},
        AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
        AllowHeaders:     []string{"Content-Type", "Authorization"},
        ExposeHeaders:    []string{"Content-Length"},
        AllowCredentials: true,
        MaxAge:           12 * time.Hour,
    })
}

// Usage
r.Use(CORSMiddleware())
```

**Production configuration:**
```go
func CORSMiddleware() gin.HandlerFunc {
    allowedOrigins := strings.Split(os.Getenv("ALLOWED_ORIGINS"), ",")

    return cors.New(cors.Config{
        AllowOriginFunc: func(origin string) bool {
            for _, allowed := range allowedOrigins {
                if origin == allowed {
                    return true
                }
            }
            return false
        },
        AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE"},
        AllowHeaders:     []string{"Content-Type", "Authorization"},
        AllowCredentials: true,
        MaxAge:           12 * time.Hour,
    })
}
```

---

## GDPR Compliance

### User Rights

**Right to Access:**
```go
// GET /users/:id/export
func (s *UserService) ExportUserData(ctx context.Context, userID string) (map[string]interface{}, error) {
    var user models.User
    if err := s.db.First(&user, "id = ?", userID).Error; err != nil {
        return nil, err
    }

    var mealPlans []models.MealPlan
    s.db.Find(&mealPlans, "user_id = ?", userID)

    var favorites []models.Favorite
    s.db.Find(&favorites, "user_id = ?", userID)

    var activities []models.Activity
    s.db.Find(&activities, "user_id = ?", userID)

    return map[string]interface{}{
        "user":       user,
        "mealPlans":  mealPlans,
        "favorites":  favorites,
        "activities": activities,
        "exportedAt": time.Now(),
    }, nil
}
```

**Right to Erasure:**
```go
// DELETE /users/:id (soft delete)
func (s *UserService) DeleteUser(ctx context.Context, userID string) error {
    // Soft delete (sets deleted_at timestamp)
    if err := s.db.Delete(&models.User{}, "id = ?", userID).Error; err != nil {
        return err
    }

    // Schedule anonymization after 30 days (handled by cron job)
    return nil
}

// Anonymization (run by cron after 30 days)
func (s *UserService) AnonymizeDeletedUsers(ctx context.Context) error {
    thirtyDaysAgo := time.Now().AddDate(0, 0, -30)

    return s.db.Model(&models.User{}).
        Where("deleted_at < ?", thirtyDaysAgo).
        Updates(map[string]interface{}{
            "email":    "deleted@deleted.com",
            "name":     "Deleted User",
            "password": "",
        }).Error
}
```

**Data Retention:**
- Active user data: Indefinite
- Deleted user data: 30 days (soft delete)
- Logs: 90 days
- Backups: 30 days

---

## Vulnerability Management

### Dependency Scanning

```bash
# Run weekly in CI/CD
go list -m all  # List all dependencies

# Vulnerability scanning with govulncheck
go install golang.org/x/vuln/cmd/govulncheck@latest
govulncheck ./...

# Security scanning with gosec
go install github.com/securego/gosec/v2/cmd/gosec@latest
gosec ./...

# Dependency updates
go get -u all
go mod tidy
```

**Alert Thresholds:**
- Critical: Fix within 24 hours
- High: Fix within 7 days
- Medium: Fix within 30 days
- Low: Fix in next release

---

### Penetration Testing

**Frequency:**
- Automated (OWASP ZAP): Weekly
- Manual: Quarterly
- Third-party audit: Annually

**OWASP ZAP Scan:**
```bash
docker run -t owasp/zap2docker-stable zap-baseline.py \
  -t https://api.mealplanner.com \
  -r zap-report.html
```

---

## Incident Response

### Security Incident Procedure

1. **Detect** (< 15 minutes)
   - Automated alerts (Sentry, CloudWatch)
   - Manual report (bug bounty, user report)

2. **Contain** (< 1 hour)
   - Isolate affected systems
   - Revoke compromised credentials
   - Block malicious IPs

3. **Eradicate** (< 4 hours)
   - Patch vulnerability
   - Deploy fix to production
   - Verify fix with testing

4. **Recover** (< 24 hours)
   - Restore from backup if needed
   - Monitor for recurrence
   - Notify affected users

5. **Post-Mortem** (< 7 days)
   - Root cause analysis
   - Update security procedures
   - Improve monitoring/alerts

**GDPR Breach Notification:**
- Notify authorities within 72 hours
- Notify affected users if high risk
- Document incident in breach register

---

## Security Checklist

### Development

- [ ] All inputs validated (go-playground/validator)
- [ ] Parameterized queries (GORM or ? placeholders)
- [ ] Passwords hashed with bcrypt (12+ rounds)
- [ ] No secrets in code (environment variables)
- [ ] Error messages don't leak sensitive info

### Deployment

- [ ] HTTPS only (TLS 1.3)
- [ ] Security headers (secure middleware)
- [ ] CORS properly configured
- [ ] Rate limiting enabled
- [ ] Secrets in AWS Secrets Manager

### Operations

- [ ] govulncheck shows zero high/critical
- [ ] Security updates applied within SLA
- [ ] Backups tested quarterly
- [ ] Incident response plan documented
- [ ] GDPR compliance verified

---

## Go-Specific Security Best Practices

### Memory Safety
- Go prevents buffer overflows and memory corruption
- Use `defer` for cleanup to prevent resource leaks
- No manual memory management

### Concurrency Safety
- Use mutexes for shared state: `sync.Mutex`, `sync.RWMutex`
- Use channels for communication between goroutines
- Avoid race conditions (test with `go test -race`)

### Error Handling
- Always check errors: `if err != nil`
- Don't expose internal errors to users
- Log detailed errors, return generic messages

### Type Safety
- Strong static typing prevents many bugs
- Use `interface{}` sparingly
- Leverage compiler checks

---

**Document Version:** 2.0
**Last Updated:** 2025-10-17
**Next Review:** 2026-01-17 (Quarterly)
