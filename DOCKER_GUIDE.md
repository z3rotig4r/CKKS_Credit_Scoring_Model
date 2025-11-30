# Docker 배포 가이드

완전동형암호(CKKS) 신용평가 시스템의 Docker 컨테이너화 및 배포 가이드입니다.

## 목차

1. [빠른 시작](#빠른-시작)
2. [아키텍처](#아키텍처)
3. [이미지 세부사항](#이미지-세부사항)
4. [환경 변수](#환경-변수)
5. [네트워크 구성](#네트워크-구성)
6. [프로덕션 배포](#프로덕션-배포)
7. [트러블슈팅](#트러블슈팅)

## 빠른 시작

### Option 1: 자동 배포 스크립트

```bash
./deploy.sh
```

이 스크립트는:
- ✅ Docker 및 Docker Compose 설치 확인
- ✅ 이미지 자동 빌드
- ✅ 서비스 자동 시작
- ✅ Health check 수행
- ✅ 접속 URL 표시

### Option 2: Makefile 명령어

```bash
# 빌드 + 실행 (한 번에)
make quick-start

# 단계별 실행
make docker-build    # 이미지 빌드
make docker-up       # 서비스 시작
make docker-logs     # 로그 확인
make docker-down     # 서비스 중지
```

### Option 3: Docker Compose 직접 사용

```bash
# 빌드 및 시작
docker-compose up -d

# 로그 확인
docker-compose logs -f

# 중지
docker-compose down
```

## 아키텍처

### 전체 구조

```
┌────────────────────────────────────────────────────────┐
│              Docker Host (Linux/Mac/Windows)           │
├────────────────────────────────────────────────────────┤
│                                                        │
│  ┌──────────────────────────────────────────────────┐ │
│  │           Bridge Network: ckks-network           │ │
│  ├──────────────────────────────────────────────────┤ │
│  │                                                  │ │
│  │  ┌────────────────┐      ┌──────────────────┐  │ │
│  │  │   Frontend     │      │     Backend      │  │ │
│  │  │   Container    │ ───▶ │    Container     │  │ │
│  │  ├────────────────┤      ├──────────────────┤  │ │
│  │  │ Nginx:alpine   │      │ golang:1.22      │  │ │
│  │  │ + React build  │      │ + Lattigo        │  │ │
│  │  │ + WASM files   │      │ + CKKS engine    │  │ │
│  │  │                │      │                  │  │ │
│  │  │ Port: 3000     │      │ Port: 8080       │  │ │
│  │  └────────────────┘      └──────────────────┘  │ │
│  │         ▲                                      │ │
│  └─────────┼──────────────────────────────────────┘ │
│            │                                        │
└────────────┼────────────────────────────────────────┘
             │
             │ Port Mapping
             │ 3000:3000, 8080:8080
             ▼
       User Browser
    http://localhost:3000
```

### 컨테이너 통신

1. **외부 → Frontend**: `localhost:3000` → Nginx 서버
2. **Frontend → Backend**: `/api/*` → Nginx proxy → `backend:8080`
3. **Backend 내부**: CKKS 연산 수행, 결과 반환

## 이미지 세부사항

### Backend Image

**Dockerfile 분석**:

```dockerfile
# Stage 1: Builder
FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo -o server main.go

# Stage 2: Runtime
FROM alpine:latest
WORKDIR /root/
COPY --from=builder /app/server .
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=3s CMD wget --spider http://localhost:8080/health
CMD ["./server"]
```

**특징**:
- **Multi-stage build**: 최종 이미지 크기 최소화 (~55MB)
- **CGO 활성화**: Lattigo의 NTT 최적화 지원
- **Alpine Linux**: 경량 베이스 이미지
- **Health check**: 자동 상태 모니터링

**빌드 시간**:
- 첫 빌드: ~5분 (의존성 다운로드 포함)
- 재빌드: ~30초 (캐시 활용)

### Frontend Image

**Dockerfile 분석**:

```dockerfile
# Stage 1: Node Builder
FROM node:20-alpine AS builder
WORKDIR /app
COPY package*.json ./
RUN npm ci --silent
COPY . .

# Stage 2: WASM Builder
FROM golang:1.22-alpine AS wasm-builder
WORKDIR /wasm
COPY ../wasm .
RUN GOOS=js GOARCH=wasm go build -o main.wasm main.go

# Stage 3: React Builder
FROM node:20-alpine AS app-builder
COPY --from=builder /app/node_modules ./node_modules
COPY --from=wasm-builder /wasm/main.wasm ./public/
RUN npm run build

# Stage 4: Production
FROM nginx:alpine
COPY nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=app-builder /app/build /usr/share/nginx/html
EXPOSE 3000
CMD ["nginx", "-g", "daemon off;"]
```

**특징**:
- **4-stage build**: WASM → React → Nginx
- **정적 파일 서빙**: Nginx로 최적화된 성능
- **WASM 통합**: Go 코드가 브라우저에서 실행
- **Reverse proxy**: `/api/*` 요청을 백엔드로 전달

**빌드 시간**:
- 첫 빌드: ~8분 (npm 의존성 + WASM 컴파일)
- 재빌드: ~1분 (캐시 활용)

## 환경 변수

### Backend 환경 변수

| 변수 | 기본값 | 설명 |
|------|--------|------|
| `GO_ENV` | `production` | Go 실행 환경 |
| `BACKEND_PORT` | `8080` | 서버 포트 |
| `LOG_LEVEL` | `info` | 로그 레벨 (debug/info/warn/error) |
| `GOMAXPROCS` | (자동) | Go 워커 스레드 수 |

### Frontend 환경 변수

| 변수 | 기본값 | 설명 |
|------|--------|------|
| `NODE_ENV` | `production` | React 빌드 모드 |
| `REACT_APP_API_URL` | `http://localhost:8080` | 백엔드 API URL |

### 환경 변수 설정 방법

**Option 1: `.env` 파일** (권장)

```bash
# .env.example을 복사
cp .env.example .env

# .env 파일 편집
vim .env
```

**Option 2: `docker-compose.yml` 수정**

```yaml
services:
  backend:
    environment:
      - GO_ENV=production
      - LOG_LEVEL=debug
```

**Option 3: 실행 시 직접 지정**

```bash
GO_ENV=production docker-compose up -d
```

## 네트워크 구성

### Bridge Network

Docker Compose가 자동으로 `ckks-network`라는 bridge 네트워크를 생성합니다.

**특징**:
- 컨테이너 간 DNS 자동 해석
- 외부와 격리된 내부 네트워크
- 서비스 이름으로 통신 가능 (`backend`, `frontend`)

**네트워크 검사**:

```bash
# 네트워크 목록
docker network ls

# 상세 정보
docker network inspect ckks-network

# 연결된 컨테이너 확인
docker network inspect ckks-network | grep -A 10 "Containers"
```

### Nginx Reverse Proxy

Frontend의 Nginx가 `/api/*` 요청을 백엔드로 프록시합니다.

**설정** (`frontend/nginx.conf`):

```nginx
location /api {
    proxy_pass http://backend:8080;
    proxy_http_version 1.1;
    proxy_set_header Host $host;
    proxy_read_timeout 300s;
}
```

**장점**:
- CORS 문제 해결
- 단일 Origin (localhost:3000)
- API 요청 투명한 전달

## 프로덕션 배포

### 1. HTTPS 설정

**Let's Encrypt 인증서 발급**:

```bash
# Certbot 설치
sudo apt-get install certbot

# 인증서 발급
sudo certbot certonly --standalone -d yourdomain.com

# 인증서 위치
/etc/letsencrypt/live/yourdomain.com/fullchain.pem
/etc/letsencrypt/live/yourdomain.com/privkey.pem
```

**docker-compose.yml 수정**:

```yaml
services:
  frontend:
    volumes:
      - /etc/letsencrypt:/etc/letsencrypt:ro
    environment:
      - ENABLE_HTTPS=true
```

**nginx.conf 수정**:

```nginx
server {
    listen 443 ssl;
    ssl_certificate /etc/letsencrypt/live/yourdomain.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/yourdomain.com/privkey.pem;
    
    # ... 나머지 설정
}
```

### 2. 리소스 제한

**docker-compose.prod.yml**:

```yaml
version: '3.8'

services:
  backend:
    deploy:
      resources:
        limits:
          cpus: '2'
          memory: 4G
        reservations:
          cpus: '1'
          memory: 2G
    restart: always
  
  frontend:
    deploy:
      resources:
        limits:
          cpus: '1'
          memory: 1G
        reservations:
          cpus: '0.5'
          memory: 512M
    restart: always
```

**실행**:

```bash
docker-compose -f docker-compose.yml -f docker-compose.prod.yml up -d
```

### 3. 로드 밸런싱

**다중 백엔드 인스턴스**:

```yaml
services:
  backend:
    deploy:
      replicas: 3
```

**Nginx 로드 밸런서**:

```nginx
upstream backend {
    server backend-1:8080;
    server backend-2:8080;
    server backend-3:8080;
}

location /api {
    proxy_pass http://backend;
}
```

### 4. 모니터링

**Prometheus + Grafana 추가**:

```yaml
services:
  prometheus:
    image: prom/prometheus
    ports:
      - "9090:9090"
    volumes:
      - ./prometheus.yml:/etc/prometheus/prometheus.yml
  
  grafana:
    image: grafana/grafana
    ports:
      - "3001:3000"
    depends_on:
      - prometheus
```

## 트러블슈팅

### 이미지 빌드 실패

**문제**: `ERROR [builder 5/6] RUN go build -o server main.go`

**해결**:
```bash
# 캐시 없이 재빌드
docker-compose build --no-cache backend

# 개별 Dockerfile 테스트
cd backend
docker build -t test-backend .
```

### 컨테이너 시작 실패

**문제**: `Error starting userland proxy: listen tcp4 0.0.0.0:8080: bind: address already in use`

**해결**:
```bash
# 포트 사용 중인 프로세스 확인
lsof -i :8080
sudo lsof -i :8080

# 프로세스 종료
kill -9 <PID>

# 또는 docker-compose.yml에서 포트 변경
ports:
  - "8081:8080"  # 외부:내부
```

### Health Check 실패

**문제**: 컨테이너가 `unhealthy` 상태

**진단**:
```bash
# 컨테이너 로그 확인
docker-compose logs backend

# 컨테이너 내부 접속
docker exec -it ckks-backend sh

# Health endpoint 수동 테스트
wget -O- http://localhost:8080/health
curl http://localhost:8080/health
```

**해결**:
```bash
# Health check 설정 확인
docker inspect ckks-backend | grep -A 10 "Healthcheck"

# Health check 일시 비활성화 (디버깅용)
docker-compose up -d --no-healthcheck
```

### 네트워크 연결 문제

**문제**: Frontend → Backend 통신 실패

**진단**:
```bash
# 네트워크 존재 확인
docker network ls

# 컨테이너가 네트워크에 연결되었는지 확인
docker network inspect ckks-network

# Frontend에서 Backend ping 테스트
docker exec -it ckks-frontend sh
ping backend
wget -O- http://backend:8080/health
```

**해결**:
```bash
# 네트워크 재생성
docker-compose down
docker network rm ckks-network
docker-compose up -d
```

### WASM 로딩 실패

**문제**: 브라우저에서 "WASM 파일을 찾을 수 없습니다"

**진단**:
```bash
# Frontend 컨테이너 파일 확인
docker exec -it ckks-frontend sh
ls -lh /usr/share/nginx/html/
ls -lh /usr/share/nginx/html/main.wasm

# Nginx 로그 확인
docker-compose logs frontend | grep wasm
```

**해결**:
```bash
# WASM 빌드 다시 실행
cd wasm
./build.sh

# Frontend 재빌드
docker-compose build --no-cache frontend
docker-compose up -d frontend
```

### 로그 분석

**전체 로그**:
```bash
docker-compose logs
docker-compose logs -f          # Follow mode
docker-compose logs --tail=100  # 마지막 100줄
```

**특정 서비스**:
```bash
docker-compose logs backend
docker-compose logs backend -f --tail=50
```

**시간 필터**:
```bash
docker-compose logs --since 30m          # 최근 30분
docker-compose logs --since 2024-11-30   # 특정 날짜 이후
```

### 성능 문제

**컨테이너 리소스 사용 확인**:
```bash
# 실시간 모니터링
docker stats

# 특정 컨테이너
docker stats ckks-backend
docker stats ckks-frontend
```

**리소스 부족 시**:
```bash
# Docker Desktop 설정에서 리소스 증가
# Settings → Resources → Advanced
# - CPUs: 4+
# - Memory: 8GB+
# - Disk: 60GB+
```

## 추가 명령어

### 데이터 백업

```bash
# 볼륨 백업 (필요시)
docker run --rm -v ckks_backend-data:/data -v $(pwd):/backup alpine tar czf /backup/backup.tar.gz /data
```

### 이미지 export/import

```bash
# 이미지 저장
docker save ckks-backend:latest | gzip > backend.tar.gz
docker save ckks-frontend:latest | gzip > frontend.tar.gz

# 이미지 로드
docker load < backend.tar.gz
docker load < frontend.tar.gz
```

### 컨테이너 디버깅

```bash
# 실행 중인 컨테이너에서 명령 실행
docker exec ckks-backend ps aux
docker exec ckks-backend netstat -tuln

# 파일 복사
docker cp ckks-backend:/root/server ./server.backup
docker cp config.json ckks-backend:/root/config.json
```

## 참고 자료

- [Docker Documentation](https://docs.docker.com/)
- [Docker Compose Documentation](https://docs.docker.com/compose/)
- [Nginx Docker Official Image](https://hub.docker.com/_/nginx)
- [Go Docker Official Image](https://hub.docker.com/_/golang)
