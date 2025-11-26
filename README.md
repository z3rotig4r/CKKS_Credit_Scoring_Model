# CKKS Credit Scoring System

Privacy-preserving credit scoring using homomorphic encryption (CKKS scheme).

## Overview

**완전동형암호(FHE) 기반 신용평가 시스템** - 사용자의 민감한 금융 정보가 암호화된 상태로 서버에서 처리되어 프라이버시를 완벽하게 보장합니다.

### Key Features
- 🔒 **End-to-End Encryption**: 모든 데이터가 암호화된 상태로 연산
- 🚀 **Real-time Inference**: 브라우저에서 직접 암호화/복호화 (WASM)
- 🎨 **Modern UI**: Tailwind CSS 기반 핀테크 디자인
- 🔐 **Secure Key Management**: PBKDF2 + AES-GCM 256-bit 암호화
- ⚡ **Performance Monitoring**: 실시간 타이밍 측정 (~1.7초)

### Tech Stack
- **Frontend**: React 19.2.0 + TypeScript + Tailwind CSS
- **Backend**: Go + Lattigo v6 (CKKS)
- **Security**: IndexedDB + Web Crypto API + 5-min idle timeout
- **Parameters**: LogN=14, LogQ=[60,40,40,60], Scale=2^40

## Architecture

![System Architecture](architecture.png)

**데이터 흐름**:
1. 사용자가 신용정보 입력 (나이, 소득, 신용기록 등)
2. WASM 모듈이 각 피처를 CKKS로 암호화
3. 암호화된 데이터를 백엔드로 전송
4. 백엔드가 암호화 상태에서 로지스틱 회귀 추론
5. 암호화된 결과를 클라이언트로 반환
6. WASM 모듈이 결과 복호화 → 신용점수 표시

## Quick Start

### 1. Build WASM Module
```bash
cd wasm && ./build.sh
```

### 2. Start Backend

#### Option A: HTTPS (Recommended)
```bash
cd backend
./generate_cert.sh  # Generate self-signed certificate
go run main.go      # Server starts with HTTPS on port 8080
```

⚠️ **Note**: Browsers will show security warnings for self-signed certificates. Click "Advanced" → "Proceed to localhost".

#### Option B: HTTP (Development only)
```bash
cd backend
rm -f server.crt server.key  # Remove certificates
go run main.go               # Server starts with HTTP
```

### 3. Start Frontend
```bash
cd frontend
npm install
npm start
```

Visit `http://localhost:3000`

## Features

- **6-feature logistic regression** on encrypted data
- **Multiple sigmoid approximations** (Chebyshev, Minimax, Composite)
- **TypeScript + Tailwind CSS** frontend
- **Real-time encrypted inference**

## Sigmoid Benchmarking

```bash
cd backend && go run cmd/benchmark/main.go
```

## Model Details

- **Inputs:** Age, Income, Credit History, Loan Amount, Employment, Debt-to-Income
- **Coefficients:** Placeholder values (replace with trained model)
- **Approximation:** Configurable (default: Chebyshev-5)
