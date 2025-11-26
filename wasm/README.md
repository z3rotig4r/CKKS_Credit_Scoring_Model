# Lattigo Wasm Module

이 디렉토리는 Lattigo FHE 라이브러리를 WebAssembly로 컴파일한 모듈을 포함합니다.

## 📋 파일 구조

```
wasm/
├── main.go          # Wasm 래퍼 코드 (syscall/js 사용)
├── go.mod           # Go 모듈 정의
├── go.sum           # 의존성 체크섬
├── build.sh         # 빌드 스크립트
└── README.md        # 이 문서
```

## 🔧 빌드 방법

### 방법 1: 빌드 스크립트 사용 (권장)

```bash
cd wasm
./build.sh
```

### 방법 2: 수동 빌드

```bash
cd wasm

# wasm_exec.js 복사
cp "$(go env GOROOT)/lib/wasm/wasm_exec.js" ../frontend/public/

# Wasm 컴파일
GOOS=js GOARCH=wasm go build -o ../frontend/public/main.wasm main.go
```

## 📦 출력 파일

빌드가 완료되면 다음 파일들이 생성됩니다:

- `../frontend/public/main.wasm` - Lattigo Wasm 모듈
- `../frontend/public/wasm_exec.js` - Go Wasm 런타임

## 🎯 노출된 JavaScript 함수

Wasm 모듈이 로드되면 다음 함수들이 전역 객체에 등록됩니다:

### 1. `fheKeygen()`
FHE 키 쌍 생성 (SK + PK)

**반환값:** `Promise<{secretKey: Uint8Array, publicKey: Uint8Array}>`

```javascript
const keys = await fheKeygen();
console.log("Secret Key:", keys.secretKey);
console.log("Public Key:", keys.publicKey);
```

### 2. `fheGenAllKeys()`
모든 평가 키 한번에 생성 (SK + PK + RLK + Galois Keys)

**반환값:** `Promise<{secretKey: Uint8Array, publicKey: Uint8Array, relinearizationKey: Uint8Array, galoisKeys: Uint8Array[]}>`

```javascript
const allKeys = await fheGenAllKeys();
console.log("All keys generated:", allKeys);
```

### 3. `fheGenRelinearizationKey(secretKey)`
Relinearization Key 생성 (곱셈 후 차수 감소용)

**매개변수:**
- `secretKey: Uint8Array` - 비밀키

**반환값:** `Promise<Uint8Array>` - Relinearization Key

```javascript
const rlk = await fheGenRelinearizationKey(keys.secretKey);
```

### 4. `fheGenGaloisKeys(secretKey, [galoisElements])`
Galois Keys 생성 (회전 연산용)

**매개변수:**
- `secretKey: Uint8Array` - 비밀키
- `galoisElements?: number[]` - (선택) 특정 갈루아 요소들. 생략시 기본 회전 키들 생성

**반환값:** `Promise<Uint8Array[]>` - Galois Keys 배열

```javascript
const gks = await fheGenGaloisKeys(keys.secretKey);
// 또는 특정 요소만:
const customGks = await fheGenGaloisKeys(keys.secretKey, [5, 25, 49]);
```

### 5. `fheGenRotationKeys(secretKey, rotations)`
특정 회전을 위한 Rotation Keys 생성

**매개변수:**
- `secretKey: Uint8Array` - 비밀키
- `rotations: number[]` - 회전 인덱스 배열 (예: [1, 2, 4, 8])

**반환값:** `Promise<Uint8Array[]>` - Rotation Keys 배열

```javascript
const rotKeys = await fheGenRotationKeys(keys.secretKey, [1, 2, 4, 8]);
```

### 6. `fheGenConjugationKey(secretKey)`
Conjugation Key 생성 (복소수 켤레 연산용)

**매개변수:**
- `secretKey: Uint8Array` - 비밀키

**반환값:** `Promise<Uint8Array>` - Conjugation Key

```javascript
const conjKey = await fheGenConjugationKey(keys.secretKey);
```

### 7. `fheEncrypt(publicKey, plaintext)`
평문 암호화

**매개변수:**
- `publicKey: Uint8Array` - 공개키
- `plaintext: number` - 암호화할 숫자

**반환값:** `Promise<Uint8Array>` - 암호문

```javascript
const ciphertext = await fheEncrypt(keys.publicKey, 7500);
```

### 8. `fheDecrypt(secretKey, ciphertext)`
암호문 복호화

**매개변수:**
- `secretKey: Uint8Array` - 비밀키
- `ciphertext: Uint8Array` - 암호문

**반환값:** `Promise<number>` - 복호화된 평문

```javascript
const plaintext = await fheDecrypt(keys.secretKey, ciphertext);
console.log("Decrypted:", plaintext);
```

### 9. `fheGetParamsInfo()`
CKKS 파라미터 정보 조회

**반환값:** `string` - JSON 형식의 파라미터 정보

```javascript
const paramsJson = fheGetParamsInfo();
const params = JSON.parse(paramsJson);
console.log("Parameters:", params);
```

## 🔒 보안 고려사항

1. **비밀키 관리:** `secretKey`는 절대 서버로 전송하지 마세요
2. **평가 키 관리:** 
   - `relinearizationKey`, `galoisKeys`는 서버로 전송 가능 (공개 키와 동일한 보안 수준)
   - 이 키들은 암호문에 대한 연산에만 사용되며, 복호화에는 사용되지 않음
3. **메모리 관리:** 복호화 후 `secretKey`를 즉시 메모리에서 제거하세요
4. **저장소:** `secretKey`는 암호화된 형태로만 IndexedDB에 저장하세요

## 🔑 키 타입 설명

### Secret Key (SK)
- **용도:** 복호화 전용
- **보안:** 절대 공개 불가
- **크기:** ~16 KB

### Public Key (PK)
- **용도:** 암호화
- **보안:** 공개 가능
- **크기:** ~32 KB

### Relinearization Key (RLK)
- **용도:** 곱셈 연산 후 암호문 차수 감소
- **보안:** 공개 가능 (서버 전송 OK)
- **크기:** ~900 KB
- **필요한 경우:** 암호문 간 곱셈 수행 시

### Galois Keys (Rotation Keys)
- **용도:** 슬롯 회전 (Rotate) 연산
- **보안:** 공개 가능 (서버 전송 OK)
- **크기:** 개당 ~900 KB (여러 개)
- **필요한 경우:** 벡터 연산, 합산, Bootstrap 등

### Conjugation Key
- **용도:** 복소수 켤레 (Complex Conjugation) 연산
- **보안:** 공개 가능 (서버 전송 OK)
- **크기:** ~900 KB
- **필요한 경우:** 실수 인코딩, 특정 변환 연산

## 📚 CKKS 파라미터

현재 설정:
- **LogN:** 14 (링 차원: 2^14 = 16384)
- **LogQ:** [60, 40, 40, 60] (모듈러스 체인)
- **LogP:** [61] (키 스위칭용)
- **LogDefaultScale:** 40 (스케일: 2^40)

## 🛠️ 의존성

- **Lattigo:** v6.0.0
- **Go:** 1.21+

## 🐛 디버깅

Wasm 모듈이 로드되면 콘솔에 다음과 같은 메시지가 출력됩니다:

```
Lattigo CKKS Wasm module initialized
Parameters: LogN=14, LogQ=[60 40 40 60], MaxLevel=3, MaxSlots=8192
FHE functions exposed to JavaScript:
  - fheKeygen()
  - fheEncrypt(publicKey, plaintext)
  - fheDecrypt(secretKey, ciphertext)
  - fheGetParamsInfo()
```
