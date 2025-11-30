# Frontend Dockerfile for CKKS Credit Scoring
# Build stage
FROM node:20-alpine AS builder

WORKDIR /app

# Copy package files
COPY package*.json ./

# Install dependencies
RUN npm ci --silent

# Copy source code
COPY . .

# Build WASM (requires Go)
FROM golang:1.22-alpine AS wasm-builder

WORKDIR /wasm

# Copy WASM source
COPY --from=builder /app/public/wasm_exec.js ./
COPY ../wasm /wasm

# Build WASM module
RUN GOOS=js GOARCH=wasm go build -o main.wasm main.go

# Final build stage
FROM node:20-alpine AS app-builder

WORKDIR /app

# Copy dependencies from builder
COPY --from=builder /app/node_modules ./node_modules
COPY --from=builder /app/package*.json ./

# Copy source and WASM
COPY --from=builder /app/src ./src
COPY --from=builder /app/public ./public
COPY --from=wasm-builder /wasm/main.wasm ./public/

# Build React app
ENV NODE_ENV=production
RUN npm run build

# Production stage
FROM nginx:alpine

# Copy custom nginx config
COPY nginx.conf /etc/nginx/conf.d/default.conf

# Copy built app
COPY --from=app-builder /app/build /usr/share/nginx/html

# Expose port
EXPOSE 3000

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:3000 || exit 1

CMD ["nginx", "-g", "daemon off;"]
