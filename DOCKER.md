# Docker Guide for Meal Planner Backend

This guide covers everything you need to run the Meal Planner backend using Docker and Docker Compose.

## Table of Contents

- [Prerequisites](#prerequisites)
- [Quick Start](#quick-start)
- [Architecture](#architecture)
- [Development Workflow](#development-workflow)
- [Production Build](#production-build)
- [Makefile Commands](#makefile-commands)
- [Environment Variables](#environment-variables)
- [Database Management](#database-management)
- [Troubleshooting](#troubleshooting)
- [Best Practices](#best-practices)

## Prerequisites

### Windows Requirements

1. **Docker Desktop for Windows**
   - Download from: https://www.docker.com/products/docker-desktop
   - Minimum version: 20.10.x or later
   - **Important**: Enable WSL 2 backend for better performance

2. **WSL 2 (Recommended)**
   - Open PowerShell as Administrator and run:
     ```powershell
     wsl --install
     ```
   - Restart your computer
   - Configure Docker Desktop to use WSL 2 backend (Settings > General > Use WSL 2 based engine)

3. **Make (Optional but Recommended)**
   - Already available in Git Bash/MINGW64
   - Or install via: `choco install make` (requires Chocolatey)

### Verify Installation

```bash
# Check Docker
docker --version
# Expected: Docker version 20.10.x or higher

# Check Docker Compose
docker-compose --version
# Expected: Docker Compose version v2.x.x or higher

# Check Docker is running
docker ps
# Should return empty list (no containers running yet)
```

## Quick Start

### Option 1: Using Make (Recommended)

```bash
# Navigate to backend directory
cd C:/Users/mishr/myApp/backend

# Start all services (backend + database)
make docker-up

# View logs
make docker-logs

# Stop services
make docker-down
```

### Option 2: Using Docker Compose Directly

```bash
# Start services
docker-compose up -d

# View logs
docker-compose logs -f

# Stop services
docker-compose down
```

### Verify Everything is Running

After running `docker-up` or `docker-compose up -d`:

1. **Check Container Status**:
   ```bash
   docker-compose ps
   # or
   make docker-ps
   ```

   Expected output:
   ```
   NAME                       STATUS              PORTS
   meal-planner-backend       Up (healthy)        0.0.0.0:3001->3001/tcp
   meal-planner-db            Up (healthy)        0.0.0.0:5432->5432/tcp
   ```

2. **Test Backend Health**:
   ```bash
   curl http://localhost:3001/health
   # or open in browser: http://localhost:3001/health
   ```

   Expected response:
   ```json
   {"status":"ok","timestamp":"..."}
   ```

3. **Test Frontend Integration**:
   - Ensure frontend is running on http://localhost:3000
   - Try login/register from frontend
   - Check browser console for any CORS errors

## Architecture

### Services

#### Backend Service (`backend`)
- **Image**: Built from local Dockerfile (multi-stage build)
- **Port**: 3001 (mapped to host)
- **Language**: Go 1.21
- **Framework**: Gin
- **ORM**: GORM
- **Features**:
  - Multi-stage build (builder + runtime)
  - Alpine-based runtime (minimal image size)
  - Non-root user for security
  - Health checks every 30s
  - Auto-restart on failure

#### PostgreSQL Service (`postgres`)
- **Image**: postgres:14-alpine
- **Port**: 5432 (mapped to host)
- **Database**: meal_planner
- **User/Password**: postgres/postgres (development only)
- **Features**:
  - Named volume for data persistence
  - Health checks every 10s
  - Auto-restart on failure

### Network

- **Name**: meal-planner-network
- **Type**: Bridge
- **Purpose**: Isolates services and enables service discovery by name

### Volumes

- **postgres_data**: Persists PostgreSQL data across container restarts
- **Location**: Managed by Docker (use `docker volume inspect meal-planner-postgres-data`)

## Development Workflow

### Development Mode (Hot Reload)

For active development with automatic code reloading:

```bash
# Start development containers
make docker-dev-up
# or
docker-compose -f docker-compose.dev.yml up -d

# View logs (see code changes being detected)
make docker-dev-logs
# or
docker-compose -f docker-compose.dev.yml logs -f

# Stop development containers
make docker-dev-down
```

**How it works**:
- Uses `Dockerfile.dev` instead of `Dockerfile`
- Mounts local code as volume (changes reflect immediately)
- Uses Air for hot reload (automatically rebuilds on file changes)
- Faster iteration (no need to rebuild Docker image)

**When to use**:
- Active feature development
- Debugging
- Testing code changes quickly
- Learning the codebase

### Production Mode

For production-like testing:

```bash
# Build and start
make docker-up

# Rebuild from scratch (after changing dependencies)
make docker-rebuild
```

**Differences from dev mode**:
- Multi-stage build (smaller image)
- No code mounting (code baked into image)
- No hot reload (restart required for changes)
- Optimized binary (smaller, faster)

**When to use**:
- Testing before deployment
- Performance testing
- Integration testing
- CI/CD pipelines

## Production Build

### Building for Production

```bash
# Build optimized image
make docker-build
# or
docker-compose build

# Build without cache (clean build)
docker-compose build --no-cache
```

### Image Details

The production Dockerfile uses a multi-stage build:

**Stage 1: Builder**
- Base: golang:1.21-alpine
- Installs build dependencies
- Downloads Go modules (cached layer)
- Compiles static binary with optimizations
- Binary size reduction flags: `-ldflags="-w -s"`

**Stage 2: Runtime**
- Base: alpine:latest (very small)
- Only includes compiled binary + CA certificates
- Runs as non-root user (appuser)
- Final image size: ~15-20 MB (vs ~800 MB with full Go image)

### Security Features

1. **Non-root User**: Container runs as `appuser` (UID 1000)
2. **Minimal Base Image**: Alpine Linux (fewer vulnerabilities)
3. **No Build Tools**: Runtime image has no compilers or dev tools
4. **Static Binary**: No dynamic linking (CGO_ENABLED=0)
5. **Health Checks**: Auto-restart unhealthy containers

## Makefile Commands

### Quick Reference

```bash
make help                  # Show all available commands
```

### Docker Commands (Production)

| Command | Description |
|---------|-------------|
| `make docker-build` | Build Docker image |
| `make docker-up` | Start containers (detached) |
| `make docker-down` | Stop and remove containers |
| `make docker-logs` | View logs (all services) |
| `make docker-logs-backend` | View backend logs only |
| `make docker-logs-db` | View database logs only |
| `make docker-ps` | Show container status |
| `make docker-shell` | Open shell in backend container |
| `make docker-db-shell` | Open PostgreSQL shell |
| `make docker-restart` | Restart containers |
| `make docker-rebuild` | Rebuild and restart (clean build) |
| `make docker-clean` | Remove containers, volumes, images |

### Docker Development Commands

| Command | Description |
|---------|-------------|
| `make docker-dev-up` | Start dev containers with hot reload |
| `make docker-dev-down` | Stop dev containers |
| `make docker-dev-logs` | View dev logs |
| `make docker-dev-rebuild` | Rebuild dev containers |

### Quick Aliases

| Command | Same As |
|---------|---------|
| `make quick-start` | `make docker-up` |
| `make stop` | `make docker-down` |

## Environment Variables

### Configuration Files

- **`.env`**: Local development (not in Docker)
- **`.env.example`**: Template with documentation
- **`.env.docker`**: Docker-specific defaults
- **`docker-compose.yml`**: Environment variables for containers

### Key Differences

**Local Development** (`.env`):
```bash
DATABASE_URL=postgresql://postgres:postgres@localhost:5432/meal_planner?sslmode=disable
DB_HOST=localhost
```

**Docker** (configured in `docker-compose.yml`):
```bash
DATABASE_URL=postgresql://postgres:postgres@postgres:5432/meal_planner?sslmode=disable
DB_HOST=postgres  # Docker service name, not 'localhost'
```

### Required Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | 3001 | Backend server port |
| `ENVIRONMENT` | development | Environment mode |
| `DATABASE_URL` | See above | PostgreSQL connection string |
| `JWT_SECRET` | (change in prod!) | Secret key for JWT tokens |
| `JWT_EXPIRATION_HOURS` | 24 | Access token expiration |
| `JWT_REFRESH_DAYS` | 30 | Refresh token expiration |
| `BCRYPT_COST` | 12 | Password hashing cost |
| `FRONTEND_URL` | http://localhost:3000 | CORS allowed origin |
| `RATE_LIMIT_ENABLED` | true | Enable rate limiting |
| `RATE_LIMIT_PER_MIN` | 100 | Requests per minute limit |

### Overriding Variables

You can override environment variables in multiple ways:

1. **Edit `docker-compose.yml`** (permanent):
   ```yaml
   environment:
     JWT_SECRET: your-custom-secret
   ```

2. **Create `.env` file** (docker-compose reads it automatically):
   ```bash
   JWT_SECRET=your-custom-secret
   ```

3. **Pass at runtime** (temporary):
   ```bash
   JWT_SECRET=test docker-compose up
   ```

## Database Management

### Accessing the Database

#### Option 1: Docker Shell (Recommended)

```bash
# Open PostgreSQL shell in container
make docker-db-shell
# or
docker-compose exec postgres psql -U postgres -d meal_planner

# Now you're in psql shell:
# \dt          # List tables
# \d users     # Describe users table
# SELECT * FROM users;
# \q           # Quit
```

#### Option 2: External Client

Connect using any PostgreSQL client (pgAdmin, DBeaver, etc.):

```
Host: localhost
Port: 5432
Database: meal_planner
Username: postgres
Password: postgres
```

### Database Operations

#### View Data

```bash
# Connect to database
make docker-db-shell

# In psql:
SELECT * FROM users;
SELECT * FROM meals;
```

#### Reset Database

```bash
# Stop containers
make docker-down

# Remove volumes (deletes all data)
docker-compose down -v

# Start fresh
make docker-up
```

#### Backup Database

```bash
# Create backup file
docker-compose exec postgres pg_dump -U postgres meal_planner > backup.sql

# Restore from backup
docker-compose exec -T postgres psql -U postgres meal_planner < backup.sql
```

#### Run Migrations

Migrations run automatically when the backend starts (see `cmd/server/main.go`).

To manually trigger migrations:

```bash
# Shell into backend container
make docker-shell

# Run the application (will run migrations)
./server
```

### Data Persistence

Data persists across container restarts thanks to Docker volumes:

```bash
# View volume details
docker volume inspect meal-planner-postgres-data

# List all volumes
docker volume ls

# Remove volume (DELETES ALL DATA)
docker volume rm meal-planner-postgres-data
```

## Troubleshooting

### Common Issues

#### 1. Port Already in Use

**Error**:
```
Error: bind: address already in use
```

**Solution**:
```bash
# Find what's using the port
netstat -ano | findstr :3001    # Windows
lsof -i :3001                   # Linux/Mac

# Stop the conflicting process or change port in docker-compose.yml
ports:
  - "3002:3001"  # Map host:3002 to container:3001
```

#### 2. Database Connection Refused

**Error**:
```
failed to connect to database: dial tcp 127.0.0.1:5432: connect: connection refused
```

**Causes**:
1. PostgreSQL container not healthy yet
2. Wrong hostname (using 'localhost' instead of 'postgres')
3. Database container failed to start

**Solution**:
```bash
# Check container status
make docker-ps

# Check if postgres is healthy
docker-compose ps postgres

# View postgres logs
make docker-logs-db

# Wait for health check
# The backend has depends_on health check - should wait automatically
# If not, restart:
make docker-restart
```

#### 3. Containers Not Starting

**Error**:
```
Container exited with code 1
```

**Solution**:
```bash
# View logs for errors
make docker-logs

# Check specific service
docker-compose logs backend
docker-compose logs postgres

# Common fixes:
# 1. Rebuild images
make docker-rebuild

# 2. Clean everything and start fresh
make docker-clean
make docker-up

# 3. Check .env variables
cat docker-compose.yml | grep -A 20 environment
```

#### 4. Cannot Access from Frontend

**Error**: CORS errors in browser console

**Solution**:
1. Check CORS configuration in `docker-compose.yml`:
   ```yaml
   FRONTEND_URL: http://localhost:3000
   ```

2. Verify backend is accessible:
   ```bash
   curl http://localhost:3001/health
   ```

3. Check browser console for exact error
4. Ensure frontend is on http://localhost:3000 (not 127.0.0.1:3000)

#### 5. Changes Not Reflecting

**Production Mode**:
```bash
# Code is baked into image - must rebuild
make docker-rebuild
```

**Development Mode**:
```bash
# Should auto-reload with Air
# If not:
make docker-dev-logs  # Check for Air errors
make docker-dev-rebuild  # Rebuild dev containers
```

#### 6. Out of Disk Space

**Error**:
```
no space left on device
```

**Solution**:
```bash
# Remove unused images, containers, volumes
docker system prune -a --volumes

# Or selectively:
docker image prune -a        # Remove unused images
docker volume prune          # Remove unused volumes
docker container prune       # Remove stopped containers
```

#### 7. Windows Path Issues

If you see path-related errors:

```bash
# Ensure you're using forward slashes or proper escaping
cd C:/Users/mishr/myApp/backend   # Good
cd C:\Users\mishr\myApp\backend   # May cause issues in some contexts

# Or use Git Bash which handles paths better
```

### Debug Mode

Run containers in foreground to see live logs:

```bash
# Foreground mode (see all logs)
docker-compose up

# Stop with Ctrl+C
```

### Health Check Issues

```bash
# Check health status
docker inspect meal-planner-backend | grep -A 10 Health
docker inspect meal-planner-db | grep -A 10 Health

# Manual health check
curl http://localhost:3001/health
docker-compose exec postgres pg_isready -U postgres -d meal_planner
```

## Best Practices

### Development

1. **Use Dev Mode for Development**:
   ```bash
   make docker-dev-up  # Hot reload enabled
   ```

2. **Use Production Mode for Testing**:
   ```bash
   make docker-up  # Test production build locally
   ```

3. **Check Logs Regularly**:
   ```bash
   make docker-logs  # See what's happening
   ```

4. **Clean Up Periodically**:
   ```bash
   docker system prune  # Remove unused resources
   ```

### Security

1. **Never Commit Secrets**:
   - `.env` is in `.gitignore`
   - Use different secrets for each environment
   - Rotate JWT_SECRET in production

2. **Update Base Images**:
   ```bash
   # Pull latest images
   docker-compose pull

   # Rebuild with latest
   make docker-rebuild
   ```

3. **Scan for Vulnerabilities**:
   ```bash
   docker scan meal-planner-api:latest
   ```

### Performance

1. **Use Multi-stage Builds**: Already implemented (see Dockerfile)

2. **Optimize Layer Caching**:
   - `go.mod` and `go.sum` copied first
   - Source code copied last
   - Dependencies cached unless go.mod changes

3. **Minimize Image Size**:
   - Alpine base image
   - Remove build artifacts
   - Static binary (no CGO)

4. **Resource Limits** (optional):
   ```yaml
   # Add to docker-compose.yml
   services:
     backend:
       deploy:
         resources:
           limits:
             cpus: '1'
             memory: 512M
   ```

### Production Deployment

For actual production (not just local testing):

1. **Use Environment-Specific Configs**:
   ```bash
   # docker-compose.prod.yml
   version: '3.8'
   services:
     backend:
       environment:
         ENVIRONMENT: production
         JWT_SECRET: ${JWT_SECRET}  # From secure vault
   ```

2. **Use External Database**:
   - Don't run PostgreSQL in Docker in production
   - Use managed service (RDS, Cloud SQL, etc.)
   - Update DATABASE_URL to point to external DB

3. **Enable HTTPS**:
   - Add nginx/traefik as reverse proxy
   - Use Let's Encrypt for SSL certificates

4. **Add Monitoring**:
   - Prometheus for metrics
   - Grafana for visualization
   - ELK stack for log aggregation

5. **Use Orchestration**:
   - Kubernetes for multi-node deployment
   - Docker Swarm for simpler setups
   - Cloud-native solutions (ECS, GKE, AKS)

## Additional Resources

- [Docker Documentation](https://docs.docker.com/)
- [Docker Compose Documentation](https://docs.docker.com/compose/)
- [Go Docker Best Practices](https://docs.docker.com/language/golang/)
- [PostgreSQL Docker Hub](https://hub.docker.com/_/postgres)

## Getting Help

If you encounter issues not covered here:

1. Check container logs: `make docker-logs`
2. Check container status: `make docker-ps`
3. Try clean rebuild: `make docker-clean && make docker-up`
4. Search Docker documentation
5. Check application-specific logs in backend code

---

**Last Updated**: 2025-10-16
**Docker Version**: 20.10+
**Docker Compose Version**: v2.x+
