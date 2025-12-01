# Web Worker 및 Zero-Copy 최적화 가이드

## 개요

이 문서는 CKKS 신용평가 시스템에 도입된 Web Worker와 SharedArrayBuffer(Zero-Copy) 최적화에 대해 설명합니다.

## 최적화 목표

### 1. Web Worker 도입
- **목적**: UI 블로킹 제거
- **방법**: 암호화 연산을 백그라운드 스레드로 분리
- **이점**: 
  - 메인 스레드 응답성 유지
  - 사용자 경험 개선 (버튼, 입력 필드 등이 멈추지 않음)
  - 진행 상태를 실시간으로 표시 가능

### 2. SharedArrayBuffer (Zero-Copy)
- **목적**: 메모리 복사 오버헤드 제거
- **방법**: JS와 WASM 간 메모리 직접 공유
- **이점**:
  - 데이터 전송 시간 감소
  - 메모리 사용량 최적화
  - 대용량 데이터 처리 시 성능 향상

## 구현 세부사항

### 아키텍처

```
┌─────────────────────────────────────────────────────────────┐
│                    Main Thread (UI)                         │
├─────────────────────────────────────────────────────────────┤
│  React Components                                           │
│  ├─ KeyManagement                                           │
│  ├─ EncryptPanel                                            │
│  └─ WorkerDemo (성능 비교)                                   │
│                                                             │
│  FHEContext (State Management)                              │
│  └─ WorkerManager (Worker 통신)                             │
└─────────────────┬───────────────────────────────────────────┘
                  │ postMessage / SharedArrayBuffer
                  ↓
┌─────────────────────────────────────────────────────────────┐
│                  Web Worker Thread                          │
├─────────────────────────────────────────────────────────────┤
│  crypto.worker.js                                           │
│  ├─ WASM Module (Go/Lattigo)                                │
│  ├─ 키 생성 (비동기)                                         │
│  ├─ 암호화 (진행 상태 리포트)                                │
│  └─ 복호화                                                   │
│                                                             │
│  SharedArrayBuffer (Zero-Copy)                              │
│  └─ 메모리 직접 공유 (복사 없음)                             │
└─────────────────────────────────────────────────────────────┘
```

### 핵심 컴포넌트

#### 1. crypto.worker.js
```javascript
// Worker 초기화
self.addEventListener('message', async (event) => {
  const { type, data, id } = event.data;
  
  switch (type) {
    case 'GENERATE_KEYS':
      // 키 생성 (백그라운드)
      const keys = await window.generateKeys();
      
      if (data.useSharedMemory) {
        // Zero-Copy: SharedArrayBuffer에 직접 쓰기
        sharedView.set(new Uint8Array(keys.secretKey), offset);
      } else {
        // 일반: postMessage로 전송 (복사)
        self.postMessage({ keys }, [keys.secretKey]);
      }
      break;
  }
});
```

#### 2. workerManager.js
```javascript
class WorkerManager {
  async generateKeys(useSharedMemory = true) {
    const result = await this.sendMessage({
      type: 'GENERATE_KEYS',
      data: { useSharedMemory }
    });
    
    if (result.useSharedMemory) {
      // Zero-Copy: SharedArrayBuffer에서 읽기
      return sharedView.slice(offset, offset + length);
    } else {
      // 일반: 전달받은 데이터 사용
      return result.keys;
    }
  }
}
```

#### 3. setupProxy.js (개발 서버)
```javascript
// SharedArrayBuffer 지원을 위한 보안 헤더
module.exports = function(app) {
  app.use((req, res, next) => {
    res.setHeader('Cross-Origin-Opener-Policy', 'same-origin');
    res.setHeader('Cross-Origin-Embedder-Policy', 'require-corp');
    next();
  });
};
```

#### 4. nginx.conf (프로덕션)
```nginx
# SharedArrayBuffer support
add_header Cross-Origin-Opener-Policy "same-origin" always;
add_header Cross-Origin-Embedder-Policy "require-corp" always;
```

## SharedArrayBuffer 요구사항

### 보안 헤더

SharedArrayBuffer는 Spectre 취약점 완화를 위해 다음 헤더가 필요합니다:

```http
Cross-Origin-Opener-Policy: same-origin
Cross-Origin-Embedder-Policy: require-corp
```

### 브라우저 지원

| 브라우저 | 최소 버전 | 지원 여부 |
|---------|----------|----------|
| Chrome | 92+ | ✅ |
| Firefox | 79+ | ✅ |
| Safari | 15.2+ | ✅ |
| Edge | 92+ | ✅ |

**확인 방법**:
```javascript
const supported = typeof SharedArrayBuffer !== 'undefined';
console.log('SharedArrayBuffer:', supported ? '✅' : '❌');
```

## 성능 비교

### 예상 성능 개선

#### Web Worker (UI 응답성)
- **Main Thread**: UI 블로킹 (~33ms 동안 응답 없음)
- **Web Worker**: UI 항상 응답 (0ms 블로킹)
- **개선**: ∞ (질적 개선)

#### Zero-Copy (데이터 전송)
- **일반 복사**: 5개 특성 × 770KB = 3.85MB 복사 (~10-15ms)
- **Zero-Copy**: 0ms (메모리 직접 공유)
- **개선**: 10-15ms 절약

### 실제 측정 (예상)

| 방법 | 키 생성 | 암호화(5개) | 전송 오버헤드 | 총 시간 | UI 응답성 |
|------|---------|------------|--------------|---------|----------|
| **Main Thread** | 29ms | 33ms | 0ms | 62ms | ❌ 블로킹 |
| **Worker (Copy)** | 29ms | 33ms | ~5ms | 67ms | ✅ 응답 |
| **Worker (Zero-Copy)** | 29ms | 33ms | ~0ms | 62ms | ✅ 응답 |

**주요 발견**:
1. **Worker 오버헤드**: 초기화 및 메시지 전달로 5-10ms 추가
2. **Zero-Copy 이득**: 대용량 데이터에서 효과적 (작은 데이터는 미미)
3. **UI 응답성**: 최대 이점 (숫자로 측정 불가)

## 사용 방법

### 1. WorkerDemo 컴포넌트

성능 비교 데모:

```jsx
import WorkerDemo from './components/WorkerDemo';

function App() {
  return <WorkerDemo />;
}
```

기능:
- Main Thread vs Worker 비교
- Zero-Copy On/Off 전환
- 실시간 로그 및 진행 상태
- 성능 측정 및 분석

### 2. FHEContext에서 Worker 사용

```jsx
const { encrypt, encryptMultiple } = useFHE();

// 자동으로 최적 방법 선택
// - Worker 사용 가능 시: Worker 사용
// - SharedArrayBuffer 지원 시: Zero-Copy 사용
const encrypted = await encryptMultiple([0.5, 0.75, 0.3]);
```

### 3. 수동 제어

```javascript
import { workerManager } from './services/workerManager';

// Worker 초기화
await workerManager.initialize();

// SharedArrayBuffer 초기화
await workerManager.initializeSharedMemory();

// 암호화 (Zero-Copy)
const result = await workerManager.encryptFeatures(
  features,
  publicKey,
  true  // useSharedMemory
);

console.log('Zero-Copy:', result.zeroCopy);
console.log('Time:', result.duration);
```

## 디버깅

### SharedArrayBuffer 지원 확인

브라우저 콘솔:
```javascript
// 1. SharedArrayBuffer 존재 확인
console.log(typeof SharedArrayBuffer !== 'undefined');

// 2. 보안 헤더 확인
fetch(window.location.href).then(r => {
  console.log('COOP:', r.headers.get('Cross-Origin-Opener-Policy'));
  console.log('COEP:', r.headers.get('Cross-Origin-Embedder-Policy'));
});

// 3. Worker 통신 확인
const worker = new Worker('/crypto.worker.js');
worker.postMessage({ type: 'CHECK_SHARED_MEMORY', data: { size: 1024 } });
worker.addEventListener('message', (e) => {
  console.log('Worker response:', e.data);
});
```

### 일반적인 문제

#### 1. SharedArrayBuffer is not defined

**원인**: 보안 헤더 누락

**해결**:
- 개발: `setupProxy.js` 확인
- 프로덕션: `nginx.conf`에 COOP/COEP 헤더 추가
- HTTPS 사용 (HTTP는 지원 안 됨)

#### 2. Worker 초기화 실패

**원인**: WASM 파일 경로 오류

**해결**:
```javascript
// crypto.worker.js에서 절대 경로 사용
importScripts('/wasm_exec.js');
fetch('/main.wasm');
```

#### 3. postMessage 전송 실패

**원인**: Transferable objects 사용 시 원본 invalidation

**해결**:
```javascript
// 전송 후 원본 사용 금지
self.postMessage({ data }, [data.buffer]);  // data는 이제 사용 불가
```

## 최적화 체크리스트

### 개발 환경
- [ ] `setupProxy.js` COOP/COEP 헤더 설정
- [ ] `crypto.worker.js` 생성 및 WASM 로드
- [ ] `workerManager.js` 통신 로직 구현
- [ ] `WorkerDemo.jsx` 데모 컴포넌트

### 프로덕션 환경
- [ ] `nginx.conf` COOP/COEP 헤더 추가
- [ ] HTTPS 인증서 설정 (필수)
- [ ] Worker 파일 정적 서빙 확인
- [ ] 브라우저 호환성 테스트

### 성능 검증
- [ ] Worker vs Main Thread 벤치마크
- [ ] Zero-Copy vs Copy 벤치마크
- [ ] UI 응답성 테스트 (클릭, 스크롤)
- [ ] 메모리 사용량 프로파일링

## 향후 개선 사항

### 1. Worklet API
```javascript
// Audio/Animation Worklet과 결합
// 더 세밀한 스레드 제어
```

### 2. OffscreenCanvas
```javascript
// 시각화를 Worker에서 수행
// UI 렌더링 성능 개선
```

### 3. WebGPU
```javascript
// GPU 가속 암호화 연산
// CKKS NTT를 GPU에서 수행
```

## 참고 자료

- [MDN: Web Workers](https://developer.mozilla.org/en-US/docs/Web/API/Web_Workers_API)
- [MDN: SharedArrayBuffer](https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Global_Objects/SharedArrayBuffer)
- [COOP/COEP Headers](https://web.dev/coop-coep/)
- [Transferable Objects](https://developer.mozilla.org/en-US/docs/Web/API/Web_Workers_API/Transferable_objects)

## 요약

**Web Worker 최적화**:
- ✅ UI 블로킹 제거 (질적 개선)
- ✅ 진행 상태 실시간 표시
- ⚠️ 5-10ms 오버헤드 (작은 데이터)

**Zero-Copy 최적화**:
- ✅ 메모리 복사 제거 (10-15ms 절약)
- ✅ 대용량 데이터에서 효과적
- ⚠️ HTTPS + 보안 헤더 필수
- ⚠️ 브라우저 호환성 고려

**권장사항**:
- UI 응답성이 중요한 경우: **Web Worker 사용**
- 대용량 데이터 처리 시: **Zero-Copy 추가**
- 작은 데이터셋: **Main Thread 유지** (단순)
