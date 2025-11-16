.PHONY: help install build run dev test test-coverage clean fmt lint vet \
        docker-build docker-up docker-down docker-logs docker-shell docker-db-shell \
        docker-clean docker-rebuild docker-dev-up docker-dev-down docker-dev-logs \
        db-create db-drop db-reset

# Variables
BINARY_NAME=meal-planner-api
GO=go
GOFLAGS=-v
DB_URL?=postgresql://postgres:postgres@localhost:5432/meal_planner?sslmode=disable
DOCKER_COMPOSE=docker-compose
DOCKER_COMPOSE_DEV=docker-compose -f docker-compose.dev.yml

help: ## Display this help screen
	@grep -h -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

# Local Development Commands
install: ## Install dependencies
	$(GO) mod download
	$(GO) mod tidy

build: ## Build the application
	$(GO) build $(GOFLAGS) -o bin/$(BINARY_NAME) cmd/server/main.go

run: ## Run the application
	$(GO) run cmd/server/main.go

dev: ## Run with hot reload (requires air: go install github.com/cosmtrek/air@latest)
	air

test: ## Run tests
	$(GO) test ./... -v

test-coverage: ## Run tests with coverage
	$(GO) test ./... -v -coverprofile=coverage.out
	$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

clean: ## Clean build artifacts
	rm -rf bin/
	rm -f coverage.out coverage.html
	$(GO) clean

fmt: ## Format code
	$(GO) fmt ./...

lint: ## Run linter (requires golangci-lint)
	golangci-lint run

vet: ## Run go vet
	$(GO) vet ./...

# Docker Commands (Production-like)
docker-build: ## Build Docker image
	$(DOCKER_COMPOSE) build

docker-up: ## Start Docker containers
	$(DOCKER_COMPOSE) up -d
	@echo "Waiting for services to be healthy..."
	@sleep 5
	@echo "Backend: http://localhost:3001"
	@echo "Database: localhost:5432"
	@echo "Health: http://localhost:3001/health"

docker-down: ## Stop Docker containers
	$(DOCKER_COMPOSE) down

docker-logs: ## View Docker logs (follow mode)
	$(DOCKER_COMPOSE) logs -f

docker-logs-backend: ## View backend logs only
	$(DOCKER_COMPOSE) logs -f backend

docker-logs-db: ## View database logs only
	$(DOCKER_COMPOSE) logs -f postgres

docker-ps: ## Show running containers
	$(DOCKER_COMPOSE) ps

docker-shell: ## Open shell in backend container
	$(DOCKER_COMPOSE) exec backend sh

docker-db-shell: ## Open PostgreSQL shell
	$(DOCKER_COMPOSE) exec postgres psql -U postgres -d meal_planner

docker-restart: ## Restart Docker containers
	$(DOCKER_COMPOSE) restart

docker-rebuild: ## Rebuild and restart containers
	$(DOCKER_COMPOSE) down
	$(DOCKER_COMPOSE) build --no-cache
	$(DOCKER_COMPOSE) up -d
	@echo "Containers rebuilt and started"

docker-clean: ## Remove containers, volumes, and images
	$(DOCKER_COMPOSE) down -v
	docker rmi meal-planner-api:latest 2>/dev/null || true
	@echo "Docker resources cleaned"

# Docker Development Commands (with hot reload)
docker-dev-up: ## Start development containers with hot reload
	$(DOCKER_COMPOSE_DEV) up -d
	@echo "Development mode started with hot reload"
	@echo "Backend: http://localhost:3001"

docker-dev-down: ## Stop development containers
	$(DOCKER_COMPOSE_DEV) down

docker-dev-logs: ## View development logs
	$(DOCKER_COMPOSE_DEV) logs -f

docker-dev-rebuild: ## Rebuild development containers
	$(DOCKER_COMPOSE_DEV) down
	$(DOCKER_COMPOSE_DEV) build --no-cache
	$(DOCKER_COMPOSE_DEV) up -d

# Database Commands
db-create: ## Create database (local PostgreSQL)
	createdb -h localhost -U postgres meal_planner

db-drop: ## Drop database (local PostgreSQL)
	dropdb -h localhost -U postgres meal_planner

db-reset: db-drop db-create ## Reset database (local PostgreSQL)

# Quick Start Commands
quick-start: docker-up ## Quick start with Docker (alias for docker-up)

stop: docker-down ## Stop all containers (alias for docker-down)

# Default target
.DEFAULT_GOAL := help
