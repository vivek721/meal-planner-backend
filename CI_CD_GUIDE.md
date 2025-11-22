# CI/CD Guide - Meal Planner Backend

This document explains the Continuous Integration and Continuous Deployment (CI/CD) setup for the Meal Planner Backend API.

## Overview

The CI/CD pipeline is implemented using GitHub Actions and includes:

- **Continuous Integration (CI)**: Automated testing, linting, security scanning, and building
- **Continuous Deployment (CD)**: Automated releases and Docker image publishing
- **Code Quality**: Automated code analysis and security scanning
- **Dependency Management**: Automated dependency updates via Dependabot

## Workflows

### 1. CI Pipeline (`.github/workflows/ci.yml`)

Runs on every push to `master`, `main`, or `develop` branches and on all pull requests.

#### Jobs:

**Lint**
- Verifies Go module dependencies
- Runs `gofmt` to check code formatting
- Runs `go vet` for static analysis
- Runs `golangci-lint` with comprehensive linter configuration

**Test (Matrix: Go 1.21, 1.22)**
- Runs unit tests with race detector
- Generates code coverage reports
- Enforces 70% minimum code coverage
- Uses PostgreSQL 14 service container for tests
- Uploads coverage to Codecov

**Security**
- Runs Gosec security scanner
- Performs vulnerability scanning with govulncheck
- Uploads results to GitHub Security tab

**Build (Matrix: linux/amd64, linux/arm64)**
- Builds binaries for multiple platforms
- Creates versioned artifacts
- Uploads build artifacts (retained for 7 days)

**Docker**
- Tests Docker image build process
- Uses build cache for faster builds
- Validates image can be created successfully

**Integration**
- Runs integration tests (if tagged)
- Uses PostgreSQL service container
- Extended timeout for longer-running tests

**All Checks Pass**
- Final job that validates all previous jobs succeeded
- Required for PR merges

#### Environment Variables Used:
```bash
DATABASE_URL=postgresql://postgres:postgres@localhost:5432/meal_planner_test?sslmode=disable
JWT_SECRET=test-secret-key-for-ci-minimum-32-characters-required
ENVIRONMENT=test
PORT=3001
BCRYPT_COST=4  # Low cost for faster tests
FRONTEND_URL=http://localhost:3000
```

### 2. CodeQL Security Analysis (`.github/workflows/codeql.yml`)

Runs on:
- Push to `master`, `main`, `develop` branches
- Pull requests to these branches
- Weekly schedule (Monday at 00:00 UTC)

#### Features:
- Static security analysis for Go code
- Runs extended security and quality queries
- Results appear in GitHub Security tab
- Helps identify potential vulnerabilities

### 3. Release Workflow (`.github/workflows/release.yml`)

Triggers on version tags (e.g., `v1.0.0`, `v2.1.3`)

#### Jobs:

**Create Release**
- Generates changelog from commits
- Builds multi-platform binaries:
  - Linux (amd64, arm64)
  - macOS (amd64, arm64)
  - Windows (amd64)
- Creates GitHub Release with:
  - Version number
  - Changelog
  - Downloadable binaries
  - Docker pull instructions
- Marks pre-releases (alpha, beta, rc)

**Build and Push Docker Image**
- Builds Docker images for linux/amd64 and linux/arm64
- Pushes to GitHub Container Registry (ghcr.io)
- Tags:
  - Semantic version (e.g., `v1.0.0`)
  - Major.minor (e.g., `1.0`)
  - Major (e.g., `1`)
  - Git SHA
- Uses build cache for efficiency

## Dependabot (`.github/dependabot.yml`)

Automated dependency updates for:

**Go Modules**
- Weekly updates (Mondays at 09:00)
- Groups minor and patch updates
- Max 10 pull requests

**GitHub Actions**
- Weekly updates (Mondays at 09:00)
- Max 5 pull requests

**Docker Base Images**
- Weekly updates (Mondays at 09:00)
- Max 5 pull requests

All updates are:
- Labeled for easy filtering
- Auto-assigned to repository owner
- Include commit message prefixes (`chore:`)

## Linter Configuration (`.golangci.yml`)

Comprehensive linter configuration with 30+ enabled linters:

**Enabled Linters:**
- `errcheck` - Unchecked errors
- `gosimple` - Code simplification
- `govet` - Static analysis
- `staticcheck` - Advanced static analysis
- `gosec` - Security issues
- `gofmt` / `gofumpt` - Code formatting
- `goimports` - Import organization
- `misspell` - Spelling errors
- `revive` - Fast linter with rules
- `gocritic` - Comprehensive checks
- `stylecheck` - Style consistency
- And 20+ more...

**Key Settings:**
- Cyclomatic complexity limit: 15
- Minimum coverage threshold: 70%
- Test files have relaxed rules
- Custom exclusions for common patterns

## Makefile Commands

New CI/CD related commands:

```bash
# Run tests with race detector
make test-race

# Run tests with coverage threshold check (like CI)
make ci-test

# Run security scan with gosec
make security-scan

# Enhanced test coverage (with atomic mode)
make test-coverage
```

## Badge Integration

Add these badges to your README.md:

```markdown
[![CI Pipeline](https://github.com/vivek721/meal-planner-backend/actions/workflows/ci.yml/badge.svg)](https://github.com/vivek721/meal-planner-backend/actions/workflows/ci.yml)
[![CodeQL](https://github.com/vivek721/meal-planner-backend/actions/workflows/codeql.yml/badge.svg)](https://github.com/vivek721/meal-planner-backend/actions/workflows/codeql.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/vivek721/meal-planner-backend)](https://goreportcard.com/report/github.com/vivek721/meal-planner-backend)
[![codecov](https://codecov.io/gh/vivek721/meal-planner-backend/branch/master/graph/badge.svg)](https://codecov.io/gh/vivek721/meal-planner-backend)
```

## Testing Workflows Locally

### Option 1: Using Act (GitHub Actions Local Runner)

Install Act:
```bash
# macOS
brew install act

# Linux
curl https://raw.githubusercontent.com/nektos/act/master/install.sh | sudo bash

# Windows
choco install act-cli
```

Run workflows:
```bash
# Run CI workflow
act push

# Run specific job
act -j lint

# Run with different event
act pull_request
```

### Option 2: Manual Local Testing

Simulate CI environment:

```bash
# Set environment variables
export DATABASE_URL="postgresql://postgres:postgres@localhost:5432/meal_planner_test?sslmode=disable"
export JWT_SECRET="test-secret-key-for-ci-minimum-32-characters-required"
export ENVIRONMENT="test"
export PORT="3001"
export BCRYPT_COST="4"
export FRONTEND_URL="http://localhost:3000"

# Start PostgreSQL (Docker)
docker run -d --name postgres-test \
  -e POSTGRES_USER=postgres \
  -e POSTGRES_PASSWORD=postgres \
  -e POSTGRES_DB=meal_planner_test \
  -p 5432:5432 \
  postgres:14-alpine

# Run CI tests
make ci-test

# Run linting
make lint

# Run security scan
make security-scan

# Clean up
docker stop postgres-test && docker rm postgres-test
```

## Workflow Triggers

### CI Pipeline Triggers:
```yaml
on:
  push:
    branches: [ master, main, develop ]
  pull_request:
    branches: [ master, main, develop ]
```

### Release Workflow Triggers:
```bash
# Create and push a version tag
git tag -a v1.0.0 -m "Release version 1.0.0"
git push origin v1.0.0
```

### CodeQL Triggers:
- Automatic on push/PR
- Weekly schedule
- Manual dispatch (from Actions tab)

## Viewing Results in GitHub

### CI Pipeline Results
1. Go to repository → Actions tab
2. Click on "CI Pipeline" workflow
3. View all runs and their status
4. Click on a run to see detailed logs for each job

### Security Results
1. Go to repository → Security tab
2. Click "Code scanning alerts"
3. View CodeQL and Gosec findings
4. Filter by severity, status, or tool

### Coverage Reports
1. CI uploads coverage to Codecov
2. View at: https://codecov.io/gh/vivek721/meal-planner-backend
3. See coverage trends, file coverage, and PR impact

### Release Artifacts
1. Go to repository → Releases
2. Each release contains:
   - Changelog
   - Binary downloads (all platforms)
   - Docker image pull commands
   - Release notes

## Common Issues and Solutions

### Issue: Tests fail in CI but pass locally

**Possible causes:**
- Race conditions (CI runs with `-race` flag)
- Different Go versions
- Missing environment variables
- Database state differences

**Solution:**
```bash
# Run tests with race detector locally
make test-race

# Use same environment variables as CI
source .env.test  # Create this file with CI env vars
```

### Issue: Coverage below threshold

**Solution:**
```bash
# Check current coverage
make test-coverage

# Add tests to increase coverage
# Focus on:
# - Handler tests
# - Service tests
# - Repository tests
```

### Issue: Linter failures

**Solution:**
```bash
# Run linter locally with same config
make lint

# Auto-fix formatting issues
make fmt

# Check specific linter
golangci-lint run --enable-only=errcheck
```

### Issue: Docker build fails in CI

**Solution:**
```bash
# Test Docker build locally
docker build -t meal-planner-api:test -f Dockerfile .

# Check Dockerfile syntax
docker build --no-cache -t meal-planner-api:test .
```

## Branch Protection Rules

Recommended branch protection settings for `master`/`main`:

1. **Require pull request before merging**
   - Require approvals: 1
   - Dismiss stale reviews on new commits

2. **Require status checks to pass**
   - Required checks:
     - `Lint`
     - `Test (1.21)`
     - `Test (1.22)`
     - `Security Scan`
     - `Build`
     - `All Checks Passed`

3. **Require conversation resolution**

4. **Require signed commits** (optional but recommended)

5. **Do not allow bypassing** (including administrators)

## Performance Optimization

### Caching Strategy:
- Go modules cached using `actions/setup-go@v5` with `cache: true`
- Docker layers cached using `type=gha` (GitHub Actions cache)
- golangci-lint cache automatic

### Typical Run Times:
- Lint: ~2-3 minutes
- Test (single version): ~3-5 minutes
- Security: ~2-3 minutes
- Build: ~2-3 minutes
- **Total CI time: ~8-12 minutes**

### Optimization Tips:
1. Use matrix strategy for parallel jobs
2. Fail fast when appropriate
3. Cache dependencies aggressively
4. Use specific action versions (not `@latest`)
5. Minimize Docker layers

## Security Best Practices

1. **Secrets Management**
   - Never commit secrets to repository
   - Use GitHub Secrets for sensitive data
   - Rotate secrets regularly

2. **Dependency Security**
   - Dependabot auto-updates dependencies
   - Review security advisories
   - Test dependency updates before merging

3. **Code Scanning**
   - CodeQL runs on every PR
   - Gosec scans for Go-specific issues
   - govulncheck checks for known vulnerabilities

4. **Docker Security**
   - Use official, minimal base images
   - Non-root user in containers
   - Multi-stage builds to reduce attack surface
   - Regular base image updates

## Monitoring and Alerts

### GitHub Actions Alerts:
- Email notifications on workflow failures
- Slack/Discord integration available
- Status checks visible on PRs

### Security Alerts:
- Dependabot security updates
- CodeQL security findings
- Secret scanning alerts

## Cost Considerations

### GitHub Actions Usage (Free tier):
- Public repositories: Unlimited
- Private repositories: 2,000 minutes/month

### Optimization:
- Current setup uses ~12 minutes per workflow run
- With Dependabot and weekly runs: ~100-150 minutes/week
- Well within free tier for most projects

## Future Enhancements

Possible additions to CI/CD:

1. **Performance Testing**
   - Load testing with k6 or Artillery
   - Benchmark tests on each PR

2. **E2E Testing**
   - API integration tests
   - Full workflow testing

3. **Automated Deployment**
   - Deploy to staging on merge to develop
   - Deploy to production on release tag
   - Blue-green deployment strategy

4. **Monitoring Integration**
   - Sentry error tracking
   - Datadog/New Relic APM
   - Prometheus metrics

5. **Documentation**
   - Auto-generate API docs
   - Deploy docs to GitHub Pages

## Support and Troubleshooting

For issues with CI/CD:
1. Check workflow logs in Actions tab
2. Review this documentation
3. Check GitHub Actions status page
4. Consult GitHub Actions documentation

## Resources

- [GitHub Actions Documentation](https://docs.github.com/en/actions)
- [golangci-lint Documentation](https://golangci-lint.run/)
- [CodeQL Documentation](https://codeql.github.com/docs/)
- [Dependabot Documentation](https://docs.github.com/en/code-security/dependabot)
- [Docker Documentation](https://docs.docker.com/)

---

**Last Updated:** 2024-11-16
**Maintained by:** Backend Team
