# Quick Start Guide - Docker Setup

Get the Meal Planner backend running in under 5 minutes using Docker.

## Step 1: Prerequisites

Ensure you have Docker Desktop installed:

```bash
# Check Docker is installed and running
docker --version
docker-compose --version
```

If not installed, download from: https://www.docker.com/products/docker-desktop

## Step 2: Start the Backend

```bash
# Navigate to backend directory
cd C:/Users/mishr/myApp/backend

# Start everything with one command
make docker-up
```

You should see:
```
Waiting for services to be healthy...
Backend: http://localhost:3001
Database: localhost:5432
Health: http://localhost:3001/health
```

## Step 3: Verify It's Working

Open a new terminal and test:

```bash
# Test health endpoint
curl http://localhost:3001/health

# Expected response:
# {"status":"ok","timestamp":"..."}
```

Or open in browser: http://localhost:3001/health

## Step 4: Check Container Status

```bash
make docker-ps
```

Expected output:
```
NAME                       STATUS              PORTS
meal-planner-backend       Up (healthy)        0.0.0.0:3001->3001/tcp
meal-planner-db            Up (healthy)        0.0.0.0:5432->5432/tcp
```

## Step 5: Test with Frontend

If your frontend is running on http://localhost:3000:

1. Try registering a new user
2. Try logging in
3. Check browser console for any errors

The backend is configured to allow CORS from http://localhost:3000

## Common Commands

```bash
# View logs
make docker-logs

# Stop everything
make docker-down

# Restart
make docker-restart

# Rebuild from scratch
make docker-rebuild

# Access database
make docker-db-shell
```

## Troubleshooting

### Port Already in Use

If you see "port already in use" error:

```bash
# Windows
netstat -ano | findstr :3001

# Then kill the process using that port
# Or change the port in docker-compose.yml
```

### Cannot Connect to Database

```bash
# Check container status
make docker-ps

# View logs
make docker-logs-db

# Ensure postgres is healthy
docker-compose ps postgres
```

### Frontend CORS Errors

Ensure your frontend is on http://localhost:3000 (not 127.0.0.1:3000)

### Need to Reset Database

```bash
# Stop containers and remove volumes
make docker-clean

# Start fresh
make docker-up
```

## Next Steps

1. Test all API endpoints (see README.md for API documentation)
2. Try registering and logging in from the frontend
3. Explore the database using `make docker-db-shell`

## Full Documentation

- [DOCKER.md](./DOCKER.md) - Comprehensive Docker guide
- [README.md](./README.md) - Full API documentation

## Getting Help

View all available commands:
```bash
make help
```

---

**That's it!** Your backend is now running with PostgreSQL in Docker.
