# CKKS Credit Scoring System

Privacy-preserving credit scoring using homomorphic encryption (CKKS scheme).

## Architecture

```
┌─────────────┐         Encrypted Data        ┌─────────────┐
│   Frontend  │ ─────────────────────────────> │   Backend   │
│   (React)   │                                │   (Go)      │
│             │ <───────────────────────────── │             │
│  - Input UI │      Encrypted Score           │  - Logistic │
│  - WASM FHE │                                │    Regression│
│  - Encrypt  │                                │  - Sigmoid  │
│  - Decrypt  │                                │    Approx   │
└─────────────┘                                └─────────────┘
```

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
