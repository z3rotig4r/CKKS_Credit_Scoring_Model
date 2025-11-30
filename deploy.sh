#!/bin/bash
# Docker deployment script for CKKS Credit Scoring

set -e

echo "🐳 CKKS Credit Scoring - Docker Deployment"
echo "=========================================="
echo ""

# Check if Docker is installed
if ! command -v docker &> /dev/null; then
    echo "❌ Error: Docker is not installed"
    echo "   Please install Docker Desktop: https://www.docker.com/products/docker-desktop"
    exit 1
fi

# Check if Docker Compose is installed
if ! command -v docker-compose &> /dev/null; then
    echo "❌ Error: Docker Compose is not installed"
    echo "   Please install Docker Compose: https://docs.docker.com/compose/install/"
    exit 1
fi

echo "✅ Docker detected: $(docker --version)"
echo "✅ Docker Compose detected: $(docker-compose --version)"
echo ""

# Check if Docker daemon is running
if ! docker info > /dev/null 2>&1; then
    echo "❌ Error: Docker daemon is not running"
    echo "   Please start Docker Desktop"
    exit 1
fi

echo "✅ Docker daemon is running"
echo ""

# Build images
echo "🔨 Building Docker images..."
echo "   This may take 5-10 minutes on first run..."
echo ""

docker-compose build

if [ $? -eq 0 ]; then
    echo ""
    echo "✅ Build successful!"
    echo ""
else
    echo ""
    echo "❌ Build failed"
    exit 1
fi

# Start services
echo "🚀 Starting services..."
docker-compose up -d

if [ $? -eq 0 ]; then
    echo ""
    echo "✅ Services started successfully!"
    echo ""
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "📡 Service URLs:"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "   🌐 Frontend:  http://localhost:3000"
    echo "   🔧 Backend:   http://localhost:8080"
    echo "   💊 Health:    http://localhost:8080/health"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo ""
    echo "📊 View logs:"
    echo "   docker-compose logs -f"
    echo ""
    echo "🛑 Stop services:"
    echo "   docker-compose down"
    echo ""
    echo "🧹 Clean up everything:"
    echo "   docker-compose down -v --rmi all"
    echo ""
else
    echo ""
    echo "❌ Failed to start services"
    exit 1
fi

# Wait for health checks
echo "⏳ Waiting for services to be healthy..."
sleep 5

# Check backend health
if curl -f http://localhost:8080/health > /dev/null 2>&1; then
    echo "✅ Backend is healthy"
else
    echo "⚠️  Backend health check failed (may still be starting)"
fi

# Check frontend
if curl -f http://localhost:3000 > /dev/null 2>&1; then
    echo "✅ Frontend is healthy"
else
    echo "⚠️  Frontend health check failed (may still be starting)"
fi

echo ""
echo "🎉 Deployment complete!"
echo ""
echo "Open http://localhost:3000 in your browser to get started."
echo ""
