# Makefile for CKKS Credit Scoring System

.PHONY: help docker-build docker-up docker-down docker-logs docker-clean docker-rebuild test benchmark

# Default target
help:
	@echo "🐳 CKKS Credit Scoring - Docker Commands"
	@echo ""
	@echo "Usage:"
	@echo "  make docker-build    - Build Docker images"
	@echo "  make docker-up       - Start services (detached)"
	@echo "  make docker-down     - Stop services"
	@echo "  make docker-logs     - View logs"
	@echo "  make docker-clean    - Remove all containers, images, volumes"
	@echo "  make docker-rebuild  - Rebuild and restart everything"
	@echo "  make test            - Run E2E tests locally"
	@echo "  make benchmark       - Run performance benchmarks"
	@echo ""

# Build Docker images
docker-build:
	@echo "🔨 Building Docker images..."
	docker-compose build

# Start services
docker-up:
	@echo "🚀 Starting CKKS services..."
	docker-compose up -d
	@echo ""
	@echo "✅ Services started!"
	@echo "   Frontend: http://localhost:3000"
	@echo "   Backend:  http://localhost:8080"
	@echo "   Health:   http://localhost:8080/health"
	@echo ""
	@echo "📊 View logs: make docker-logs"

# Stop services
docker-down:
	@echo "🛑 Stopping services..."
	docker-compose down

# View logs
docker-logs:
	docker-compose logs -f

# View backend logs only
docker-logs-backend:
	docker-compose logs -f backend

# View frontend logs only
docker-logs-frontend:
	docker-compose logs -f frontend

# Remove everything (containers, images, volumes)
docker-clean:
	@echo "🧹 Cleaning up Docker resources..."
	docker-compose down -v --rmi all --remove-orphans
	@echo "✅ Cleanup complete"

# Rebuild and restart
docker-rebuild:
	@echo "🔄 Rebuilding and restarting..."
	docker-compose down
	docker-compose build --no-cache
	docker-compose up -d
	@echo "✅ Rebuild complete"

# Run E2E tests (locally)
test:
	@echo "🧪 Running E2E tests..."
	cd test && go run e2e.go

# Run benchmarks (locally)
benchmark:
	@echo "📊 Running benchmarks..."
	./run_benchmarks.sh

# Check Docker status
docker-status:
	@echo "📊 Docker Services Status:"
	@docker-compose ps

# Shell into backend container
docker-shell-backend:
	docker exec -it ckks-backend sh

# Shell into frontend container
docker-shell-frontend:
	docker exec -it ckks-frontend sh

# Quick start (build and run)
quick-start: docker-build docker-up
	@echo "⚡ Quick start complete!"
