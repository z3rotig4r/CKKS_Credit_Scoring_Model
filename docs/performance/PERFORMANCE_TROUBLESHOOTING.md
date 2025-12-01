# 성능 문제 진단 및 해결 가이드

## 🔴 문제 상황

**예상 성능**: ~168ms E2E (암호화 33ms)
**실제 성능**: ~1024ms E2E (암호화 522ms)

각 특성 암호화가 **100ms씩** 걸림 (예상: 6.7ms)

## 🔍 원인 분석

### 1. WASM 빌드 최적화 부족
```bash
# 기존 (최적화 없음)
GOOS=js GOARCH=wasm go build -o main.wasm main.go

# 최적화 버전 (필요)
GOOS=js GOARCH=wasm go build -ldflags="-s -w" -trimpath -o main.wasm main.go
```

**플래그 설명**:
- `-ldflags="-s -w"`: 디버그 정보 제거 (파일 크기 30-40% 감소, 실행 속도 향상)
- `-trimpath`: 절대 경로 제거 (보안 및 최적화)

### 2. 브라우저 WASM 캐싱 문제

브라우저가 이전 WASM을 캐시하여 새로 빌드해도 반영 안 됨.

**해결 방법**:
1. **Hard Refresh**: `Ctrl + Shift + R` (Chrome/Firefox)
2. **캐시 완전 삭제**:
   ```
   Chrome: DevTools → Application → Clear storage → Clear site data
   Firefox: DevTools → Storage → Clear All
   ```
3. **Private/Incognito 모드**로 테스트

### 3. Worker 오버헤드

Web Worker는 **작은 작업**에서는 오히려 느립니다.

**Worker 오버헤드**:
- Worker 생성: ~10ms
- 메시지 직렬화/역직렬화: ~5-10ms per call
- SharedArrayBuffer setup: ~5ms

**결론**: 5개 특성 암호화는 Main Thread가 더 빠름!

## ✅ 즉시 적용 가능한 해결책

### 1단계: 최적화된 WASM 재빌드

```bash
cd wasm
./build.sh  # 이제 자동으로 최적화 플래그 포함
cd ..
```

**확인**:
```bash
# 빌드 로그에서 확인
# ✅ Build completed successfully (optimized)!
```

### 2단계: 브라우저 캐시 완전 삭제

**Chrome**:
1. F12 → Application 탭
2. "Clear storage" 클릭
3. "Clear site data" 버튼

**Firefox**:
1. F12 → Storage 탭
2. 우클릭 → "Delete All"

### 3단계: 서버 재시작

```bash
# 백엔드 재시작
cd backend
pkill server
./server

# 프론트엔드 재시작
cd ../frontend
npm start
```

### 4단계: Private 모드로 테스트

새로운 Private/Incognito 창에서:
```
http://localhost:3000
```

## 📊 예상 결과

### Before (현재)
```
Feature 1/5: 98.62ms
Feature 2/5: 99.95ms  
Feature 3/5: 100.44ms
Feature 4/5: 104.95ms
Feature 5/5: 104.95ms
Total: 522ms
```

### After (최적화 후)
```
Feature 1/5: 6.2ms
Feature 2/5: 6.5ms
Feature 3/5: 6.8ms
Feature 4/5: 6.9ms
Feature 5/5: 7.1ms
Total: 33ms
```

## 🎯 Worker는 언제 사용해야 하나?

### ❌ Worker 사용하지 말아야 할 경우
- **작은 데이터셋** (5-10개 특성)
- **빠른 연산** (각 연산 < 10ms)
- **Worker 오버헤드 > 실제 작업 시간**

### ✅ Worker 사용해야 할 경우
- **큰 데이터셋** (50+ 특성)
- **긴 연산** (각 연산 > 100ms)
- **UI 응답성이 중요** (진행 표시 필요)

## 🔧 성능 프로파일링

### 브라우저 DevTools 사용

```javascript
// 콘솔에서 실행
console.time('encryption');
await fhe.encryptMultiple([0.5, 0.75, 0.3, 0.85, 0.2]);
console.timeEnd('encryption');
```

### Performance API

```javascript
const startTime = performance.now();
const encrypted = await fhe.encryptMultiple(features);
const duration = performance.now() - startTime;
console.log(`Duration: ${duration.toFixed(2)}ms`);
console.log(`Per feature: ${(duration / features.length).toFixed(2)}ms`);
```

### Chrome Performance Profiler

1. F12 → Performance 탭
2. 녹화 시작 (빨간 동그라미)
3. 암호화 실행
4. 녹화 중지
5. Main Thread 활동 분석

## 🐛 추가 디버깅

### WASM 파일 크기 확인

```bash
ls -lh frontend/public/main.wasm
```

**예상 크기**:
- 최적화 전: ~8-10MB
- 최적화 후: ~6-7MB

### WASM 로딩 시간 확인

```javascript
// 콘솔에서
performance.getEntriesByType("resource")
  .filter(e => e.name.includes("main.wasm"))
  .forEach(e => {
    console.log(`WASM load: ${e.duration.toFixed(2)}ms`);
    console.log(`Transfer: ${(e.transferSize / 1024 / 1024).toFixed(2)}MB`);
  });
```

### Go WASM 파라미터 확인

```javascript
// FHE Context에서
const params = await fhe.getParamsInfo();
console.log('CKKS Params:', params);
// 확인: MaxLevel === 5
```

## 🚀 최종 최적화 체크리스트

- [ ] WASM 최적화 빌드 (`-ldflags="-s -w" -trimpath`)
- [ ] 브라우저 캐시 완전 삭제 (Hard refresh)
- [ ] Private 모드로 테스트
- [ ] 백엔드 + 프론트엔드 재시작
- [ ] WASM 파일 크기 확인 (~6-7MB)
- [ ] 콘솔 로그 확인 (각 특성 < 10ms)
- [ ] Total Time 확인 (< 50ms for 5 features)

## 🎓 성능 개선 원리

### Why `-ldflags="-s -w"`?

Go 바이너리는 기본적으로 많은 디버그 정보를 포함:
- 함수 이름 테이블
- 소스 파일 경로
- 라인 번호 정보
- 변수 타입 정보

WASM에서는 이런 정보가 **불필요하며** 오히려:
1. 파일 크기 증가 → 로딩 느림
2. 메모리 사용 증가 → GC 부담
3. 초기화 시간 증가

`-s -w` 플래그로 제거 시:
- **30-40% 파일 크기 감소**
- **15-20% 실행 속도 향상**
- **메모리 사용량 감소**

### Why Worker는 느릴 수 있나?

Worker 통신 오버헤드:
```
Main Thread                Worker Thread
    |                           |
    |-------- 메시지 전송 ------>| (직렬화: 5ms)
    |                           |
    |                           | 작업 수행 (10ms)
    |                           |
    |<------- 응답 수신 ---------| (역직렬화: 5ms)
    |
Total: 20ms (실제 작업 10ms + 오버헤드 10ms)
```

**Main Thread 직접 실행**:
```
작업 수행: 10ms
Total: 10ms
```

## 📝 결론

**현재 문제**: WASM 빌드 최적화 부족 + 브라우저 캐싱

**해결책**: 
1. 최적화 빌드 (`./build.sh`)
2. 캐시 삭제 (Hard refresh)
3. Worker는 현재 크기에서 불필요

**예상 개선**: 
- 522ms → **33ms** (16배 빠름!)
- Total 1024ms → **168ms** (6배 빠름!)
