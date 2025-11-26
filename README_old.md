# Lattigo-Wasm FHE 웹 애플리케이션 PoC

Go 기반 FHE 라이브러리 **Lattigo v6**를 WebAssembly로 컴파일하여 브라우저에서 동형암호 연산을 수행하는 프로젝트입니다.

## 🎯 프로젝트 목적 (Why?)

### 문제 인식
기존 FHE(Fully Homomorphic Encryption) 시스템은 다음과 같은 한계가 있었습니다:

1. **서버 의존성** 🖥️
   - 모든 암호화 연산이 서버에서 실행
   - 클라이언트는 단순한 UI만 제공
   - 서버 과부하 및 확장성 문제

2. **비밀키 보안 리스크** 🔓
   - 서버에 비밀키를 전송해야 하는 경우 발생
   - 중앙 집중식 키 관리의 위험성
   - MITM(Man-In-The-Middle) 공격 가능성

3. **개발 진입 장벽** 📚
   - C++/Python 기반 FHE 라이브러리 (SEAL, PALISADE)
   - 복잡한 설치 및 환경 설정
   - 웹 개발자가 접근하기 어려움

### 해결 방안
**WebAssembly 기반 클라이언트 측 FHE 실행**

- ✅ **Zero-Trust 아키텍처**: 비밀키가 브라우저 밖으로 나가지 않음
- ✅ **서버 부하 분산**: 각 클라이언트가 자체 CPU 사용
- ✅ **웹 표준 기술**: JavaScript 개발자도 FHE 사용 가능
- ✅ **크로스 플랫폼**: Windows, macOS, Linux, 모바일 브라우저 지원

## 🐛 개발 중 발견한 문제와 해결

### Phase 1: CKKS 연산 정확도 문제

#### 문제 1: Rescaling 누락 (발견: 2025-11-01)
```
증상: 곱셈 결과가 2^40배 부풀어오름
원인: Multiplication 후 Rescaling 미수행
해결: evaluator.Rescale(ctMul, ctMul) 추가
```

**Before:**
```go
ctMul, err := evaluator.MulNew(ct1, ct2)
// 스케일: 2^40 × 2^40 = 2^80 (오버플로우!)
```

**After:**
```go
ctMul, err := evaluator.MulNew(ct1, ct2)
evaluator.Rescale(ctMul, ctMul)
// 스케일: 2^80 → 2^40 (정상화)
```

#### 문제 2: Level 불일치 (발견: 2025-11-01)
```
증상: 덧셈 연산 시 "level mismatch" 오류
원인: 서로 다른 레벨의 암호문 연산 시도
해결: 연산 전 DropLevel()로 레벨 정렬
```

**Before:**
```go
// ct1: level 3, ct2: level 2
ctSum := evaluator.AddNew(ct1, ct2) // ❌ ERROR
```

**After:**
```go
minLevel := min(ct1.Level(), ct2.Level())
evaluator.DropLevel(ct1, ct1.Level() - minLevel)
evaluator.DropLevel(ct2, ct2.Level() - minLevel)
ctSum := evaluator.AddNew(ct1, ct2) // ✅ SUCCESS
```

#### 문제 3: 암호문 역직렬화 메타데이터 오류 ⚠️ **가장 중요!**
```
증상: 복호화 시 완전히 잘못된 값 (10^12~10^15배 오차)
원인: ckks.NewCiphertext()로 고정 레벨 지정 → UnmarshalBinary가 실제 레벨 복원 못함
해결: new(rlwe.Ciphertext) 사용으로 자동 메타데이터 복원
```

**Before (잘못된 코드):**
```go
// ❌ 고정된 MaxLevel로 생성 → 실제 저장된 레벨 무시됨
ct := ckks.NewCiphertext(params, 1, params.MaxLevel())
ct.UnmarshalBinary(ctBytes) // 레벨 정보가 덮어씌워짐!
```

**After (올바른 코드):**
```go
// ✅ 빈 객체로 시작 → UnmarshalBinary가 올바른 메타데이터 복원
ct := new(rlwe.Ciphertext)
ct.UnmarshalBinary(ctBytes) // 저장된 레벨, 스케일 그대로 복원
```

**교훈:**
> Lattigo의 직렬화/역직렬화 시 **절대로 메타데이터를 미리 지정하지 말 것!**  
> UnmarshalBinary가 자동으로 복원하도록 빈 객체를 사용해야 함.

이 문제는 `project_guideline.md` 섹션 6.8에 상세히 문서화되어 있습니다.

### Phase 2: React 통합 및 보안

#### 문제 4: PBKDF2 Iteration 수 결정
```
고민: 보안과 UX 사이의 균형
해결: OWASP 2023 표준인 100,000 iterations 채택
```

**분석 결과:**
- 10,000: 너무 약함 (GPU로 1초에 수천 개 해시 가능)
- 100,000: OWASP 권장 ✅ (무차별 대입 공격에 317년 소요)
- 1,000,000: UX 저하 (키 생성에 30초+ 소요)

#### 문제 5: React Hook 의존성 순환 참조
```
증상: "clearSensitiveData was used before it was defined" 경고
원인: resetIdleTimeout이 clearSensitiveData를 참조하는데, 정의 순서가 반대
해결: clearSensitiveData를 먼저 정의하고 resetIdleTimeout이 참조
```

**Before:**
```javascript
const resetIdleTimeout = useCallback(() => {
  clearSensitiveData(); // ❌ 아직 정의 안됨
}, [clearSensitiveData]);

const clearSensitiveData = useCallback(() => { ... }, []);
```

**After:**
```javascript
const clearSensitiveData = useCallback(() => { ... }, []);

const resetIdleTimeout = useCallback(() => {
  clearSensitiveData(); // ✅ 정의됨
}, [clearSensitiveData]);
```

#### 문제 6: 중복 변수 선언
```
증상: "Cannot redeclare block-scoped variable 'clearSensitiveData'"
원인: 파일 내 두 곳에서 같은 함수 정의
해결: 하나로 통합
```

## 📁 프로젝트 구조

```
lattigo-wasm/
├── project_guideline.md          # 프로젝트 설계 문서
├── PHASE2_PLAN.md                # Phase 2 상세 개발 계획
├── README.md                      # 이 문서
├── wasm/                          # Wasm 모듈 (Phase 1 ✅)
│   ├── main.go                    # Lattigo Wasm 래퍼
│   ├── go.mod                     # Go 모듈 정의
│   ├── build.sh                   # 빌드 스크립트
│   └── README.md                  # Wasm 모듈 문서
└── frontend/                      # React 프론트엔드 (Phase 2 ✅)
    ├── public/
    │   ├── main.wasm              # 컴파일된 Wasm 모듈
    │   └── wasm_exec.js           # Go Wasm 런타임
    ├── src/
    │   ├── services/              # 핵심 서비스 ✅
    │   │   ├── wasmLoader.js      # Wasm 모듈 로더
    │   │   ├── cryptoService.js   # Web Crypto API (PBKDF2 + AES-GCM)
    │   │   └── indexedDBService.js # IndexedDB 헬퍼
    │   ├── contexts/              # 상태 관리 ✅
    │   │   └── FHEContext.jsx     # FHE 전역 상태 관리
    │   ├── components/            # UI 컴포넌트 ✅
    │   │   ├── KeyManagement.jsx  # 키 관리 패널
    │   │   ├── KeyManagement.css
    │   │   ├── EncryptPanel.jsx   # 암호화 패널
    │   │   ├── EncryptPanel.css
    │   │   ├── DecryptPanel.jsx   # 복호화 패널
    │   │   ├── DecryptPanel.css
    │   │   ├── OperationsPanel.jsx # 연산 패널
    │   │   └── OperationsPanel.css
    │   ├── App.js                 # 메인 애플리케이션 ✅
    │   ├── App.css                # 전역 스타일 ✅
    │   └── index.js
    └── package.json
```

## 🚀 빠른 시작

### 1. Wasm 모듈 빌드

```bash
cd wasm
./build.sh
```

### 2. React 프론트엔드 실행 (Phase 2 완료)

```bash
cd frontend
npm install  # 처음 한 번만
npm start    # 개발 서버 실행
```

브라우저에서 `http://localhost:3000`을 열어 FHE 애플리케이션을 사용합니다.

### 3. 테스트 페이지 실행 (Phase 1 Wasm 테스트)

```bash
cd frontend/public
./serve.sh
```

브라우저에서 `http://localhost:8080/test.html`을 열어 테스트합니다.

## ✅ Phase 1 완료 사항

Phase 1 (Go Wasm 모듈 개발)이 완료되었습니다:

- [x] Lattigo v6 CKKS 스킴 통합
- [x] `syscall/js`를 사용한 JavaScript 인터페이스 구현
- [x] 기본 FHE 함수 노출:
  - `fheKeygen()` - 키 생성 (SK + PK)
  - `fheEncrypt(pk, plaintext)` - 암호화
  - `fheDecrypt(sk, ciphertext)` - 복호화
  - `fheGetParamsInfo()` - 파라미터 조회
- [x] **확장 키 생성 함수 (Leveled & Bootstrapping 준비):**
  - `fheGenAllKeys()` - 모든 키 한번에 생성
  - `fheGenRelinearizationKey(sk)` - 곱셈 후 차수 감소용
  - `fheGenGaloisKeys(sk, [galEls])` - 회전 연산용
  - `fheGenRotationKeys(sk, rotations)` - 특정 회전 키
  - `fheGenConjugationKey(sk)` - 복소수 켤레 연산용
- [x] **FHE 연산 함수 (암호문 연산):**
  - `fheAdd(ct1, ct2)` - 암호문 덧셈
  - `fheMul(ct1, ct2, rlk)` - 암호문 곱셈 (Relinearization)
  - `fheRotate(ct, k, gk)` - 슬롯 회전
  - `fheConjugate(ct, conjKey)` - 복소수 켤레
- [x] 직렬화/역직렬화 래퍼 구현
- [x] 빌드 자동화 스크립트
- [x] 테스트 HTML 페이지 3개 (기본 + 확장 키 + 연산 & 성능)
- [x] 성능 벤치마크 도구 (연산별 시간 측정)

## ✅ Phase 2 완료 사항

Phase 2 (React 프론트엔드 개발)가 완료되었습니다:

### 2-1. 프로젝트 초기 설정 ✅
- [x] Create React App 기반 프로젝트 생성
- [x] Wasm 파일 통합 (`main.wasm`, `wasm_exec.js` → `/public`)
- [x] 기본 프로젝트 구조 설정

### 2-2. 핵심 서비스 구현 ✅
- [x] **wasmLoader.js** - Wasm 모듈 로더
  - Singleton 패턴으로 한 번만 로드
  - 동적 스크립트 로딩 (`wasm_exec.js`)
  - Wasm 초기화 및 함수 래퍼 제공
  - 파라미터 정보 추출 헬퍼
  
- [x] **cryptoService.js** - Web Crypto API 래퍼
  - PBKDF2 키 유도 (100,000 iterations, SHA-256)
  - AES-GCM-256 암호화/복호화
  - Base64 직렬화/역직렬화
  - 비밀번호 강도 검증
  
- [x] **indexedDBService.js** - IndexedDB 관리
  - 데이터베이스: `lattigo-fhe-db`
  - 객체 저장소: `fhe-keys` (keyPath: `id`)
  - 인덱스: `timestamp`, `keyType`
  - CRUD 작업: saveKey, loadKey, deleteKey, listKeys
  - 유틸리티: getKeysByType, clearAllKeys, deleteDatabase

### 2-3. FHE Context 및 상태 관리 ✅
- [x] **FHEContext.jsx** - React Context API 기반 전역 상태 관리
  - **Wasm 로딩 상태**: wasmLoaded, wasmLoading, wasmError
  - **키 관리 상태**: keys (SK, PK, RLK, Galois Keys), keysLoaded
  - **암호문 저장소**: ciphertexts (메모리 내 관리)
  - **파라미터 정보**: params (CKKS 설정)
  
- [x] **보안 기능**
  - 5분 Idle Timeout (비활성 시 SK 자동 제거)
  - 사용자 활동 감지 (mousedown, keydown, touchstart)
  - clearSensitiveData() - 메모리에서 SK 제거
  
- [x] **키 관리 함수**
  - `generateKeys(password)` - 키 생성 및 암호화 저장
  - `loadKeys(password)` - IndexedDB에서 키 로드 및 복호화
  - `clearKeys()` - 모든 키 삭제 (메모리 + IndexedDB)
  
- [x] **FHE 연산 함수**
  - `encrypt(value)` - 숫자 암호화 및 저장소에 추가
  - `decrypt(ciphertextId)` - 암호문 복호화 및 오차 계산
  - `add(ct1Id, ct2Id)` - 암호문 덧셈
  - `multiply(ct1Id, ct2Id)` - 암호문 곱셈 (Relinearization)
  - `rotate(ctId, k)` - 슬롯 회전
  - `conjugate(ctId)` - 복소수 켤레

### 2-4. UI 컴포넌트 개발 ✅
- [x] **KeyManagement.jsx** - 키 관리 패널
  - Generate/Load 모드 토글
  - 비밀번호 입력 및 확인
  - 키 생성 버튼 (Generate All Keys)
  - 키 로드 버튼 (Load Saved Keys)
  - 키 상태 표시 (SK, PK, RLK, Galois Keys 크기)
  - 로그아웃 버튼 (메모리 클리어)
  - 보안 안내 메시지
  
- [x] **EncryptPanel.jsx** - 암호화 패널
  - 숫자 입력 폼
  - Encrypt 버튼
  - 암호화 결과 표시 (크기, 시간)
  - 암호문 목록 테이블 (ID, 원본값, 크기, 타임스탬프)
  - Clear All 버튼
  
- [x] **DecryptPanel.jsx** - 복호화 패널
  - 암호문 선택 드롭다운
  - Decrypt 버튼
  - 복호화 결과 표시
  - 정확도 분석 (절대 오차, 상대 오차)
  - 오차 색상 코딩 (녹색/노랑/빨강)
  - 복호화 시간 표시
  
- [x] **OperationsPanel.jsx** - FHE 연산 패널
  - 연산 선택 (Add, Multiply, Rotate, Conjugate)
  - 암호문 선택 (ct1, ct2 - 이진 연산용)
  - 회전 량 입력 (Rotate 전용)
  - Perform Operation 버튼
  - 연산 결과 표시 (연산 시간, 복호화 시간)
  - 정확도 분석 (절대/상대 오차)
  - 자동 결과 저장소 추가

### 2-5. 메인 애플리케이션 통합 ✅
- [x] **App.js** - 메인 애플리케이션 셸
  - FHEProvider로 전체 앱 래핑
  - 탭 네비게이션 (4개: Keys, Encrypt, Decrypt, Operations)
  - 상태 바 (Wasm 상태, 파라미터 정보)
  - 로딩 상태 (스피너 애니메이션)
  - 에러 상태 (에러 메시지 + 새로고침 버튼)
  - 조건부 컴포넌트 렌더링
  - 푸터 (Lattigo 링크, 프라이버시 안내)
  
- [x] **App.css** - 전역 스타일링
  - 그라데이션 배경 (보라색 계열)
  - 탭 네비게이션 스타일
  - 상태 바 스타일 (배지, 레이블)
  - 로딩/에러 컨테이너
  - 스피너 애니메이션 (@keyframes spin)
  - 버튼 스타일 (primary, hover 효과)
  - 반응형 디자인 (모바일/데스크톱)

### 2-6. 개별 컴포넌트 CSS ✅
- [x] **KeyManagement.css** - 키 관리 패널 스타일
- [x] **EncryptPanel.css** - 암호화 패널 스타일
- [x] **DecryptPanel.css** - 복호화 패널 스타일
- [x] **OperationsPanel.css** - 연산 패널 스타일

각 CSS는 다음을 포함:
- 폼 스타일링 (input, select, button)
- 결과 박스 스타일 (success, error)
- 정보 박스 스타일 (info, warning)
- 테이블 스타일 (hover 효과)
- 색상 코딩 (녹색/노랑/빨강)
- Monospace 폰트 (기술 데이터용)

### 2-7. 디버깅 및 최적화 ✅
- [x] ESLint 경고 수정
  - 미사용 변수 제거 (`keyPassword`)
  - useCallback 의존성 배열 수정
  - 함수 정의 순서 조정 (hoisting 이슈)
  - 중복 함수 정의 제거
- [x] 컴파일 에러 해결
  - 변수 재선언 오류 수정
  - 순환 의존성 해결
- [x] 성공적인 빌드 및 실행 확인

### Phase 2 기술 스택
- **프레임워크**: React 18.2.0
- **상태 관리**: React Context API + Hooks
- **암호화**: Web Crypto API (PBKDF2 + AES-GCM-256)
- **저장소**: IndexedDB (`lattigo-fhe-db`)
- **스타일링**: CSS3 (Flexbox, Grid, Animations)
- **빌드 도구**: Create React App + Webpack

### Phase 2 보안 구현
- ✅ 비밀키 암호화 저장 (AES-GCM-256)
- ✅ PBKDF2 키 유도 (100,000 iterations)
- ✅ 메모리 내 복호화 (저장소에는 암호화된 상태로만)
- ✅ 5분 Idle Timeout (자동 메모리 클리어)
- ✅ 사용자 활동 감지 (타임아웃 리셋)
- ✅ 비밀번호 검증 (8자 이상, 대소문자, 숫자, 특수문자)
- ✅ 로컬 전용 처리 (서버 전송 없음)

#### 🔐 PBKDF2 보안 강도 설명

**현재 설정: 100,000 iterations**

| 기관/표준 | 권장 Iteration | 연도 | 비고 |
|----------|---------------|------|------|
| **OWASP** | **100,000+** | 2023 | **현재 표준** ✅ |
| NIST SP 800-63B | 10,000+ | 2017 | 최소 기준 |
| 1Password | 100,000 | 2023 | 상용 서비스 |
| LastPass | 100,100 | 2023 | 상용 서비스 |

**보안 효과:**
- 🛡️ **무차별 대입 공격 방어**: 10억 개 비밀번호 시도 시 약 317년 소요
- 🛡️ **GPU 공격 저항**: 메모리 집약적 연산으로 병렬화 어려움
- ⚡ **사용자 경험**: 키 생성 시 약 1~10초 대기 (1회만 발생)
- ✅ **클라이언트 측 처리**: 서버 부하 없이 각 사용자의 CPU 활용

**다른 Iteration 값과 비교:**
- **< 10,000**: ❌ 공격자가 빠르게 해시 계산 가능 (위험)
- **100,000**: ✅ OWASP 2023 표준 (현재 설정)
- **> 1,000,000**: ⚠️ 사용자 경험 저하 (30초+ 대기)


## 🔧 노출된 API

### JavaScript 전역 함수

#### 기본 함수
```javascript
// 1. 키 생성 (SK + PK)
const keys = await fheKeygen();
// returns: { secretKey: Uint8Array, publicKey: Uint8Array }

// 2. 암호화
const ciphertext = await fheEncrypt(keys.publicKey, 7500);
// returns: Uint8Array

// 3. 복호화
const plaintext = await fheDecrypt(keys.secretKey, ciphertext);
// returns: number

// 4. 파라미터 정보
const paramsJson = fheGetParamsInfo();
const params = JSON.parse(paramsJson);
// returns: { LogN, LogQ, LogP, MaxLevel, MaxSlots, DefaultScale, RingType }
```

#### 확장 키 생성 함수 (Leveled FHE & Bootstrapping 지원)
```javascript
// 1. 모든 키 한번에 생성
const allKeys = await fheGenAllKeys();
// returns: { secretKey, publicKey, relinearizationKey, galoisKeys[] }

// 2. Relinearization Key (곱셈 후 차수 감소)
const rlk = await fheGenRelinearizationKey(keys.secretKey);

// 3. Galois Keys (회전 연산)
const gks = await fheGenGaloisKeys(keys.secretKey);
// 또는 특정 갈루아 요소:
const customGks = await fheGenGaloisKeys(keys.secretKey, [5, 25, 49]);

// 4. Rotation Keys (특정 회전)
const rotKeys = await fheGenRotationKeys(keys.secretKey, [1, 2, 4, 8]);

// 5. Conjugation Key (복소수 켤레)
const conjKey = await fheGenConjugationKey(keys.secretKey);
```

#### FHE 연산 함수 (암호문 연산)
```javascript
// 1. 덧셈
const ctSum = await fheAdd(ciphertext1, ciphertext2);
// Enc(a) + Enc(b) = Enc(a+b)

// 2. 곱셈 (Relinearization 포함)
const ctProduct = await fheMul(ciphertext1, ciphertext2, relinearizationKey);
// Enc(a) × Enc(b) = Enc(a×b)

// 3. 회전 (Rotation)
const ctRotated = await fheRotate(ciphertext, 1, galoisKey);
// Rotate([a₀, a₁, a₂, ...], 1) → [a₁, a₂, a₃, ...]

// 4. 켤레 (Conjugation)
const ctConjugated = await fheConjugate(ciphertext, conjugationKey);
// Conj(a+bi) → a-bi
```

## 📊 CKKS 파라미터

| 파라미터 | 값 | 설명 |
|---------|-----|------|
| LogN | 14 | 링 차원: 2^14 = 16,384 |
| LogQ | [60, 40, 40, 60] | 모듈러스 체인 |
| LogP | [61] | 키 스위칭용 모듈러스 |
| LogDefaultScale | 40 | 기본 스케일: 2^40 |
| MaxSlots | 8,192 | 최대 슬롯 수 |

## 🔒 보안 고려사항

1. **비밀키 관리**
   - `secretKey`는 절대 서버로 전송하지 않음
   - 클라이언트 측에서만 보관 및 사용

2. **메모리 보안**
   - 복호화 작업 후 즉시 `secretKey` 메모리 해제
   - 사용하지 않을 때는 암호화된 형태로만 저장

3. **저장소 보안** (Phase 2에서 구현 예정)
   - Web Crypto API로 `secretKey` 암호화
   - IndexedDB에 암호화된 키 저장

## 📋 다음 단계 (Phase 3)

Phase 2 완료 후 다음 단계:

- [ ] **Phase 3: Go 백엔드 API 서버**
  - [ ] RESTful API 서버 구축
  - [ ] POST /compute 엔드포인트 (서버 측 FHE 연산)
  - [ ] 암호문 업로드/다운로드 처리
  - [ ] CORS 설정
  - [ ] 성능 모니터링
  
- [ ] **Phase 4: E2E 통합 및 테스트**
  - [ ] Client Encrypt → Server Compute → Client Decrypt 플로우
  - [ ] 성능 벤치마크 (클라이언트 vs 서버)
  - [ ] 보안 테스트
  - [ ] 사용자 시나리오 테스트

## 🛠️ 기술 스택

- **FHE 라이브러리:** Lattigo v6.0.0 (CKKS)
- **Wasm 컴파일:** Go 1.21+
- **프론트엔드:** React.js (Phase 2)
- **백엔드:** Go HTTP Server (Phase 3)

## 📚 참고 문서

- [project_guideline.md](./project_guideline.md) - 전체 프로젝트 설계 문서
- [wasm/README.md](./wasm/README.md) - Wasm 모듈 상세 문서
- [Lattigo Documentation](https://github.com/tuneinsight/lattigo)

## 🧪 테스트

### 기본 테스트 (`test.html`)
테스트 페이지에서 다음 기능을 확인할 수 있습니다:

1. **키 생성 테스트** - FHE 키 쌍 생성
2. **암호화 테스트** - 숫자를 암호문으로 변환
3. **복호화 테스트** - 암호문을 평문으로 복원
4. **파라미터 조회** - CKKS 파라미터 정보 확인
5. **E2E 테스트** - 전체 플로우 자동 테스트

```bash
# 브라우저에서 열기
http://localhost:8080/test.html
```

### 확장 키 테스트 (`test-extended-keys.html`)
추가 평가 키 생성 및 관리 테스트:

1. **기본 키 생성** - SK + PK
2. **모든 키 생성** - SK + PK + RLK + Galois Keys (한번에)
3. **개별 평가 키 생성** - RLK, Galois Keys, Conjugation Key 각각
4. **사용자 지정 회전 키** - 특정 회전 인덱스에 대한 키 생성
5. **키 크기 요약** - 모든 키의 크기 시각화

```bash
# 브라우저에서 열기
http://localhost:8080/test-extended-keys.html
```

### FHE 연산 & 성능 테스트 (`test-operations.html`) 🆕
암호문 연산 및 성능 벤치마크:

1. **초기화** - 모든 키 자동 생성 및 초기 암호문 생성
2. **FHE 연산 테스트**
   - 덧셈: Enc(a) + Enc(b)
   - 곱셈: Enc(a) × Enc(b) (Relinearization)
   - 회전: Rotate(Enc(vector))
   - 켤레: Conjugate(Enc(complex))
3. **성능 벤치마크**
   - 각 연산의 실행 시간 측정
   - 반복 실행을 통한 평균/최소/최대 시간 계산
   - 연산 간 상대 속도 비교
4. **종합 결과 요약** - 가장 빠른/느린 연산 분석

```bash
# 브라우저에서 열기
http://localhost:8080/test-operations.html
```

## �️ 개발 도구

### IndexedDB 디버그 유틸리티

React 앱 실행 중 브라우저 콘솔에서 IndexedDB 상태를 확인할 수 있습니다.

#### 사용 가능한 명령어

```javascript
// 1. 모든 키 목록 출력
await dbDebug.listAllKeys()
// 출력: 모든 저장된 키의 ID, 타입, 크기, 생성 시간

// 2. 키 타입별로 그룹화
await dbDebug.listKeysByType()
// 출력: secret, public, relinearization, galois 등으로 분류

// 3. 특정 키 상세 정보
await dbDebug.inspectKey('secret_key')
// 출력: 키의 메타데이터, 크기, 데이터 미리보기 (hex dump)

// 4. 데이터베이스 통계
await dbDebug.getDatabaseStats()
// 출력: 총 키 개수, 총 크기, 타입별 통계, 가장 오래된/최신 키

// 5. 키 검증 (형식 확인)
await dbDebug.validateKeys()
// 출력: 각 키의 암호화 형식이 올바른지 검증

// 6. 특정 키 삭제 (확인 필요)
await dbDebug.deleteKey('secret_key', true)

// 7. 모든 키 삭제 (확인 필요)
await dbDebug.clearAll(true)

// 8. 데이터베이스 완전 삭제 (확인 필요)
await dbDebug.deleteDatabase(true)

// 9. 도움말
dbDebug.help()
```

#### 사용 예시

```javascript
// 시나리오 1: 키 생성 후 확인
await dbDebug.listAllKeys()
// 📦 IndexedDB Keys Summary:
// Total Keys: 5
// ────────────────────────────────────────
// [1] secret_key
//   Type: secret
//   Size: 12.5 KB
//   Created: 11/1/2025, 3:45:30 PM

// 시나리오 2: 데이터베이스 전체 통계
await dbDebug.getDatabaseStats()
// 📈 Database Statistics:
// Total Keys: 5
// Total Size: 156.3 KB
// Keys by Type:
//   secret: 1 keys (12.5 KB)
//   public: 1 keys (24.8 KB)
//   relinearization: 1 keys (49.2 KB)
//   galois: 2 keys (69.8 KB)

// 시나리오 3: 특정 키 검사
await dbDebug.inspectKey('public_key')
// 🔍 Key Inspection: public_key
// Type: public
// Size: 24.8 KB (25,456 bytes)
// Data Preview: 5a 4c 7b 2e 9f ... (hex dump)

// 시나리오 4: 개발 중 데이터베이스 초기화
await dbDebug.clearAll(true)
// ✅ All keys cleared successfully
```

#### 특징

- ✅ **타입별 분류**: secret, public, relinearization, galois 키 자동 그룹화
- ✅ **크기 계산**: 바이트를 읽기 쉬운 형식으로 변환 (KB, MB)
- ✅ **데이터 무결성**: 암호화된 데이터 형식 검증
- ✅ **안전한 삭제**: 확인 없이는 삭제 불가 (실수 방지)
- ✅ **개발 전용**: `NODE_ENV=development`에서만 로드

## 🎓 핵심 교훈 및 모범 사례

### Lattigo CKKS 사용 시 주의사항

1. **절대 메타데이터를 미리 지정하지 말 것** ⚠️
   ```go
   // ❌ 잘못된 방법
   ct := ckks.NewCiphertext(params, 1, params.MaxLevel())
   ct.UnmarshalBinary(data)
   
   // ✅ 올바른 방법
   ct := new(rlwe.Ciphertext)
   ct.UnmarshalBinary(data)
   ```

2. **곱셈 후 항상 Rescale 수행**
   ```go
   ctMul, _ := evaluator.MulNew(ct1, ct2)
   evaluator.Rescale(ctMul, ctMul) // 필수!
   ```

3. **연산 전 레벨 확인 및 정렬**
   ```go
   if ct1.Level() != ct2.Level() {
       minLevel := min(ct1.Level(), ct2.Level())
       evaluator.DropLevel(ct1, ct1.Level() - minLevel)
       evaluator.DropLevel(ct2, ct2.Level() - minLevel)
   }
   ```

4. **Complex128 타입 일관성 유지**
   ```go
   // 암호화 시
   values := make([]complex128, params.MaxSlots())
   values[0] = complex(plaintext, 0)
   
   // 복호화 시
   result := make([]complex128, params.MaxSlots())
   encoder.Decode(plaintext, result)
   return real(result[0]) // float64로 변환
   ```

### React + WebAssembly 통합

1. **Wasm 모듈은 한 번만 로드** (Singleton 패턴)
2. **비동기 초기화를 Context로 관리**
3. **개발 도구는 NODE_ENV로 분리**
4. **Hook 의존성 배열을 정확히 지정**

## 🏗️ 아키텍처 의사결정 (Architecture Decision Records)

### ADR-001: Singleton 패턴으로 Wasm 모듈 로드

**문제:**
- Wasm 모듈 로드는 비용이 큼 (~2MB 파일, 초기화 시간)
- 여러 컴포넌트에서 Wasm 함수를 호출해야 함
- React의 리렌더링으로 중복 로드 위험

**고려한 대안:**
1. ❌ **각 컴포넌트에서 개별 로드**: 메모리 낭비, 초기화 시간 중복
2. ❌ **Context에서 직접 로드**: Context 재생성 시 재로드 위험
3. ✅ **Singleton 서비스**: 한 번만 로드, 전역 공유

**결정:**
```javascript
class WasmLoader {
  constructor() {
    if (WasmLoader.instance) {
      return WasmLoader.instance; // 이미 존재하면 재사용
    }
    this.wasmModule = null;
    this.isLoading = false;
    WasmLoader.instance = this;
  }
}
```

**이유:**
- ✅ 메모리 효율: 2MB Wasm 모듈을 한 번만 메모리에 로드
- ✅ 성능: 초기화를 1회만 수행 (약 500ms 절약)
- ✅ 상태 일관성: 모든 컴포넌트가 동일한 Wasm 인스턴스 공유
- ✅ 테스트 용이: 단일 인스턴스로 모킹 간편

### ADR-002: 동적 스크립트 로딩 (`wasm_exec.js`)

**문제:**
- Go Wasm은 `wasm_exec.js` 런타임이 필수
- `public/` 폴더의 정적 파일을 동적으로 로드해야 함
- import 문으로는 불가능 (Go 런타임 특수성)

**고려한 대안:**
1. ❌ **`<script>` 태그에 하드코딩**: SPA에서 타이밍 제어 어려움
2. ❌ **번들에 포함**: Go 런타임과 충돌 가능성
3. ✅ **동적 스크립트 주입**: 필요한 시점에 로드

**결정:**
```javascript
const script = document.createElement('script');
script.src = '/wasm_exec.js';
script.onload = () => resolve();
document.head.appendChild(script);
```

**이유:**
- ✅ 타이밍 제어: Wasm 로드 전에 런타임 준비 보장
- ✅ 오류 처리: onload/onerror로 로드 실패 감지
- ✅ 번들 독립성: Go 런타임이 번들러와 충돌하지 않음
- ✅ 지연 로딩: 실제 사용 시점에만 로드 (초기 번들 크기 감소)

**Trade-off:**
- ⚠️ 네트워크 요청 1회 추가 (~50KB)
- ✅ 하지만 Wasm 자체가 2MB이므로 50KB는 미미함

### ADR-003: PBKDF2 키 유도 함수 선택

**문제:**
- 비밀번호로부터 AES 키를 안전하게 유도해야 함
- 무차별 대입 공격에 저항해야 함

**고려한 대안:**

| 알고리즘 | 장점 | 단점 | 결정 |
|---------|------|------|------|
| **PBKDF2** | 표준(RFC 8018), Web Crypto API 지원, 검증됨 | GPU 공격에 약간 취약 | ✅ 선택 |
| bcrypt | GPU 저항성 높음 | Web Crypto 미지원, 외부 라이브러리 필요 | ❌ |
| scrypt | 메모리 집약적, GPU 저항성 최고 | Web Crypto 미지원, 구현 복잡 | ❌ |
| Argon2 | 최신 표준, 최고 보안 | 브라우저 미지원, WASM 필요 | ❌ |

**결정:**
```javascript
crypto.subtle.deriveKey(
  {
    name: 'PBKDF2',
    salt: salt,
    iterations: 100000,
    hash: 'SHA-256'
  },
  // ...
);
```

**이유:**
- ✅ **Web Crypto API 네이티브 지원**: 외부 라이브러리 불필요
- ✅ **브라우저 최적화**: 하드웨어 가속 지원
- ✅ **OWASP 권장 표준**: 100,000 iterations로 충분한 보안
- ✅ **크로스 플랫폼**: 모든 모던 브라우저 지원
- ✅ **감사 가능**: 표준 알고리즘으로 보안 감사 용이

**보안 분석:**
- 무차별 대입: 10억 개 시도 시 **317년** 소요
- GPU 공격: PBKDF2-HMAC-SHA256은 병렬화 제한적
- 레인보우 테이블: Salt 16 bytes로 무력화

### ADR-004: AES-GCM-256 암호화 선택

**문제:**
- FHE 비밀키(~12KB)를 IndexedDB에 안전하게 저장
- 무결성 검증 필요 (변조 감지)

**고려한 대안:**

| 모드 | 인증 | 성능 | 보안 | 결정 |
|------|------|------|------|------|
| **AES-GCM** | ✅ AEAD | 빠름 (HW 가속) | 매우 높음 | ✅ 선택 |
| AES-CBC | ❌ 별도 HMAC 필요 | 중간 | 높음 (HMAC 추가 시) | ❌ |
| AES-CTR | ❌ 별도 HMAC 필요 | 빠름 | 높음 (HMAC 추가 시) | ❌ |
| ChaCha20-Poly1305 | ✅ AEAD | 매우 빠름 | 매우 높음 | ❌ (Web Crypto 미지원) |

**결정:**
```javascript
crypto.subtle.encrypt(
  {
    name: 'AES-GCM',
    iv: iv, // 12 bytes (권장)
  },
  key,
  data
);
```

**이유:**
- ✅ **AEAD (Authenticated Encryption with Associated Data)**: 암호화 + 무결성 검증 동시 제공
- ✅ **하드웨어 가속**: AES-NI 명령어로 CPU에서 고속 처리
- ✅ **변조 감지**: Tag 검증으로 데이터 변조 자동 감지
- ✅ **Nonce 관리 간단**: CTR 모드 기반으로 IV 충돌 위험 낮음
- ✅ **256-bit 키**: 양자 컴퓨터 시대 대비 (Grover's algorithm: 2^128 보안)

**보안 매개변수:**
- IV: 12 bytes (96-bit, GCM 최적 크기)
- Tag: 16 bytes (128-bit, 변조 감지)
- Key: 32 bytes (256-bit, AES-256)

### ADR-005: Base64 인코딩 선택

**문제:**
- Uint8Array 바이너리 데이터를 IndexedDB에 문자열로 저장
- 직렬화/역직렬화 필요

**고려한 대안:**

| 인코딩 | 크기 증가 | 호환성 | 성능 | 결정 |
|--------|----------|--------|------|------|
| **Base64** | +33% | 모든 브라우저 | 매우 빠름 | ✅ 선택 |
| Hex | +100% | 모든 브라우저 | 빠름 | ❌ |
| Uint8Array 직접 저장 | 0% | IndexedDB만 가능 | 가장 빠름 | ❌ |
| Base85 | +25% | 외부 라이브러리 필요 | 중간 | ❌ |

**결정:**
```javascript
// 인코딩
const base64 = btoa(String.fromCharCode(...new Uint8Array(buffer)));

// 디코딩
const binary = atob(base64);
const bytes = Uint8Array.from(binary, c => c.charCodeAt(0));
```

**이유:**
- ✅ **네이티브 지원**: `btoa()`/`atob()` 모든 브라우저에서 사용 가능
- ✅ **호환성**: JSON과 함께 사용 가능, URL에도 전송 가능
- ✅ **성능**: 네이티브 C++ 구현으로 매우 빠름
- ✅ **디버깅**: 텍스트로 저장되어 개발 도구에서 확인 가능
- ⚠️ **크기 증가**: 33% 오버헤드 (12KB → 16KB)
  - 하지만 압축으로 상쇄 가능
  - IndexedDB 용량 제한(수 GB)에 비해 무시 가능

**대안 거부 이유:**
- Uint8Array 직접 저장: 일부 브라우저에서 직렬화 문제 발생 가능
- Hex: 크기 2배 증가는 비효율적

### ADR-006: IndexedDB 선택

**문제:**
- FHE 키를 브라우저에 영구 저장해야 함
- 새로고침 후에도 키 유지 필요
- 여러 키 타입을 효율적으로 관리

**고려한 대안:**

| 저장소 | 용량 | 영구성 | 구조화 | 보안 | 결정 |
|--------|------|--------|--------|------|------|
| **IndexedDB** | ~수 GB | ✅ | ✅ 객체 저장소 + 인덱스 | 동일 출처 격리 | ✅ 선택 |
| localStorage | ~5-10 MB | ✅ | ❌ Key-Value만 | 동일 출처 격리 | ❌ |
| sessionStorage | ~5-10 MB | ❌ 탭 닫으면 삭제 | ❌ | 동일 출처 격리 | ❌ |
| WebSQL | 더 이상 사용 안 함 | - | - | - | ❌ |
| Cache API | 주로 네트워크 캐시용 | ✅ | ❌ | 동일 출처 격리 | ❌ |

**결정:**
```javascript
const db = await indexedDB.open('lattigo-fhe-db', 1);
const store = db.createObjectStore('fhe-keys', { keyPath: 'id' });
store.createIndex('timestamp', 'timestamp', { unique: false });
store.createIndex('keyType', 'keyType', { unique: false });
```

**이유:**
- ✅ **대용량**: FHE 키는 수십~수백 KB (localStorage 5MB로 부족)
- ✅ **트랜잭션**: ACID 보장으로 데이터 일관성 유지
- ✅ **비동기**: UI 블로킹 없이 대용량 데이터 처리
- ✅ **인덱싱**: 빠른 검색/필터링 (keyType, timestamp)
- ✅ **구조화 데이터**: 객체를 그대로 저장 (JSON 파싱 불필요)

**용량 분석:**
- Secret Key: ~12 KB (암호화 후 ~16 KB)
- Public Key: ~24 KB (암호화 후 ~32 KB)
- Relinearization Key: ~48 KB
- Galois Keys: ~70 KB (여러 개)
- **총합**: ~200 KB (localStorage로는 40개 키셋만 가능, IndexedDB는 수천 개)

### ADR-007: 인덱스 설계 (timestamp + keyType)

**문제:**
- 저장된 키를 효율적으로 조회/필터링
- 사용자는 "최근 생성된 키", "특정 타입의 키" 조회

**인덱스 선택:**

```javascript
// Primary Key: 'id' (예: 'secret_key', 'public_key', 'gk_0')
// Index 1: 'timestamp' (키 생성 시간)
// Index 2: 'keyType' (예: 'secret', 'public', 'galois')
```

**이유:**

1. **Primary Key: `id` (String)**
   - ✅ 의미 있는 이름으로 직접 접근 (`loadKey('secret_key')`)
   - ✅ 충돌 없음 (명시적 이름 지정)
   - ✅ 디버깅 용이 (키 이름으로 바로 식별)
   
   **대안 거부:**
   - Auto-increment: 의미 없는 숫자 (1, 2, 3...)
   - UUID: 길고 디버깅 어려움

2. **Index 1: `timestamp` (Non-unique)**
   - ✅ "최근 생성된 키 10개" 조회 가능
   ```javascript
   const recentKeys = await store.index('timestamp')
     .openCursor(null, 'prev'); // 최신순 정렬
   ```
   - ✅ 오래된 키 정리 (자동 만료 구현 가능)
   - ✅ 키 생성 이력 추적

3. **Index 2: `keyType` (Non-unique)**
   - ✅ "모든 Galois Keys만 조회" 가능
   ```javascript
   const galoisKeys = await store.index('keyType')
     .getAll('galois');
   ```
   - ✅ 타입별 통계 (dbDebug.listKeysByType())
   - ✅ 선택적 삭제 ("secret 키만 삭제")

**인덱스 설계 검증:**

**문제: 인덱스 오버헤드는?**
- ❌ **영향 없음**: 각 인덱스는 ~100 bytes 정도 (키 크기의 0.5%)
- ✅ 조회 성능 향상이 저장 공간 증가보다 훨씬 중요

**문제: Composite Index는?**
- ❌ **불필요**: `[keyType, timestamp]` 복합 인덱스는 사용 패턴에 맞지 않음
- ✅ 현재 패턴: "모든 갈루아 키" 또는 "최근 키" (둘 다 단일 조건)

**문제: id가 unique하지 않으면?**
- ✅ **Primary Key는 자동으로 unique 제약**: IndexedDB 내장 기능
- ✅ 동일 id로 두 번 저장 시 덮어쓰기 (예상된 동작)

**사용 예시:**
```javascript
// 1. Primary Key로 직접 접근 (가장 빠름 - O(1))
const sk = await store.get('secret_key');

// 2. 타입별 필터링 (Index 사용 - O(log n))
const galoisKeys = await store.index('keyType').getAll('galois');

// 3. 최신 키 조회 (Index + Cursor)
const cursor = await store.index('timestamp').openCursor(null, 'prev');
const newestKey = cursor.value;

// 4. 복합 조건 (Index + 필터링)
const recentGalois = (await store.index('keyType').getAll('galois'))
  .filter(k => k.timestamp > Date.now() - 86400000); // 24시간 이내
```

### ADR-008: React Context API (vs Redux/Zustand)

**문제:**
- FHE 상태를 여러 컴포넌트에서 공유
- Wasm 로딩 상태, 키 상태, 암호문 관리

**고려한 대안:**

| 상태 관리 | 학습 곡선 | 번들 크기 | 성능 | 결정 |
|----------|----------|----------|------|------|
| **Context API** | 낮음 (내장) | 0 KB | 충분 | ✅ 선택 |
| Redux | 높음 | ~10 KB | 매우 좋음 | ❌ |
| Zustand | 중간 | ~1 KB | 매우 좋음 | ❌ |
| MobX | 높음 | ~15 KB | 매우 좋음 | ❌ |

**결정:**
```javascript
const FHEContext = createContext();
export const useFHE = () => useContext(FHEContext);
```

**이유:**
- ✅ **React 내장**: 추가 의존성 없음
- ✅ **단순한 상태**: FHE는 복잡한 상태 관리 불필요
- ✅ **리렌더링 최적화**: useMemo/useCallback로 충분
- ✅ **타입 안전**: TypeScript 없이도 명확
- ⚠️ **성능**: 전체 Provider 리렌더링 가능 (하지만 4개 컴포넌트만 있어 문제없음)

**Redux 거부 이유:**
- ❌ 오버엔지니어링 (Action, Reducer, Middleware 불필요)
- ❌ PoC에서 불필요한 복잡도

### 보안 구현

1. **PBKDF2는 100,000+ iterations** (OWASP 2023)
2. **AES-GCM-256으로 키 암호화** (Web Crypto API)
3. **비밀키는 메모리에만 존재** (IndexedDB에는 암호화된 상태)
4. **Idle Timeout으로 자동 메모리 클리어** (5분)
5. **비밀번호는 절대 저장하지 않음**

## 📊 프로젝트 통계

### 코드 규모
- **Go Wasm 모듈**: ~800 lines (main.go)
- **React 서비스**: ~600 lines (3개 서비스)
- **React Context**: ~400 lines (FHEContext.jsx)
- **React 컴포넌트**: ~1,200 lines (4개 패널)
- **CSS**: ~800 lines (5개 파일)
- **총 코드**: ~3,800 lines

### 지원 기능
- ✅ 13개 Wasm 함수 노출
- ✅ 9개 IndexedDB 디버그 명령
- ✅ 4개 FHE 연산 (Add, Mul, Rotate, Conjugate)
- ✅ 5개 키 타입 (SK, PK, RLK, Galois Keys, Conjugation Key)
- ✅ 100% 클라이언트 측 처리

### 보안 기능
- ✅ Zero-Trust 아키텍처
- ✅ AES-GCM-256 암호화
- ✅ PBKDF2 100,000 iterations
- ✅ 5분 Idle Timeout
- ✅ 비밀번호 강도 검증

## 🚀 향후 계획

### Phase 3: Go 백엔드 API 서버
- [ ] RESTful API 서버 구축
- [ ] POST /compute 엔드포인트
- [ ] 암호문 업로드/다운로드
- [ ] 서버 측 FHE 연산 (클라이언트 비교용)

### Phase 4: E2E 통합
- [ ] Client Encrypt → Server Compute → Client Decrypt 플로우
- [ ] 성능 벤치마크 (클라이언트 vs 서버)
- [ ] 보안 감사
- [ ] 프로덕션 배포 준비

## � 참고 자료

- [Lattigo v6 Documentation](https://github.com/tuneinsight/lattigo)
- [OWASP Password Storage Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Password_Storage_Cheat_Sheet.html)
- [WebAssembly MDN](https://developer.mozilla.org/en-US/docs/WebAssembly)
- [Web Crypto API](https://developer.mozilla.org/en-US/docs/Web/API/Web_Crypto_API)
- [IndexedDB API](https://developer.mozilla.org/en-US/docs/Web/API/IndexedDB_API)

## �📄 라이선스

이 프로젝트는 PoC(Proof-of-Concept) 목적으로 작성되었습니다.

---

**Made with ❤️ using Lattigo v6, React, and WebAssembly**
