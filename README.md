# Meal Planner Backend

A REST API in Go (Gin, GORM, PostgreSQL) for a meal-planning web app. This first milestone covers
authentication and user accounts: registration and login with JWTs, bcrypt password hashing,
password-strength rules, account lockout after repeated failed logins, and endpoints for profile,
preferences and onboarding state. The code is split into layers (handler, service, repository) and
runs locally or in Docker Compose with a PostgreSQL container.

Frontend (React + TypeScript): [vivek721/meal-planner-frontend](https://github.com/vivek721/meal-planner-frontend)

> Status: the authentication and user-account API is implemented. Recipes, meal plans, shopping
> lists and recommendations are planned but not built yet (see [Roadmap](#roadmap)).

## Features

- **Registration and login.** Email/password sign-up and sign-in return a signed JWT (HS256)
  along with the public user object.
- **Password security.** Passwords are hashed with bcrypt (cost set by `BCRYPT_COST`, default 12).
  They must be at least 8 characters and contain an uppercase letter, a lowercase letter, a digit
  and a symbol.
- **Account lockout.** After 3 failed logins an account is locked for 5 minutes, and the API
  returns `403` while it stays locked. A successful login resets the counter.
- **JWT middleware.** Protected routes require `Authorization: Bearer <token>`. The middleware
  checks the signature, the signing method and the expiry, then puts the user ID into the request
  context.
- **Token refresh.** A still-valid token can be exchanged for a new one with a fresh expiry.
- **Profile management.** Users can update their name and email (the new email is checked for
  format and uniqueness), change their password (the current password is verified), set
  theme/notification preferences and mark onboarding as complete.
- **Email handling.** Emails are trimmed and lowercased before storage, and lookups ignore case.
- **Safe responses.** A dedicated `PublicUser` DTO keeps the password hash and lockout fields out
  of API responses.
- **Middleware chain.** Panic recovery returns a JSON 500, requests are logged with method, path,
  IP, status and latency, and CORS is locked to a single configured frontend origin.
- **Database.** GORM AutoMigrate runs on startup, users are soft-deleted, and the connection pool
  is configured.
- **Containers.** A multi-stage Dockerfile builds a static binary that runs on Alpine as a
  non-root user with a `HEALTHCHECK`. There are Compose files for a production-like stack and for a
  hot-reload (Air) dev stack.
- **CI.** GitHub Actions workflows run golangci-lint, `go vet`, tests on Go 1.21 and 1.22, gosec,
  CodeQL and a Docker build. Dependabot handles Go module and Docker updates.

## Tech Stack

| Area | Choice |
|------|--------|
| Language | Go 1.21 |
| HTTP | [Gin](https://github.com/gin-gonic/gin), [gin-contrib/cors](https://github.com/gin-contrib/cors) |
| Database | PostgreSQL 14, [GORM](https://gorm.io) (pgx driver) |
| Auth | [golang-jwt/jwt v5](https://github.com/golang-jwt/jwt), `golang.org/x/crypto/bcrypt` |
| Config | Environment variables, optional `.env` via [godotenv](https://github.com/joho/godotenv) |
| Tooling | Air (hot reload), golangci-lint, gosec, Docker / Docker Compose, GitHub Actions |

## Architecture

```
Request -> Gin router -> middleware (recover, logger, CORS, [JWT auth])
        -> handler (bind/validate JSON, map errors to HTTP codes)
        -> service (business rules: hashing, lockout, token issuing)
        -> repository (GORM queries) -> PostgreSQL
```

Dependencies are wired by hand in `router.Setup`: the repository goes into the services, and the
services go into the handlers. Services and repositories are exposed as Go interfaces.

```
cmd/server/main.go        # entry point: load .env/config, connect DB, AutoMigrate, start Gin
internal/
  config/                 # env-var configuration with defaults
  database/               # connection, pool settings, AutoMigrate
  router/                 # route registration and dependency wiring
  middleware/             # auth (JWT), CORS, logger, panic recovery
  handlers/               # auth_handler.go, user_handler.go
  services/               # auth_service.go, user_service.go
  repository/             # user_repository.go
  models/                 # User model, PublicUser DTO, ID generation
  utils/                  # JWT, bcrypt, email/password validation (+ unit tests)
scripts/seed.go           # inserts sample users into an empty database
```

## API Endpoints

Base URL: `http://localhost:3001`. Request and response bodies are JSON with camelCase fields.
Every error response has the form `{"error": "<message>"}`.

| Method | Path | Auth | Purpose |
|--------|------|------|---------|
| GET | `/health` | No | Liveness check: `{"status":"healthy"}` |
| GET | `/api` | No | Service info and a list of endpoints |
| POST | `/api/auth/register` | No | Create an account. Body: `email`, `password`, optional `name`. Returns `201 {user, token}` |
| POST | `/api/auth/login` | No | Log in. Body: `email`, `password`. Returns `{user, token}`: `401` for bad credentials, `403` when locked |
| POST | `/api/auth/refresh` | No | Body: `{"token": "<current JWT>"}`. Returns `{token}` with a new expiry. The old token must still be valid |
| GET | `/api/auth/me` | Bearer | Current user, loaded from the database |
| POST | `/api/auth/logout` | Bearer | Returns a success message. Tokens are stateless, so the client discards the token (no server-side revocation) |
| PUT | `/api/auth/profile` | Bearer | Update `name` and/or `email`. Returns `409` if the email is taken |
| PUT | `/api/auth/password` | Bearer | Body: `currentPassword`, `newPassword` (same strength rules as registration) |
| PUT | `/api/auth/preferences` | Bearer | Body: `theme`, `notifications`. Replaces the whole preferences object |
| POST | `/api/auth/onboarding/complete` | Bearer | Set `hasCompletedOnboarding` to `true` |

Example:

```bash
curl -X POST http://localhost:3001/api/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","password":"SecurePass123!","name":"Jane"}'
```

```json
{
  "user": {
    "id": "user_1729074600000000000_k3j9x2m1q",
    "email": "user@example.com",
    "name": "Jane",
    "hasCompletedOnboarding": false,
    "createdAt": "2024-10-16T10:30:00Z"
  },
  "token": "eyJhbGciOiJIUzI1NiIs..."
}
```

## Getting Started

### Option A: Docker Compose (no local Postgres needed)

Prerequisites: Docker with Compose, plus `make` (you can also run the `docker-compose` commands
directly).

```bash
git clone https://github.com/vivek721/meal-planner-backend.git
cd meal-planner-backend

make docker-up        # builds the API image and starts API + PostgreSQL 14
curl http://localhost:3001/health

make docker-logs      # follow logs
make docker-down      # stop
```

To get hot reload in a container (Air plus a mounted source tree), use `make docker-dev-up` and
`make docker-dev-down`. Run `make help` for the full list of targets.

### Option B: Run locally

Prerequisites: Go 1.21+ and PostgreSQL. To get hot reload, also install Air
(`go install github.com/cosmtrek/air@latest`).

```bash
cp .env.example .env          # then edit values as needed
make db-create                # or: createdb -h localhost -U postgres meal_planner
make run                      # go run cmd/server/main.go  (or `make dev` for Air hot reload)
```

The server creates or updates the `users` table on startup through GORM AutoMigrate.

To load sample users into an empty database, run `go run scripts/seed.go`. It creates
`test@example.com`, `demo@example.com` and `newuser@example.com`, and prints their passwords.

### Environment variables

These are read in `internal/config/config.go`. A `.env` file is loaded if one exists.

| Variable | Default | Notes |
|----------|---------|-------|
| `PORT` | `3001` | HTTP port |
| `ENVIRONMENT` | `development` | `production` switches Gin to release mode. `development` turns on GORM SQL logging |
| `DATABASE_URL` | none | Postgres connection URL. If it is set, the `DB_*` values are ignored |
| `DB_HOST` / `DB_PORT` / `DB_USER` / `DB_PASSWORD` / `DB_NAME` / `DB_SSLMODE` | `localhost` / `5432` / `postgres` / `postgres` / `meal_planner` / `disable` | Used when `DATABASE_URL` is empty |
| `JWT_SECRET` | placeholder | HMAC signing key. **Set a strong value (32+ chars) outside local dev** |
| `JWT_EXPIRATION_HOURS` | `24` | Token lifetime |
| `BCRYPT_COST` | `12` | bcrypt work factor |
| `FRONTEND_URL` | `http://localhost:3000` | The only allowed CORS origin |

`JWT_REFRESH_DAYS`, `RATE_LIMIT_ENABLED` and `RATE_LIMIT_PER_MIN` appear in `.env.example` and are
parsed into the config, but no code reads them yet.

## Running Tests

```bash
make test             # go test ./... -v
make test-coverage    # writes coverage.out and coverage.html
make test-race        # with the race detector
```

The unit tests currently cover `internal/utils`: password hashing and verification, and email and
password validation. The handlers, services and repository have no tests yet.

## Roadmap

These items are planned (see `docs/epics/`) and **not implemented yet**:

- Recipe service: recipe CRUD and search
- Meal-planning API: weekly meal plans
- Shopping-list generation from meal plans
- Meal recommendations and nutrition analysis
- Notifications (email) and admin features
- Rate limiting, server-side token revocation, separate long-lived refresh tokens
- Handler/service tests and integration tests against PostgreSQL
