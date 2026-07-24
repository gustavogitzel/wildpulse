# WildPulse Monorepo Makefile

.PHONY: help build run-api run-worker test docker-up docker-down doc clean

# Detect Colima socket if present
COLIMA_SOCK := $(HOME)/.colima/default/docker.sock
DOCKER_ENV := $(shell if [ -S $(COLIMA_SOCK) ]; then echo "DOCKER_HOST=unix://$(COLIMA_SOCK)"; fi)
DOCKER_COMPOSE := $(shell if command -v docker-compose >/dev/null 2>&1; then echo "$(DOCKER_ENV) docker-compose"; else echo "$(DOCKER_ENV) docker compose"; fi)

# Default target
all: help

## help: Display available Makefile commands
help:
	@echo "🦫 WildPulse Backend - Commands:"
	@echo ""
	@sed -n 's/^##//p' $(MAKEFILE_LIST) | column -t -s ':' | sed -e 's/^/ /'

## build: Build binaries for API server and ingestion workers
build:
	@echo "🔨 Building API Server and Ingestion Workers..."
	@mkdir -p bin
	go build -o bin/wildpulse-api ./apps/api/cmd/server/main.go
	go build -o bin/wildpulse-worker ./apps/workers/cmd/worker/main.go
	@echo "✅ Binaries built in bin/"

## run-api: Run the REST API server locally
run-api:
	@echo "🚀 Starting WildPulse REST API on http://localhost:8080..."
	go run ./apps/api/cmd/server/main.go

## run-worker: Run the ingestion worker runner locally
run-worker:
	@echo "🦫 Starting WildPulse Ingestion Workers..."
	go run ./apps/workers/cmd/worker/main.go

## test: Run unit and integration tests across all workspace packages
test:
	@echo "🧪 Running test suite..."
	go test -v ./pkg/... ./apps/api/... ./apps/workers/...

## docker-up: Start PostgreSQL 15 + PostGIS container in background
docker-up:
	@echo "🐘 Starting PostGIS database container..."
	$(DOCKER_COMPOSE) up -d

## docker-down: Stop and remove local Docker containers
docker-down:
	@echo "🛑 Stopping PostGIS database container..."
	$(DOCKER_COMPOSE) down

## doc: Display package documentation using go doc
doc:
	@echo "📖 Package documentation for wildpulse/pkg/domain:"
	go doc ./pkg/domain
	@echo "\n📖 Package documentation for wildpulse/pkg/spatial:"
	go doc ./pkg/spatial

## clean: Remove compiled binary output directory
clean:
	@echo "🧹 Cleaning binaries..."
	rm -rf bin/
