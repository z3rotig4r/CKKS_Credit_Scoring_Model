#!/bin/bash

# Lattigo Wasm 빌드 스크립트

set -e

echo "🔨 Building Lattigo Wasm module..."

# wasm_exec.js 복사
echo "📋 Copying wasm_exec.js..."
cp "$(go env GOROOT)/lib/wasm/wasm_exec.js" ../frontend/public/

# Wasm 빌드
echo "🚀 Compiling Go to WebAssembly..."

# Check if SIMD flag is set
if [ "$1" = "simd" ]; then
  echo "⚡ Building with WASM optimizations (satconv,signext)..."
  GOOS=js GOARCH=wasm GOWASM=satconv,signext go build -ldflags="-s -w" -trimpath -o ../frontend/public/main.wasm main.go
  echo "✅ Build completed successfully with optimizations!"
  echo "ℹ️  Note: Full SIMD support requires browser with WASM SIMD enabled"
else
  echo "📦 Building optimized WASM with release flags..."
  GOOS=js GOARCH=wasm go build -ldflags="-s -w" -trimpath -o ../frontend/public/main.wasm main.go
  echo "✅ Build completed successfully (optimized)!"
  echo "💡 Use './build.sh simd' for additional SIMD optimizations"
fi
echo "📦 Output files:"
echo "   - ../frontend/public/main.wasm"
echo "   - ../frontend/public/wasm_exec.js"
