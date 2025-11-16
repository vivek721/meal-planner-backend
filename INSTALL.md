# Golang Backend Installation Guide

## Prerequisites Installation

### 1. Install Go (if not already installed)

**Windows:**
```powershell
# Using Chocolatey
choco install golang

# Or download from: https://golang.org/dl/
# Download the .msi installer and run it
```

**macOS:**
```bash
# Using Homebrew
brew install go

# Or download from: https://golang.org/dl/
```

**Linux (Ubuntu/Debian):**
```bash
# Using apt
sudo apt update
sudo apt install golang-go

# Or download from: https://golang.org/dl/
```

Verify installation:
```bash
go version
# Should output: go version go1.21.x ...
```

### 2. Install PostgreSQL

**Windows:**
```powershell
# Using Chocolatey
choco install postgresql

# Or download from: https://www.postgresql.org/download/windows/
```

**macOS:**
```bash
# Using Homebrew
brew install postgresql
brew services start postgresql
```

**Linux (Ubuntu/Debian):**
```bash
sudo apt update
sudo apt install postgresql postgresql-contrib
sudo systemctl start postgresql
sudo systemctl enable postgresql
```

Verify installation:
```bash
psql --version
# Should output: psql (PostgreSQL) 14.x
```

### 3. Install Air (optional, for hot reload)

```bash
go install github.com/cosmtrek/air@latest
```

Make sure `$GOPATH/bin` is in your PATH:
```bash
# Add to ~/.bashrc or ~/.zshrc or ~/.bash_profile
export PATH=$PATH:$(go env GOPATH)/bin
```

---

## Backend Setup

### Step 1: Navigate to backend directory
```bash
cd C:\Users\mishr\myApp\backend
```

### Step 2: Install Go dependencies
```bash
make install

# Or manually:
go mod download
go mod tidy
```

### Step 3: Set up environment variables
```bash
# Copy the example file
cp .env.example .env

# Edit .env file with your settings
# On Windows, use: notepad .env
# On macOS/Linux, use: nano .env or vim .env
```

**Important environment variables:**
```bash
# Database configuration
DATABASE_URL=postgresql://postgres:postgres@localhost:5432/meal_planner?sslmode=disable

# JWT secret (CHANGE THIS!)
JWT_SECRET=your-super-secret-jwt-key-change-this-in-production-min-32-chars

# Server configuration
PORT=3001
ENVIRONMENT=development

# CORS
FRONTEND_URL=http://localhost:3000
```

### Step 4: Create PostgreSQL database

**Option A: Using psql command line**
```bash
# Login to PostgreSQL (password is usually 'postgres' for local setup)
psql -U postgres

# Inside psql, create database
CREATE DATABASE meal_planner;

# Exit psql
\q
```

**Option B: Using Makefile**
```bash
make db-create
```

**Option C: Using pgAdmin GUI**
1. Open pgAdmin
2. Right-click on "Databases"
3. Select "Create" → "Database"
4. Name it "meal_planner"
5. Click "Save"

### Step 5: Verify database connection
```bash
# Test connection
psql -U postgres -d meal_planner -c "SELECT version();"
```

### Step 6: Run the backend server

**Development mode (with hot reload):**
```bash
make dev
```

**Development mode (without hot reload):**
```bash
make run
```

**Build and run production binary:**
```bash
make build
./bin/meal-planner-api  # On Windows: .\bin\meal-planner-api.exe
```

The server should start on `http://localhost:3001`

### Step 7: Test the API

Open your browser or use curl:
```bash
# Health check
curl http://localhost:3001/health

# API info
curl http://localhost:3001/api

# Test registration
curl -X POST http://localhost:3001/api/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "test@example.com",
    "password": "TestPass123!",
    "name": "Test User"
  }'
```

---

## Troubleshooting

### Go not found
**Error:** `go: command not found`

**Solution:**
1. Verify Go is installed: `go version`
2. Add Go to PATH:
   - Windows: Add `C:\Go\bin` to system PATH
   - macOS/Linux: Add to `~/.bashrc` or `~/.zshrc`:
     ```bash
     export PATH=$PATH:/usr/local/go/bin
     ```
3. Restart terminal

### PostgreSQL connection error
**Error:** `failed to connect to database`

**Solutions:**
1. Ensure PostgreSQL is running:
   ```bash
   # Windows
   net start postgresql-x64-14  # Replace 14 with your version

   # macOS
   brew services list

   # Linux
   sudo systemctl status postgresql
   ```

2. Check credentials in `.env` file
3. Test connection manually:
   ```bash
   psql -U postgres -d meal_planner
   ```

### Port already in use
**Error:** `bind: address already in use`

**Solutions:**
1. Find process using port 3001:
   ```bash
   # Windows
   netstat -ano | findstr :3001

   # macOS/Linux
   lsof -i :3001
   ```

2. Kill the process or change `PORT` in `.env`

### Module errors
**Error:** `cannot find module` or dependency issues

**Solutions:**
```bash
# Clean module cache
go clean -modcache

# Re-download dependencies
rm go.sum
go mod download
go mod tidy
```

### Air not working
**Error:** `air: command not found`

**Solutions:**
1. Install air:
   ```bash
   go install github.com/cosmtrek/air@latest
   ```

2. Add to PATH:
   ```bash
   export PATH=$PATH:$(go env GOPATH)/bin
   ```

3. Or run without air:
   ```bash
   make run
   ```

---

## Verification Checklist

- [ ] Go 1.21+ installed (`go version`)
- [ ] PostgreSQL installed and running
- [ ] Database `meal_planner` created
- [ ] Dependencies installed (`make install`)
- [ ] `.env` file created with correct values
- [ ] Server starts without errors (`make dev` or `make run`)
- [ ] Health endpoint accessible: `http://localhost:3001/health`
- [ ] API endpoint accessible: `http://localhost:3001/api`

---

## Next Steps

1. **Test the API**: Use Postman, curl, or your frontend to test endpoints
2. **Integrate with Frontend**: Update frontend AuthService to use backend API
3. **Read Documentation**: Check `README.md` for API endpoint documentation
4. **Run Tests**: Execute `make test` to ensure everything works

---

## Quick Reference Commands

```bash
# Development
make install       # Install dependencies
make dev           # Run with hot reload
make run           # Run without hot reload
make build         # Build binary

# Database
make db-create     # Create database
make db-drop       # Drop database
make db-reset      # Reset database

# Testing
make test          # Run tests
make test-coverage # Run tests with coverage

# Code Quality
make fmt           # Format code
make vet           # Run go vet

# Utilities
make clean         # Clean build artifacts
make help          # Show all commands
```

---

## Support

If you encounter issues:
1. Check this troubleshooting guide
2. Review `README.md` for detailed documentation
3. Ensure all prerequisites are correctly installed
4. Check `.env` configuration
5. Verify PostgreSQL is running and accessible

---

**Ready to develop!** 🚀

Once the backend is running, you can:
- Test API endpoints with Postman or curl
- Integrate with the React frontend
- Start building additional features (Recipes, Meal Plans, etc.)
