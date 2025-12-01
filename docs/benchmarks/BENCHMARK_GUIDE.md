# 📋 벤치마크 및 파일 구조 가이드

## 📁 프로젝트 구조

```
ckks_credit/
├── backend/                      # 프로덕션 백엔드 (LogN=13 최적화)
│   ├── main.go                  # 메인 서버 (LogN=13, 6 levels)
│   ├── server                   # 빌드된 실행 파일
│   ├── sigmoid/                 # Sigmoid 근사 구현
│   │   └── sigmoid.go
│   └── cmd/benchmark/
│       └── sigmoid_benchmark.go # Sigmoid 근사 비교 벤치마크
│
├── test/                        # 프로덕션 E2E 테스트
│   ├── e2e.go                  # E2E 테스트 (LogN=13)
│   └── e2e_packed.go           # Packed ciphertext 테스트
│
├── baseline/                    # 벤치마크용 변형 버전들
│   ├── optimized/              # 현재 최적화 버전 백업
│   │   ├── main.go             # LogN=13 백엔드
│   │   └── e2e.go              # LogN=13 E2E 테스트
│   ├── logn14/                 # Baseline 비교용
│   │   ├── main_baseline.go    # LogN=14, 4 levels
│   │   ├── e2e_baseline.go     # LogN=14 E2E 테스트
│   │   └── server_baseline     # 빌드된 실행 파일
│   ├── packed/                 # (미구현) Galois key 사용
│   └── simd/                   # (미구현) WASM SIMD 최적화
│
├── benchmark_results/           # 벤치마크 결과 저장
│   ├── 1_baseline_logn14.txt   # Baseline 결과
│   ├── 2_optimized_logn13.txt  # 최적화 버전 결과
│   └── 3_sigmoid_methods.txt   # Sigmoid 근사 비교
│
├── image/                       # 시각화 결과
│   └── presentation/           # 발표용 고품질 그래프
│       ├── 1_e2e_comparison.png
│       ├── 2_sigmoid_analysis.png
│       └── 3_optimization_impact.png
│
├── run_benchmarks.sh            # 전체 벤치마크 자동화 스크립트
└── visualize_benchmark.py       # 발표용 시각화 생성 스크립트
```

## 🚀 벤치마크 실행 방법

### 1. 전체 벤치마크 실행 (권장)

```bash
# 1단계: 전체 벤치마크 실행 (약 5분 소요)
./run_benchmarks.sh

# 2단계: 결과 시각화
python3 visualize_benchmark.py
```

이 명령은 다음을 순차적으로 실행합니다:
1. **Baseline (LogN=14)**: 원래 파라미터로 E2E 테스트
2. **Optimized (LogN=13)**: 최적화된 파라미터로 E2E 테스트
3. **Sigmoid 근사**: 7가지 방법 정확도/속도 비교

### 2. 개별 벤치마크 실행

#### Baseline (LogN=14) 테스트
```bash
# 백엔드 시작
cd baseline/logn14
go build -o server_baseline main_baseline.go
./server_baseline &
cd ../..

# E2E 테스트
cd baseline/logn14
go run e2e_baseline.go
cd ../..

# 백엔드 종료
pkill server_baseline
```

#### Optimized (LogN=13) 테스트
```bash
# 백엔드 시작
cd backend
go build -o server main.go
./server &
cd ..

# E2E 테스트
cd test
go run e2e.go
cd ..

# 백엔드 종료
pkill server
```

#### Sigmoid 근사 비교
```bash
cd backend/cmd/benchmark
go run sigmoid_benchmark.go
cd ../../..
```

## 📊 생성되는 시각화

### 1. E2E Performance Comparison (`1_e2e_comparison.png`)
- **전체 E2E 시간**: Baseline vs Optimized
- **단계별 분해**: 암호화/백엔드/복호화 시간
- **네트워크 트래픽**: 요청당 전송 데이터
- **최적화 요약 표**: 모든 메트릭 비교

**핵심 발견**:
- ⚡ **2.1x faster**: 평균 E2E 시간 감소
- 📦 **50% smaller**: 네트워크 트래픽 절반
- ✅ **100% success**: 모든 테스트 통과

### 2. Sigmoid Approximation Analysis (`2_sigmoid_analysis.png`)
- **정확도 비교**: 각 근사 방법의 평균 오차
- **계산 시간**: 방법별 FHE 연산 시간
- **정확도 vs 속도**: Trade-off 분석
- **회로 깊이**: 필요한 CKKS 레벨

**테스트된 방법**:
- Chebyshev-3, 5, 7 (다항식 근사)
- Minimax-3, 5, 7 (최적 근사)
- Composite-3 (구간별 근사)

### 3. Optimization Impact (`3_optimization_impact.png`)
- **다차원 비교**: 레이더 차트로 전체 성능
- **비용-이익 분석**: 최적화 기법별 효과
- **추천 매트릭스**: 사용 사례별 권장 설정

## 🔧 파일별 상세 설명

### 프로덕션 파일 (현재 사용)

#### `backend/main.go`
- **파라미터**: LogN=13, 6 levels [60,40,40,40,40,60]
- **최적화**: 2.1x 빠름, 50% 작은 크기
- **상태**: 프로덕션 준비 완료
- **빌드**: `go build -o server main.go`

#### `test/e2e.go`
- **테스트**: 5가지 신용 시나리오
- **검증**: 복호화 값 범위 [0, 1]
- **출력**: 상세한 타이밍 및 네트워크 메트릭
- **실행**: `go run e2e.go`

### 벤치마크 파일

#### `baseline/logn14/main_baseline.go`
- **파라미터**: LogN=14, 4 levels [60,40,40,60]
- **목적**: 원래 설정 대비 성능 비교
- **차이점**: 더 큰 파라미터, 적은 레벨

#### `baseline/logn14/e2e_baseline.go`
- **독립 실행**: 자체 파라미터 내장
- **backend 불필요**: 직접 CKKS 연산 수행
- **출력 형식**: 표준 벤치마크 포맷

#### `backend/cmd/benchmark/sigmoid_benchmark.go`
- **테스트 범위**: [-8, 8] logit 값
- **신용 점수 범위**: [-3, 0] 특별 분석
- **출력**: 7가지 방법 비교 표

## 📈 벤치마크 결과 해석

### E2E 메트릭
- **E2E Time**: 클라이언트 암호화 → 서버 추론 → 클라이언트 복호화 전체 시간
- **Encryption Time**: 5개 feature를 ciphertext로 변환하는 시간
- **Backend Time**: FHE 연산 (가중치 곱셈 + Sigmoid) 시간
- **Decryption Time**: 결과 ciphertext를 점수로 변환하는 시간
- **Network Traffic**: 요청 (5 ct + RLK) + 응답 (1 ct) 총 크기

### Sigmoid 메트릭
- **Mean Error**: 전체 테스트 포인트의 평균 절대 오차
- **Max Error**: 가장 큰 오차 (worst case)
- **Std Dev**: 오차의 표준 편차 (일관성 지표)
- **Time**: 17개 테스트 포인트 총 실행 시간
- **Depth**: 필요한 CKKS 레벨 수

## 🎯 권장 사항

### 프로덕션 배포
- **설정**: Optimized (LogN=13, 6 levels)
- **이유**: 2.1x 빠름, 50% 네트워크 절감
- **확인**: E2E 테스트 5/5 통과

### 개발/테스트
- **설정**: Optimized (LogN=13)
- **이유**: 빠른 반복 주기
- **참고**: Baseline과 동일한 보안 레벨

### 모바일/IoT
- **설정**: Optimized (LogN=13)
- **이유**: 낮은 대역폭 요구사항
- **대안**: Packed ciphertext (추가 50% 절감)

### 높은 보안 요구
- **설정**: Baseline (LogN=14) 또는 더 큼
- **이유**: 더 큰 파라미터 공간
- **trade-off**: 2x 느림, 2x 큰 크기

## 🐛 문제 해결

### 벤치마크 실행 실패
```bash
# 이전 서버 프로세스 종료
pkill -9 server server_baseline

# 포트 확인
lsof -i :8080

# 재실행
./run_benchmarks.sh
```

### 시각화 에러
```bash
# matplotlib 설치
pip3 install matplotlib numpy

# 결과 파일 확인
ls -lh benchmark_results/

# 재시도
python3 visualize_benchmark.py
```

### Go 빌드 에러
```bash
# 의존성 업데이트
cd backend
go mod tidy

# 또는
go get github.com/tuneinsight/lattigo/v6/schemes/ckks
```

## 📝 결과 파일 형식

### E2E 결과 (`*_logn*.txt`)
```
🧪 CKKS Credit Scoring E2E Test
================================

📊 CKKS Parameters: LogN=13, MaxLevel=5, MaxSlots=4096

🔑 Generating keys...
✅ Keys generated in 54.20ms

Test 1/5: Good Credit - High Income
------------------------------------------------------------
🔒 Encrypting features...
✅ Encryption completed in 64.89ms (Total: 7681.83 KB)
📡 Sending to backend...
✅ Backend inference completed in 321.36ms
🔓 Decrypting result...
✅ Decryption completed in 8.66ms
📊 Results:
  Total E2E Time: 394.91ms
  Total Network: 25348.09 KB
✅ PASS

============================================================
🎯 Test Summary: 5/5 passed (100.0%)
⚡ Average E2E Time: 346.97ms
📦 Average Network: 24.75 MB
```

### Sigmoid 결과 (`3_sigmoid_methods.txt`)
```
📈 Sigmoid Approximation Methods Comparison
============================================

Results:
--------
Method               | Mean Error   | Max Error    | Std Dev      | Time (ms) | Depth   
-------------------------------------------------------------------------------------------------
Chebyshev-3          | 1.234e-02    | 3.456e-02    | 5.678e-03    |      45.2 |        3
Minimax-5            | 2.345e-03    | 6.789e-03    | 1.234e-03    |      78.9 |        5
...

🏆 Recommendations:
Best Accuracy:  Minimax-7 (Mean Error: 1.234e-04)
Fastest:        Chebyshev-3 (Time: 45.2 ms)
Best Balanced:  Minimax-5 (Error: 2.345e-03, Time: 78.9 ms)
```

## 🔄 업데이트 이력

### 2025-11-30
- ✅ 파일 구조 체계화 (baseline/ 디렉토리 분리)
- ✅ 독립 실행 가능한 벤치마크 생성
- ✅ 전체 자동화 스크립트 (`run_benchmarks.sh`)
- ✅ 발표용 시각화 (`visualize_benchmark.py`)
- ✅ 상세 문서 작성

### 이전
- Sigmoid 근사 비교 구현
- LogN=13 최적화 (2.1x speedup)
- E2E 테스트 프레임워크
- 프로덕션 백엔드 배포
