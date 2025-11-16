# Meal Planner Backend - Golang API

Production-ready REST API for the AI-Powered Meal Planner application built with Golang.

## Tech Stack

- **Go 1.21+** - Programming language
- **Gin** - High-performance HTTP web framework
- **GORM** - ORM library for database operations
- **PostgreSQL** - Primary database
- **JWT** - Authentication & authorization
- **Bcrypt** - Password hashing
- **Air** - Live reload for development

## Features

### Epic 1: User Authentication & Profile Management ✅

- [x] User registration with email/password
- [x] User login with JWT tokens
- [x] Token refresh mechanism
- [x] Password hashing with bcrypt
- [x] User profile management
- [x] Change password functionality
- [x] Account lockout after failed login attempts
- [x] Email validation
- [x] Password strength validation
- [x] Protected route middleware
- [x] CORS configuration
- [x] Onboarding completion tracking
- [x] User preferences management

## Project Structure

```
backend/
├── cmd/
│   └── server/
│       └── main.go                 # Application entry point
├── internal/
│   ├── config/
│   │   └── config.go               # Configuration management
│   ├── database/
│   │   └── database.go             # Database connection & migrations
│   ├── handlers/
│   │   ├── auth_handler.go         # Authentication HTTP handlers
│   │   └── user_handler.go         # User profile HTTP handlers
│   ├── middleware/
│   │   ├── auth.go                 # JWT authentication middleware
│   │   ├── cors.go                 # CORS middleware
│   │   ├── error.go                # Error handling middleware
│   │   └── logger.go               # Request logging middleware
│   ├── models/
│   │   ├── user.go                 # User model & business logic
│   │   └── utils.go                # Model utilities
│   ├── repository/
│   │   └── user_repository.go      # User data access layer
│   ├── router/
│   │   └── router.go               # Route configuration
│   ├── services/
│   │   ├── auth_service.go         # Authentication business logic
│   │   └── user_service.go         # User management business logic
│   └── utils/
│       ├── jwt.go                  # JWT token utilities
│       ├── password.go             # Password hashing utilities
│       ├── validator.go            # Input validation utilities
│       ├── password_test.go        # Password tests
│       └── validator_test.go       # Validation tests
├── .air.toml                       # Air configuration (hot reload)
├── .env.example                    # Environment variables template
├── .gitignore                      # Git ignore rules
├── go.mod                          # Go module dependencies
├── go.sum                          # Go module checksums
├── Makefile                        # Build & development commands
└── README.md                       # This file
```

## Getting Started

### Prerequisites

1. **Install Go 1.21 or higher**
   ```bash
   # Download from https://golang.org/dl/
   # Or use package manager:
   # Windows (chocolatey): choco install golang
   # macOS: brew install go
   # Linux: sudo apt install golang-go
   ```

2. **Install PostgreSQL**
   ```bash
   # Windows: Download from https://www.postgresql.org/download/windows/
   # macOS: brew install postgresql
   # Linux: sudo apt install postgresql
   ```

3. **Install Air (optional, for hot reload)**
   ```bash
   go install github.com/cosmtrek/air@latest
   ```

### Installation

1. **Navigate to backend directory**
   ```bash
   cd backend
   ```

2. **Install Go dependencies**
   ```bash
   make install
   # Or manually:
   go mod download
   go mod tidy
   ```

3. **Set up environment variables**
   ```bash
   # Copy example env file
   cp .env.example .env

   # Edit .env with your configuration
   # Update database credentials, JWT secret, etc.
   ```

4. **Create PostgreSQL database**
   ```bash
   # Using psql
   createdb meal_planner

   # Or using Makefile
   make db-create
   ```

5. **Run the server**
   ```bash
   # Development with hot reload
   make dev

   # Or run directly
   make run

   # Or build and run
   make build
   ./bin/meal-planner-api
   ```

The API will be available at `http://localhost:3001`

## Docker Setup (Recommended)

The easiest way to run the backend is using Docker, which eliminates the need to install PostgreSQL locally.

### Prerequisites

- [Docker Desktop for Windows](https://www.docker.com/products/docker-desktop) (version 20.10+)
- Enable WSL 2 backend in Docker Desktop settings (recommended for better performance)

### Quick Start with Docker

```bash
# Navigate to backend directory
cd backend

# Start all services (backend + PostgreSQL)
make docker-up

# View logs
make docker-logs

# Stop services
make docker-down
```

### Verify Installation

```bash
# Check containers are running
make docker-ps

# Test backend health
curl http://localhost:3001/health

# Access PostgreSQL shell
make docker-db-shell
```

### Docker Development Workflow

**Production Mode** (optimized build):
```bash
make docker-up          # Start containers
make docker-logs        # View logs
make docker-rebuild     # Rebuild from scratch
make docker-down        # Stop containers
```

**Development Mode** (hot reload):
```bash
make docker-dev-up      # Start with hot reload
make docker-dev-logs    # View dev logs
make docker-dev-down    # Stop dev containers
```

### Docker Commands Reference

| Command | Description |
|---------|-------------|
| `make docker-up` | Start backend + database |
| `make docker-down` | Stop all containers |
| `make docker-logs` | View all logs |
| `make docker-logs-backend` | View backend logs only |
| `make docker-logs-db` | View database logs only |
| `make docker-ps` | Show container status |
| `make docker-shell` | Open shell in backend |
| `make docker-db-shell` | Open PostgreSQL shell |
| `make docker-rebuild` | Clean rebuild |
| `make docker-clean` | Remove all Docker resources |

### What's Included

- **Backend Service**: Golang API on port 3001
- **PostgreSQL Database**: Version 14 on port 5432
- **Data Persistence**: Named Docker volume for database
- **Health Checks**: Auto-restart on failure
- **Security**: Non-root user, minimal Alpine base image

### Access Points

- Backend API: `http://localhost:3001`
- Health Check: `http://localhost:3001/health`
- PostgreSQL: `localhost:5432` (user: postgres, password: postgres, database: meal_planner)

### Full Documentation

See [DOCKER.md](./DOCKER.md) for:
- Detailed Docker setup instructions
- Troubleshooting guide
- Development vs production modes
- Database management
- Security best practices
- Production deployment strategies

## Environment Variables

Create a `.env` file based on `.env.example`:

```bash
# Server Configuration
PORT=3001
ENVIRONMENT=development

# Database Configuration
DATABASE_URL=postgresql://postgres:postgres@localhost:5432/meal_planner?sslmode=disable

# JWT Configuration
JWT_SECRET=your-super-secret-jwt-key-change-this-in-production-min-32-chars
JWT_EXPIRATION_HOURS=24
JWT_REFRESH_DAYS=30

# Security Configuration
BCRYPT_COST=12

# CORS Configuration
FRONTEND_URL=http://localhost:3000
```

**Important:** Always use a strong, unique `JWT_SECRET` in production!

## API Endpoints

### Health & Info

#### Check Server Health
```http
GET /health
```

**Response:**
```json
{
  "status": "healthy",
  "service": "meal-planner-api"
}
```

#### API Information
```http
GET /api
```

**Response:**
```json
{
  "service": "Meal Planner API",
  "version": "1.0.0",
  "endpoints": {
    "health": "/health",
    "auth": { ... }
  }
}
```

### Authentication Endpoints

#### Register User
```http
POST /api/auth/register
Content-Type: application/json

{
  "email": "user@example.com",
  "password": "SecurePass123!",
  "name": "John Doe"
}
```

**Response (201 Created):**
```json
{
  "user": {
    "id": "user_1234567890_abc123",
    "email": "user@example.com",
    "name": "John Doe",
    "hasCompletedOnboarding": false,
    "createdAt": "2024-10-16T10:30:00Z"
  },
  "token": "eyJhbGciOiJIUzI1NiIs..."
}
```

**Validation Rules:**
- Email: Valid email format required
- Password: Minimum 8 characters, must contain uppercase, lowercase, number, and special character
- Name: Optional

#### Login User
```http
POST /api/auth/login
Content-Type: application/json

{
  "email": "user@example.com",
  "password": "SecurePass123!",
  "rememberMe": false
}
```

**Response (200 OK):**
```json
{
  "user": {
    "id": "user_1234567890_abc123",
    "email": "user@example.com",
    "name": "John Doe",
    "hasCompletedOnboarding": false,
    "createdAt": "2024-10-16T10:30:00Z"
  },
  "token": "eyJhbGciOiJIUzI1NiIs..."
}
```

**Error Response (401 Unauthorized):**
```json
{
  "error": "Invalid email or password"
}
```

**Account Lockout:**
- After 3 failed login attempts, account is locked for 5 minutes
- Error response (403 Forbidden):
```json
{
  "error": "account is locked. Please try again in 4 minute(s)"
}
```

#### Refresh Token
```http
POST /api/auth/refresh
Content-Type: application/json

{
  "token": "eyJhbGciOiJIUzI1NiIs..."
}
```

**Response (200 OK):**
```json
{
  "token": "eyJhbGciOiJIUzI1NiIs..."
}
```

### Protected Endpoints

All protected endpoints require the `Authorization` header:

```http
Authorization: Bearer <token>
```

#### Get Current User
```http
GET /api/auth/me
Authorization: Bearer <token>
```

**Response (200 OK):**
```json
{
  "user": {
    "id": "user_1234567890_abc123",
    "email": "user@example.com",
    "name": "John Doe",
    "hasCompletedOnboarding": true,
    "createdAt": "2024-10-16T10:30:00Z",
    "preferences": {
      "theme": "light",
      "notifications": true
    }
  }
}
```

#### Update Profile
```http
PUT /api/auth/profile
Authorization: Bearer <token>
Content-Type: application/json

{
  "name": "Jane Doe",
  "email": "jane@example.com"
}
```

**Response (200 OK):**
```json
{
  "user": {
    "id": "user_1234567890_abc123",
    "email": "jane@example.com",
    "name": "Jane Doe",
    "hasCompletedOnboarding": true,
    "createdAt": "2024-10-16T10:30:00Z"
  }
}
```

#### Change Password
```http
PUT /api/auth/password
Authorization: Bearer <token>
Content-Type: application/json

{
  "currentPassword": "OldPass123!",
  "newPassword": "NewSecurePass456!"
}
```

**Response (200 OK):**
```json
{
  "message": "password changed successfully"
}
```

#### Complete Onboarding
```http
POST /api/auth/onboarding/complete
Authorization: Bearer <token>
```

**Response (200 OK):**
```json
{
  "user": {
    "id": "user_1234567890_abc123",
    "email": "user@example.com",
    "hasCompletedOnboarding": true,
    ...
  }
}
```

#### Update Preferences
```http
PUT /api/auth/preferences
Authorization: Bearer <token>
Content-Type: application/json

{
  "theme": "dark",
  "notifications": false
}
```

**Response (200 OK):**
```json
{
  "user": {
    "id": "user_1234567890_abc123",
    "preferences": {
      "theme": "dark",
      "notifications": false
    },
    ...
  }
}
```

#### Logout
```http
POST /api/auth/logout
Authorization: Bearer <token>
```

**Response (200 OK):**
```json
{
  "message": "logged out successfully"
}
```

**Note:** JWT is stateless. Logout is handled client-side by removing the token. Server-side token blacklisting can be added if needed.

## Development Commands

```bash
# Install dependencies
make install

# Run with hot reload (requires air)
make dev

# Run without hot reload
make run

# Build binary
make build

# Run tests
make test

# Run tests with coverage
make test-coverage

# Format code
make fmt

# Lint code (requires golangci-lint)
make lint

# Clean build artifacts
make clean

# Database management
make db-create    # Create database
make db-drop      # Drop database
make db-reset     # Drop and recreate database

# View all commands
make help
```

## Testing

### Run All Tests
```bash
make test
```

### Run Tests with Coverage
```bash
make test-coverage
```

This generates:
- `coverage.out` - Coverage data
- `coverage.html` - HTML coverage report

### Test Coverage
- Password utilities: 100%
- Validators: 100%
- Additional tests can be added for services and handlers

## Security Features

### Password Security
- **Hashing**: Bcrypt with cost factor 12 (configurable)
- **Validation**:
  - Minimum 8 characters
  - Must contain uppercase letter
  - Must contain lowercase letter
  - Must contain number
  - Must contain special character

### Account Protection
- **Login Attempts**: Max 3 failed attempts
- **Account Lockout**: 5 minutes after max attempts
- **Lockout Reset**: Automatically resets after timeout

### JWT Security
- **Algorithm**: HS256 (HMAC with SHA-256)
- **Expiration**: 24 hours (configurable)
- **Refresh**: 30 days (configurable)
- **Secret**: Minimum 32 characters recommended

### API Security
- **CORS**: Configured for frontend origin
- **Headers**: Secure headers with Gin defaults
- **Input Validation**: All inputs validated before processing
- **Error Messages**: Generic messages to prevent information disclosure

## Database Schema

### Users Table

```sql
CREATE TABLE users (
    id VARCHAR(255) PRIMARY KEY,
    email VARCHAR(255) UNIQUE NOT NULL,
    name VARCHAR(255),
    password_hash VARCHAR(255) NOT NULL,
    has_completed_onboarding BOOLEAN DEFAULT false,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    deleted_at TIMESTAMP,

    -- Login tracking
    login_attempts INTEGER DEFAULT 0,
    last_login_attempt TIMESTAMP,
    account_locked_until TIMESTAMP,

    -- Preferences (embedded)
    pref_theme VARCHAR(50) DEFAULT 'light',
    pref_notifications BOOLEAN DEFAULT true
);

CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_deleted_at ON users(deleted_at);
```

## Error Handling

All errors follow a consistent JSON format:

```json
{
  "error": "error message here"
}
```

### Common HTTP Status Codes

- `200 OK` - Request successful
- `201 Created` - Resource created successfully
- `400 Bad Request` - Invalid input/validation error
- `401 Unauthorized` - Invalid or missing token
- `403 Forbidden` - Account locked or insufficient permissions
- `404 Not Found` - Resource not found
- `409 Conflict` - Resource already exists (e.g., email in use)
- `500 Internal Server Error` - Server error

## Architecture Decisions

### Layered Architecture
```
Router → Handler → Service → Repository → Database
         ↓
      Middleware
```

- **Router**: Route definitions and grouping
- **Handler**: HTTP request/response handling
- **Service**: Business logic and orchestration
- **Repository**: Data access and persistence
- **Middleware**: Cross-cutting concerns (auth, logging, CORS)

### Design Patterns Used
- **Repository Pattern**: Abstracts data access
- **Service Layer**: Encapsulates business logic
- **Dependency Injection**: Handlers and services receive dependencies
- **Middleware Chain**: Composable request processing

### Why Gin?
- High performance (up to 40x faster than some frameworks)
- Minimal boilerplate
- Excellent middleware support
- Large community and ecosystem
- Production-ready (used by many companies)

### Why GORM?
- Feature-rich ORM with intuitive API
- Auto-migrations support
- Preloading, transactions, hooks
- Multiple database support
- Active development and community

## Production Deployment

### Build for Production
```bash
# Set production environment
export ENVIRONMENT=production

# Build optimized binary
CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o bin/meal-planner-api cmd/server/main.go
```

### Environment Setup
1. Set strong `JWT_SECRET` (minimum 32 characters)
2. Use `DATABASE_URL` for connection string
3. Set `ENVIRONMENT=production`
4. Configure `FRONTEND_URL` for CORS
5. Enable HTTPS/TLS
6. Set up proper logging and monitoring
7. Configure database connection pooling

### Recommended Production Stack
- **Container**: Docker with multi-stage build
- **Database**: Managed PostgreSQL (AWS RDS, Azure Database, etc.)
- **Hosting**: AWS ECS, Google Cloud Run, Azure Container Instances
- **Reverse Proxy**: Nginx or cloud load balancer
- **SSL/TLS**: Let's Encrypt or cloud provider certificates
- **Monitoring**: Prometheus + Grafana or cloud-native solutions
- **Logging**: Structured logging to stdout/stderr (captured by container runtime)

## Frontend Integration

The frontend expects responses in this format:

### Login/Register Response
```typescript
interface AuthResponse {
  user: {
    id: string;
    email: string;
    name?: string;
    hasCompletedOnboarding: boolean;
    createdAt: string;
    preferences?: {
      theme?: string;
      notifications?: boolean;
    };
  };
  token: string;
}
```

### Token Storage
Frontend stores JWT in `localStorage`:
```javascript
localStorage.setItem('meal_planner_auth_token', token);
```

### API Requests
Frontend sends token in Authorization header:
```javascript
headers: {
  'Authorization': `Bearer ${token}`
}
```

## Troubleshooting

### Database Connection Issues
```bash
# Check PostgreSQL is running
pg_isready

# Test connection
psql -U postgres -d meal_planner

# Check connection string in .env
# Ensure DATABASE_URL or DB_* variables are correct
```

### Port Already in Use
```bash
# Find process using port 3001
lsof -i :3001  # macOS/Linux
netstat -ano | findstr :3001  # Windows

# Kill the process or change PORT in .env
```

### Go Module Issues
```bash
# Clean module cache
go clean -modcache

# Re-download dependencies
go mod download
go mod tidy
```

### CORS Errors
Ensure `FRONTEND_URL` in `.env` matches your frontend URL exactly:
```bash
FRONTEND_URL=http://localhost:3000
```

## Contributing

1. Follow Go best practices and idioms
2. Write tests for new features
3. Run `make fmt` before committing
4. Update documentation for API changes
5. Use meaningful commit messages

## License

MIT

## Support

For issues and questions:
- Check the troubleshooting section
- Review the API documentation
- Check the example `.env.example` file
- Ensure all prerequisites are installed

---

**Built with ❤️ using Go and Gin**
