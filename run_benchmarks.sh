#!/bin/bash

# ============================================================
# CKKS Credit Scoring - Complete Benchmark Suite
# ============================================================
# Tests all optimization techniques with proper isolation:
# 1. Baseline (LogN=14)
# 2. LogN=13 Optimization  
# 3. Sigmoid Approximations
# 
# Each test runs with its own backend build to ensure
# proper parameter matching.
# ============================================================

set -e  # Exit on error

RESULTS_DIR="./benchmark_results"
mkdir -p "$RESULTS_DIR"

echo "🧪 CKKS Credit Scoring - Complete Benchmark Suite"
echo "============================================================"
echo ""
echo "📁 Results directory: $RESULTS_DIR"
echo ""

# ============================================================
# Test 1: Baseline (LogN=14)
# ============================================================
echo "📊 Test 1/3: Baseline (LogN=14, 6 levels, 16384 slots)"
echo "------------------------------------------------------------"

echo "🔨 Building baseline backend..."
cd baseline/logn14

# Ensure go.mod exists
if [ ! -f "go.mod" ]; then
    cp ../../backend/go.mod .
    cp ../../backend/go.sum .
fi

go build -o server_baseline main_baseline.go
pkill -9 server_baseline 2>/dev/null || true
sleep 1
./server_baseline > /tmp/backend_baseline.log 2>&1 &
BACKEND_PID=$!
echo "✅ Backend started (PID: $BACKEND_PID)"
sleep 5
cd ../..

echo "🧪 Running baseline E2E test..."
cd baseline/logn14
timeout 120 go run e2e_baseline.go > "../../$RESULTS_DIR/1_baseline_logn14.txt" 2>&1 || true
cd ../..

echo "🛑 Stopping baseline backend..."
pkill -9 server_baseline 2>/dev/null || true
sleep 2

echo "✅ Baseline test completed"
echo ""

# ============================================================
# Test 2: LogN=13 Optimization
# ============================================================
echo "📊 Test 2/3: Optimized (LogN=13, 6 levels, 8192 slots)"
echo "------------------------------------------------------------"

echo "🔨 Building optimized backend..."
cd backend
go build -o server main.go
pkill -9 server 2>/dev/null || true
sleep 1
./server > /tmp/backend_optimized.log 2>&1 &
BACKEND_PID=$!
echo "✅ Backend started (PID: $BACKEND_PID)"
sleep 5
cd ..

echo "🧪 Running optimized E2E test..."
cd test
timeout 120 go run e2e.go > "../$RESULTS_DIR/2_optimized_logn13.txt" 2>&1 || true
cd ..

echo "✅ LogN=13 test completed"
echo ""

# ============================================================
# Test 3: Sigmoid Approximations
# ============================================================
echo "📊 Test 3/3: Sigmoid Approximation Comparison"
echo "------------------------------------------------------------"

echo "🧪 Running sigmoid benchmark..."
cd backend/cmd/benchmark
go run sigmoid_benchmark.go > "../../../$RESULTS_DIR/3_sigmoid_methods.txt" 2>&1
cd ../../..

echo "✅ Sigmoid benchmark completed"
echo ""

# ============================================================
# Cleanup & Summary
# ============================================================
echo "🛑 Stopping all backend servers..."
pkill -9 server 2>/dev/null || true
pkill -9 server_baseline 2>/dev/null || true
sleep 1

echo ""
echo "============================================================"
echo "📈 Benchmark Results Summary"
echo "============================================================"
echo ""
echo "Results saved to: $RESULTS_DIR/"
echo ""
ls -lh "$RESULTS_DIR/" 2>/dev/null || echo "No results files found"
echo ""
echo "============================================================"
echo "✅ All benchmarks completed!"
echo "============================================================"
echo ""
echo "Next step: Run visualization script"
echo "  python3 visualize_benchmark.py"
echo ""
