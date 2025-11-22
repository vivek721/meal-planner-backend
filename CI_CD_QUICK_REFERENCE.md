# CI/CD Quick Reference

## Quick Commands

```bash
# Run all tests like CI does
make ci-test

# Run tests with race detector
make test-race

# Run security scan
make security-scan

# Run linter
make lint

# Format code
make fmt

# Check code
make vet
```

## Workflow Triggers

| Workflow | Triggers |
|----------|----------|
| **CI Pipeline** | Push to master/main/develop, Pull Requests |
| **CodeQL** | Push, PR, Weekly (Monday 00:00 UTC) |
| **Release** | Version tags (v1.0.0, v2.1.3, etc.) |
| **PR Checks** | Pull Requests only |

## Creating a Release

```bash
# Tag the release
git tag -a v1.0.0 -m "Release version 1.0.0"

# Push the tag
git push origin v1.0.0

# GitHub Actions automatically:
# - Builds binaries for all platforms
# - Creates GitHub Release with changelog
# - Builds and pushes Docker images
```

## CI Jobs Flow

```
CI Pipeline:
├── Lint (gofmt, go vet, golangci-lint)
├── Test - Go 1.21 (tests + coverage)
├── Test - Go 1.22 (tests only)
├── Security (Gosec, govulncheck)
├── Build - linux/amd64
├── Build - linux/arm64
├── Docker (build test)
├── Integration (if tests tagged)
└── All Checks Pass ✓
```

## Coverage Requirements

- **Minimum:** 70%
- **Enforced by:** CI pipeline
- **Check locally:**
  ```bash
  make test-coverage
  # or
  make ci-test
  ```

## Environment Variables (CI)

```bash
DATABASE_URL=postgresql://postgres:postgres@localhost:5432/meal_planner_test?sslmode=disable
JWT_SECRET=test-secret-key-for-ci-minimum-32-characters-required
ENVIRONMENT=test
PORT=3001
BCRYPT_COST=4
FRONTEND_URL=http://localhost:3000
```

## Local Testing

### With Docker (Recommended)

```bash
# Start test database
docker run -d --name postgres-test \
  -e POSTGRES_USER=postgres \
  -e POSTGRES_PASSWORD=postgres \
  -e POSTGRES_DB=meal_planner_test \
  -p 5432:5432 \
  postgres:14-alpine

# Set environment variables
export DATABASE_URL="postgresql://postgres:postgres@localhost:5432/meal_planner_test?sslmode=disable"
export JWT_SECRET="test-secret-key-for-ci-minimum-32-characters-required"
export ENVIRONMENT="test"
export BCRYPT_COST="4"

# Run CI tests
make ci-test

# Clean up
docker stop postgres-test && docker rm postgres-test
```

### Using Act (GitHub Actions locally)

```bash
# Install act
brew install act  # macOS
choco install act-cli  # Windows

# Run CI workflow
act push

# Run specific job
act -j lint
act -j test
```

## Viewing Results

### GitHub Actions
1. Go to repository
2. Click "Actions" tab
3. Select workflow run
4. View job logs

### Security Findings
1. Go to repository
2. Click "Security" tab
3. Click "Code scanning alerts"
4. View CodeQL and Gosec results

### Coverage Reports
1. Codecov: https://codecov.io/gh/vivek721/meal-planner-backend
2. Or check CI job "Test (1.21)" artifacts

## Common Issues

### Tests fail in CI but pass locally
```bash
# Run with race detector (CI uses this)
make test-race

# Use low bcrypt cost like CI
export BCRYPT_COST=4
make test
```

### Coverage below threshold
```bash
# Check which packages need tests
go test -cover ./...

# Focus on:
# - handlers/ - HTTP handlers
# - services/ - Business logic
# - repository/ - Data access
```

### Linter errors
```bash
# Auto-fix formatting
make fmt

# Run linter locally
make lint

# Fix specific issues
golangci-lint run --fix
```

### Docker build fails
```bash
# Test build locally
docker build -t test .

# See full output
docker build --progress=plain -t test .
```

## PR Best Practices

- **Size:** Keep PRs small (<500 lines)
- **Tests:** Add tests for new features
- **Commit messages:** Use conventional format
  ```
  feat(scope): description
  fix(scope): description
  docs: description
  refactor: description
  test: description
  chore: description
  ```
- **Coverage:** Don't decrease coverage
- **Linting:** Fix all lint warnings

## Branch Protection (Recommended)

Enable on master branch:
- ✓ Require PR reviews (1 approval)
- ✓ Require status checks:
  - Lint
  - Test (1.21)
  - Test (1.22)
  - Security Scan
  - Build
  - All Checks Passed
- ✓ Require conversation resolution
- ✓ Require linear history

## Useful Links

- **Repository:** https://github.com/vivek721/meal-planner-backend
- **Actions:** https://github.com/vivek721/meal-planner-backend/actions
- **Releases:** https://github.com/vivek721/meal-planner-backend/releases
- **Security:** https://github.com/vivek721/meal-planner-backend/security
- **Full Guide:** CI_CD_GUIDE.md

## Makefile Commands

```bash
make help              # Show all commands
make install           # Install dependencies
make build             # Build binary
make run               # Run server
make dev               # Run with hot reload
make test              # Run tests
make test-coverage     # Tests with coverage report
make test-race         # Tests with race detector
make ci-test           # CI tests with threshold
make security-scan     # Security scanning
make clean             # Clean artifacts
make fmt               # Format code
make lint              # Run linter
make vet               # Run go vet
```

## Files Structure

```
.github/
├── workflows/
│   ├── ci.yml          # Main CI pipeline
│   ├── codeql.yml      # Security analysis
│   ├── pr.yml          # PR automation
│   └── release.yml     # Release automation
└── dependabot.yml      # Dependency updates

.golangci.yml           # Linter config
CI_CD_GUIDE.md          # Full documentation
Makefile                # Build commands
```

## Monitoring

### Watch for:
- ⚠️ Failed CI runs
- ⚠️ Security alerts
- ⚠️ Dependabot PRs
- ⚠️ Coverage drops
- ⚠️ Large PRs (>500 lines)

### Review weekly:
- Security scan results
- Dependabot updates
- Code coverage trends
- Build performance

---

**Quick help:** `make help`
**Full docs:** `CI_CD_GUIDE.md`
**Issues:** Check GitHub Actions tab
