#!/bin/bash

# E2E Test Automation Script for CKKS Credit Scoring
# This script runs the complete end-to-end test cycle

set -e

echo "🚀 CKKS Credit Scoring - E2E Test Automation"
echo "=============================================="
echo ""

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
BACKEND_PORT=8080
BACKEND_STARTUP_TIMEOUT=5
TEST_DIR="$(cd "$(dirname "$0")" && pwd)"
ROOT_DIR="$(dirname "$TEST_DIR")"
BACKEND_DIR="$ROOT_DIR/backend"
LOG_DIR="/tmp/ckks_test_logs"

mkdir -p "$LOG_DIR"

echo -e "${BLUE}📂 Project Structure:${NC}"
echo "  Root: $ROOT_DIR"
echo "  Backend: $BACKEND_DIR"
echo "  Test: $TEST_DIR"
echo "  Logs: $LOG_DIR"
echo ""

# Step 1: Check if backend is running
echo -e "${BLUE}🔍 Step 1: Checking backend status...${NC}"
if curl -s http://localhost:$BACKEND_PORT/health > /dev/null 2>&1; then
    echo -e "${GREEN}✅ Backend already running on port $BACKEND_PORT${NC}"
    BACKEND_RUNNING=true
else
    echo -e "${YELLOW}⚠️  Backend not running${NC}"
    BACKEND_RUNNING=false
fi
echo ""

# Step 2: Build and start backend if needed
if [ "$BACKEND_RUNNING" = false ]; then
    echo -e "${BLUE}🔨 Step 2: Building backend...${NC}"
    cd "$BACKEND_DIR"
    
    if [ ! -f "main.go" ]; then
        echo -e "${RED}❌ Backend main.go not found!${NC}"
        exit 1
    fi
    
    go build -o backend main.go
    if [ $? -ne 0 ]; then
        echo -e "${RED}❌ Backend build failed!${NC}"
        exit 1
    fi
    echo -e "${GREEN}✅ Backend built successfully${NC}"
    echo ""
    
    echo -e "${BLUE}🚀 Step 3: Starting backend...${NC}"
    nohup ./backend > "$LOG_DIR/backend.log" 2>&1 &
    BACKEND_PID=$!
    echo "Backend PID: $BACKEND_PID"
    
    # Wait for backend to start
    echo -n "Waiting for backend to start"
    for i in {1..10}; do
        sleep 1
        echo -n "."
        if curl -s http://localhost:$BACKEND_PORT/health > /dev/null 2>&1; then
            echo ""
            echo -e "${GREEN}✅ Backend started successfully${NC}"
            break
        fi
        if [ $i -eq 10 ]; then
            echo ""
            echo -e "${RED}❌ Backend failed to start within ${BACKEND_STARTUP_TIMEOUT}s${NC}"
            echo "Backend logs:"
            tail -20 "$LOG_DIR/backend.log"
            exit 1
        fi
    done
else
    echo -e "${BLUE}⏭️  Step 2-3: Skipped (backend already running)${NC}"
fi
echo ""

# Step 4: Build test
echo -e "${BLUE}🔨 Step 4: Building E2E test...${NC}"
cd "$TEST_DIR"

if [ ! -f "go.mod" ]; then
    echo "Initializing Go module..."
    go mod init ckks_credit/test
    go get github.com/tuneinsight/lattigo/v6@latest
fi

go build -o e2e_test e2e_test.go
if [ $? -ne 0 ]; then
    echo -e "${RED}❌ Test build failed!${NC}"
    exit 1
fi
echo -e "${GREEN}✅ Test built successfully${NC}"
echo ""

# Step 5: Run test
echo -e "${BLUE}🧪 Step 5: Running E2E test...${NC}"
echo "=========================================="
echo ""

./e2e_test | tee "$LOG_DIR/test_output.log"
TEST_EXIT_CODE=${PIPESTATUS[0]}

echo ""
echo "=========================================="

# Step 6: Analyze results
echo ""
echo -e "${BLUE}📊 Step 6: Test Analysis${NC}"
echo ""

if [ $TEST_EXIT_CODE -eq 0 ]; then
    echo -e "${GREEN}✅ All tests passed!${NC}"
else
    echo -e "${RED}❌ Some tests failed (exit code: $TEST_EXIT_CODE)${NC}"
fi

# Extract metrics from log
PASS_COUNT=$(grep -o "passed" "$LOG_DIR/test_output.log" | wc -l)
TOTAL_TESTS=$(grep -o "Test [0-9]/" "$LOG_DIR/test_output.log" | wc -l)

echo ""
echo "Test Results:"
echo "  Total: $TOTAL_TESTS"
echo "  Passed: $PASS_COUNT"
echo "  Failed: $((TOTAL_TESTS - PASS_COUNT))"

# Extract timing metrics
echo ""
echo "Performance Metrics:"
grep "completed in" "$LOG_DIR/test_output.log" | while read line; do
    echo "  $line"
done

# Step 7: Cleanup
echo ""
echo -e "${BLUE}🧹 Step 7: Cleanup${NC}"

if [ "$BACKEND_RUNNING" = false ] && [ ! -z "$BACKEND_PID" ]; then
    echo "Stopping backend (PID: $BACKEND_PID)..."
    kill $BACKEND_PID 2>/dev/null || true
    echo -e "${GREEN}✅ Backend stopped${NC}"
else
    echo "Backend left running (was already running before test)"
fi

echo ""
echo -e "${BLUE}📁 Logs saved to:${NC}"
echo "  Backend: $LOG_DIR/backend.log"
echo "  Test: $LOG_DIR/test_output.log"
echo ""

# Final summary
echo "=========================================="
if [ $TEST_EXIT_CODE -eq 0 ]; then
    echo -e "${GREEN}🎉 E2E Test Completed Successfully!${NC}"
else
    echo -e "${RED}❌ E2E Test Failed!${NC}"
fi
echo "=========================================="

exit $TEST_EXIT_CODE
