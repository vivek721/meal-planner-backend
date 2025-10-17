# Security Guidelines
## Backend Security Best Practices

---

**Version:** 1.0
**Last Updated:** 2025-10-14
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
```typescript
import bcrypt from 'bcryptjs';

const SALT_ROUNDS = 12; // 2^12 iterations (~250ms)

async function hashPassword(password: string): Promise<string> {
  return bcrypt.hash(password, SALT_ROUNDS);
}

async function verifyPassword(password: string, hash: string): Promise<boolean> {
  return bcrypt.compare(password, hash);
}
```

**Best Practices:**
- Never log passwords (plain-text or hashed)
- Increase bcrypt rounds as hardware improves (every 2-3 years)
- Consider Argon2 for new projects (more secure than bcrypt)

---

### JWT Token Security

**Token Configuration:**
```typescript
import jwt from 'jsonwebtoken';

const ACCESS_TOKEN_EXPIRY = '1h';
const REFRESH_TOKEN_EXPIRY = '7d';
const JWT_SECRET = process.env.JWT_SECRET; // 256-bit random, never in code

function generateAccessToken(payload: TokenPayload): string {
  return jwt.sign(payload, JWT_SECRET, {
    expiresIn: ACCESS_TOKEN_EXPIRY,
    algorithm: 'HS256',
  });
}
```

**Security Measures:**
- Secret stored in AWS Secrets Manager
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

**Express Configuration:**
```typescript
import helmet from 'helmet';

app.use(helmet({
  hsts: {
    maxAge: 31536000,
    includeSubDomains: true,
    preload: true,
  },
  contentSecurityPolicy: {
    directives: {
      defaultSrc: ["'self'"],
      styleSrc: ["'self'", "'unsafe-inline'"],
      scriptSrc: ["'self'"],
      imgSrc: ["'self'", 'https://cdn.mealplanner.com'],
    },
  },
}));
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
```typescript
// ❌ NEVER DO THIS
const query = `SELECT * FROM users WHERE email = '${email}'`;

// ✅ ALWAYS DO THIS (Prisma)
const user = await prisma.user.findUnique({
  where: { email },
});

// ✅ OR THIS (Raw SQL with parameters)
const user = await db.query('SELECT * FROM users WHERE email = $1', [email]);
```

---

### Prevent XSS (Cross-Site Scripting)

**Sanitize all user inputs:**
```typescript
import { escape } from 'validator';

function sanitizeInput(input: string): string {
  return escape(input.trim());
}

// Or use Zod for automatic validation
const schema = z.object({
  name: z.string().min(2).max(255),
  email: z.string().email(),
});
```

**Content Security Policy:**
- Set CSP headers (via Helmet.js)
- Disable inline scripts
- Whitelist trusted domains

---

### Prevent CSRF (Cross-Site Request Forgery)

**For state-changing operations:**
```typescript
import csrf from 'csurf';

const csrfProtection = csrf({ cookie: true });

app.post('/api/v1/recipes', csrfProtection, (req, res) => {
  // CSRF token validated automatically
});
```

**Alternative for JWT:**
- SameSite cookie attribute: `SameSite=Strict`
- Double-submit cookie pattern

---

## Rate Limiting

### API Rate Limits

```typescript
import rateLimit from 'express-rate-limit';

const generalLimiter = rateLimit({
  windowMs: 60 * 1000, // 1 minute
  max: 100, // 100 requests per minute per IP
  message: { error: { code: 'RATE_LIMIT_EXCEEDED' } },
  standardHeaders: true,
  legacyHeaders: false,
});

const authLimiter = rateLimit({
  windowMs: 15 * 60 * 1000, // 15 minutes
  max: 5, // 5 login attempts
  skipSuccessfulRequests: true,
});

app.use('/api/v1', generalLimiter);
app.post('/api/v1/auth/login', authLimiter, authController.login);
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
```typescript
import { SecretsManager } from '@aws-sdk/client-secrets-manager';

async function getSecret(secretName: string): Promise<string> {
  const client = new SecretsManager({ region: 'us-east-1' });

  const response = await client.getSecretValue({ SecretId: secretName });

  return response.SecretString;
}

// Use in startup
const jwtSecret = await getSecret('production/jwt/secret');
process.env.JWT_SECRET = jwtSecret;
```

**Rotation:**
- Database password: Automated quarterly (RDS rotation)
- JWT secret: Manual yearly (requires all users to re-login)
- API keys: Quarterly

---

## Security Headers

### Helmet.js Configuration

```typescript
app.use(helmet({
  contentSecurityPolicy: {
    directives: {
      defaultSrc: ["'self'"],
      scriptSrc: ["'self'"],
      styleSrc: ["'self'", "'unsafe-inline'"],
      imgSrc: ["'self'", 'https://cdn.mealplanner.com'],
    },
  },
  hsts: {
    maxAge: 31536000,
    includeSubDomains: true,
  },
  frameguard: { action: 'deny' },
  noSniff: true,
  xssFilter: true,
  referrerPolicy: { policy: 'no-referrer' },
}));
```

---

## CORS Configuration

```typescript
import cors from 'cors';

const allowedOrigins = [
  'https://app.mealplanner.com',
  'http://localhost:5173', // Dev only
];

app.use(cors({
  origin: (origin, callback) => {
    if (!origin || allowedOrigins.includes(origin)) {
      callback(null, true);
    } else {
      callback(new Error('Not allowed by CORS'));
    }
  },
  credentials: true,
  methods: ['GET', 'POST', 'PUT', 'PATCH', 'DELETE'],
  allowedHeaders: ['Content-Type', 'Authorization'],
}));
```

---

## GDPR Compliance

### User Rights

**Right to Access:**
```typescript
// GET /users/:id/export
async exportUserData(userId: string): Promise<object> {
  return {
    user: await prisma.user.findUnique({ where: { id: userId } }),
    mealPlans: await prisma.mealPlan.findMany({ where: { userId } }),
    favorites: await prisma.favorite.findMany({ where: { userId } }),
    activities: await prisma.activity.findMany({ where: { userId } }),
  };
}
```

**Right to Erasure:**
```typescript
// DELETE /users/:id (soft delete)
async deleteUser(userId: string): Promise<void> {
  await prisma.user.update({
    where: { id: userId },
    data: { deletedAt: new Date() },
  });

  // Anonymize data after 30 days (cron job)
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
npm audit --audit-level=moderate

# Fix automatically
npm audit fix

# Use Snyk for continuous monitoring
npx snyk test
npx snyk monitor
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

- [ ] All inputs validated (Zod schemas)
- [ ] Parameterized queries (no string concatenation)
- [ ] Passwords hashed with bcrypt (12+ rounds)
- [ ] No secrets in code (environment variables)
- [ ] Error messages don't leak sensitive info

### Deployment

- [ ] HTTPS only (TLS 1.3)
- [ ] Security headers (Helmet.js)
- [ ] CORS properly configured
- [ ] Rate limiting enabled
- [ ] Secrets in AWS Secrets Manager

### Operations

- [ ] npm audit shows zero high/critical
- [ ] Security updates applied within SLA
- [ ] Backups tested quarterly
- [ ] Incident response plan documented
- [ ] GDPR compliance verified

---

**Document Version:** 1.0
**Last Updated:** 2025-10-14
**Next Review:** 2026-01-14 (Quarterly)
