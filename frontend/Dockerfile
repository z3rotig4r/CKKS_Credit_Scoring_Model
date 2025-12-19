# Frontend Dockerfile for CKKS Credit Scoring
# Context: project root (set in docker-compose.yml)

# Stage 1: Build WASM module (requires Go)
FROM golang:1.22-alpine AS wasm-builder

WORKDIR /wasm

# Copy WASM source code
COPY wasm/go.mod wasm/go.sum ./
RUN go mod download

COPY wasm/main.go ./

# Build WASM module with optimizations
RUN GOOS=js GOARCH=wasm go build -ldflags="-s -w" -trimpath -o main.wasm main.go

# Copy wasm_exec.js from Go SDK
RUN cp "$(go env GOROOT)/misc/wasm/wasm_exec.js" ./wasm_exec.js

# Stage 2: Build React app
FROM node:20-alpine AS app-builder

WORKDIR /app

# Copy package files from frontend directory
COPY frontend/package*.json ./

# Install dependencies
RUN npm ci --silent

# Copy frontend source code
COPY frontend/src ./src
COPY frontend/public ./public
COPY frontend/tsconfig.json ./
COPY frontend/postcss.config.js ./
COPY frontend/tailwind.config.js ./

# Copy WASM files from wasm-builder stage
COPY --from=wasm-builder /wasm/main.wasm ./public/
COPY --from=wasm-builder /wasm/wasm_exec.js ./public/

# Build React app
ENV NODE_ENV=production
RUN npm run build

# Stage 3: Production (Nginx)
FROM nginx:alpine

# Copy custom nginx config
COPY frontend/nginx.conf /etc/nginx/conf.d/default.conf

# Copy built app
COPY --from=app-builder /app/build /usr/share/nginx/html

# Expose port
EXPOSE 3000

# Health check (use 127.0.0.1 instead of localhost to avoid IPv6 issues)
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://127.0.0.1:3000 || exit 1

CMD ["nginx", "-g", "daemon off;"]
