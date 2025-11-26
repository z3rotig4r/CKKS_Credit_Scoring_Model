#!/bin/bash

# Lattigo Wasm 빌드 스크립트

set -e

echo "🔨 Building Lattigo Wasm module..."

# wasm_exec.js 복사
echo "📋 Copying wasm_exec.js..."
cp "$(go env GOROOT)/lib/wasm/wasm_exec.js" ../frontend/public/

# Wasm 빌드
echo "🚀 Compiling Go to WebAssembly..."
GOOS=js GOARCH=wasm go build -o ../frontend/public/main.wasm main.go

echo "✅ Build completed successfully!"
echo "📦 Output files:"
echo "   - ../frontend/public/main.wasm"
echo "   - ../frontend/public/wasm_exec.js"
