# CKKS 계수 크기 문제 분석 및 해결방안

**Status**: ✅ **RESOLVED** (November 28, 2025)

## 최종 해결된 모델

### 현재 프로덕션 모델 (5 features)

**계수 (모두 안전한 범위):**
```
age:              -0.2502  (10^-1 수준) ✅
loan_to_income:    0.0137  (10^-2 수준) ✅
debt_to_income:    0.0124  (10^-2 수준) ✅
credit_amount:    -0.0427  (10^-2 수준) ✅
income:            0.0063  (10^-3 수준) ✅
Intercept:        -1.4137
```

**CKKS Safety**: All coefficients in range [10^-3, 10^-1] - optimal for CKKS

---

## 과거 문제 요약 (참고용)

### 1. 계수가 너무 작아서 CKKS에서 문제 발생 (OLD)

**스케일링 없는 모델 (DEPRECATED):**
```
AMT_INCOME_TOTAL:  -0.0000002693  (10^-7 수준) ❌
AMT_CREDIT:        -0.0000003640  (10^-7 수준) ❌
DAYS_BIRTH:         0.0000628681  (10^-5 수준) ⚠️
AMT_ANNUITY:        0.0000050115  (10^-6 수준) ⚠️
DAYS_EMPLOYED:     -0.0000001404  (10^-7 수준) ❌
Intercept:         -1.3283623056
```

**CKKS 정밀도:**
- Scale = 2^40 ≈ 1.1 × 10^12
- 유효 정밀도: 약 10^-10 ~ 10^-12
- **10^-7 수준의 계수는 곱셈 후 노이즈에 묻힐 위험**

**문제점:**
1. **정밀도 손실**: 작은 계수가 노이즈에 가려짐
2. **수치 불안정**: 큰 피처값 × 작은 계수 = 부동소수점 오차
3. **노이즈 증폭**: Rescaling 과정에서 오차 누적

---

## 해결 방법 1: 유사 스케일링 적용 (권장) ✅

### Training 시 스케일링
```python
# 금액 → 만원 단위
X_unit["AMT_INCOME_TOTAL"] /= 10000  # 150,000 → 15.0
X_unit["AMT_CREDIT"] /= 10000        # 600,000 → 60.0
X_unit["AMT_ANNUITY"] /= 10000       # 20,000 → 2.0

# Days → 천일 단위 (약 3년)
X_unit["DAYS_BIRTH"] /= 1000         # -15000 → -15.0
X_unit["DAYS_EMPLOYED"] /= 1000      # -2000 → -2.0
```

### 결과 계수
```
AMT_INCOME_TOTAL:  -0.0026934041  ✅ (10^-3 수준)
AMT_CREDIT:        -0.0036403247  ✅ (10^-3 수준)
DAYS_BIRTH:         0.0628677989  ✅ (10^-2 수준)
AMT_ANNUITY:        0.0501126373  ✅ (10^-2 수준)
DAYS_EMPLOYED:     -0.0001404317  ⚠️ (10^-4 수준, 여전히 작음)
Intercept:         -1.3283632277
```

**장점:**
- 계수가 10^-2 ~ 10^-3 수준으로 안정적
- CKKS에서 충분한 정밀도 확보
- 노이즈 대비 신호 비율(SNR) 개선

**단점:**
- DAYS_EMPLOYED는 여전히 작음 (10^-4)
- 추가 스케일링 필요할 수 있음

---

## 해결 방법 2: StandardScaler 적용 (Alternative)

### Training 시 정규화
```python
from sklearn.preprocessing import StandardScaler

scaler = StandardScaler()
X_scaled = scaler.fit_transform(X_raw)

# 각 피처가 평균 0, 표준편차 1로 변환
# 계수 크기가 비슷해짐
```

### 예상 결과
```
모든 계수:  0.1 ~ 0.5 범위  ✅
Intercept:  약 -0.7
```

**장점:**
- 모든 계수가 유사한 크기로 균일화
- CKKS에서 가장 안정적
- 머신러닝 Best Practice

**단점:**
- 평균/표준편차 저장 및 관리 필요
- Inference 시 동일한 scaler 적용 필수

---

## 해결 방법 3: 계수 인위적 스케일링 (Not Recommended)

### 모든 계수 × 1000 배
```python
model.coef_ *= 1000
model.intercept_ *= 1000

# Inference 후 결과 / 1000
```

**장점:**
- 간단한 구현

**단점:**
- 수학적으로 부정확
- Sigmoid 입력값 범위 왜곡
- **권장하지 않음**

---

## 전처리 불일치 문제

### 현재 상황

**Training (Logistic.ipynb):**
```python
Features:
1. AMT_INCOME_TOTAL  (연소득, 원화)
2. AMT_CREDIT        (대출금액, 원화)
3. DAYS_BIRTH        (생년월일 Days, 음수)
4. AMT_ANNUITY       (연금액, 원화)
5. DAYS_EMPLOYED     (고용일 Days, 음수)

Preprocessing:
- 금액 / 10000
- Days / 1000
```

**Inference (features.ts):**
```typescript
Features:
1. age               (나이, 년)
2. income            (연소득, 달러)
3. creditHistory     (신용기록, 개월)
4. loanAmount        (대출금액, 달러)
5. employmentYears   (고용기간, 년)
6. debtToIncome      (부채비율, 0~1)

Preprocessing:
- age / 100
- income / 100000
- creditHistory / 120
- loanAmount / 50000
- employmentYears / 40
- debtToIncome (none)
```

**문제점:**
1. ❌ **피처 이름 불일치**
2. ❌ **피처 개수 불일치** (5개 vs 6개)
3. ❌ **전처리 방법 불일치**
4. ❌ **단위 불일치** (원화 vs 달러, Days vs 년)

---

## 해결 방안 (권장)

### 옵션 A: Training 모델 재훈련 (Best Practice) ✅

#### 1. 피처 매핑 정의
```python
# logistic/train_production_model.py

FEATURE_MAPPING = {
    'age': 'DAYS_BIRTH',           # -15000 days → 41 years
    'income': 'AMT_INCOME_TOTAL',  # 150000원 → $50000
    'creditHistory': 'DAYS_CREDIT', # months
    'loanAmount': 'AMT_CREDIT',    # 원 → $
    'employmentYears': 'DAYS_EMPLOYED', # days → years
    'debtToIncome': 'CALCULATED'   # 새로 계산
}
```

#### 2. 전처리 통일
```python
# Training preprocessing
def preprocess_for_production(df):
    X = pd.DataFrame()
    
    # Age: DAYS_BIRTH를 년으로 변환 후 /100
    X['age'] = (-df['DAYS_BIRTH'] / 365.25) / 100
    
    # Income: 원화를 달러로 변환 후 /100000
    X['income'] = (df['AMT_INCOME_TOTAL'] / 1200) / 100000
    
    # Credit History: 신용기록 개월 / 120
    X['creditHistory'] = df['MONTHS_BALANCE'].abs() / 120
    
    # Loan Amount: 원화를 달러로 변환 후 /50000
    X['loanAmount'] = (df['AMT_CREDIT'] / 1200) / 50000
    
    # Employment Years: DAYS_EMPLOYED를 년으로 변환 후 /40
    X['employmentYears'] = (-df['DAYS_EMPLOYED'] / 365.25) / 40
    
    # Debt to Income: 계산
    X['debtToIncome'] = df['AMT_ANNUITY'] / df['AMT_INCOME_TOTAL']
    
    return X

# 재훈련
X_prod = preprocess_for_production(df)
y = df['TARGET']

model = LogisticRegression(max_iter=1000)
model.fit(X_prod, y)

# 계수 출력
print("Production Model Coefficients:")
for name, coef in zip(['age', 'income', 'creditHistory', 
                       'loanAmount', 'employmentYears', 'debtToIncome'], 
                      model.coef_[0]):
    print(f"{name:20s}  {coef:.6f}")
```

#### 3. Backend 업데이트
```go
// backend/main.go

var model = LogisticRegressionModel{
    // 재훈련된 계수로 교체
    Weights: []float64{
        -0.123456,  // age
        -0.234567,  // income
        0.345678,   // creditHistory
        -0.456789,  // loanAmount
        0.567890,   // employmentYears
        -0.678901,  // debtToIncome
    },
    Bias: -1.234567,
}
```

---

### 옵션 B: Frontend 전처리 수정 (Quick Fix) ⚠️

#### features.ts 수정
```typescript
export const CREDIT_FEATURES: FeatureConfig[] = [
  {
    name: 'AMT_INCOME_TOTAL',  // 피처명 변경
    label: 'Annual Income (KRW)',
    type: 'number',
    min: 0,
    max: 1000000000,  // 10억원
    step: 1000000,
    defaultValue: 150000000,  // 1.5억원
    preprocessing: {
      method: 'normalize',
      params: { scale: 10000 },  // 만원 단위
    },
  },
  // ... 나머지 피처도 동일하게 수정
];
```

**문제점:**
- 사용자가 "나이"가 아닌 "DAYS_BIRTH"를 입력해야 함
- UI/UX 매우 나쁨
- **권장하지 않음**

---

## 최종 권장 사항

### 📋 Action Items

1. **즉시 (Phase 2.5):**
   - [ ] `logistic/train_production_model.py` 생성
   - [ ] Frontend와 동일한 6개 피처로 재훈련
   - [ ] 전처리 통일 (features.ts와 일치)
   - [ ] 계수 크기 확인 (10^-2 ~ 10^-1 목표)
   - [ ] Backend에 새 계수 적용

2. **테스트:**
   - [ ] Dummy 데이터로 Training 결과 재현
   - [ ] CKKS Encryption → Inference → Decryption
   - [ ] 정확도 vs 평문 비교 (오차 < 1%)

3. **문서화:**
   - [ ] 전처리 스펙 문서화
   - [ ] 피처 매핑 테이블 작성
   - [ ] CKKS 정밀도 검증 리포트

---

## CKKS 정밀도 검증 방법

### 테스트 코드
```python
# test_ckks_precision.py

import numpy as np

# CKKS 시뮬레이션 (간단 버전)
def ckks_simulate(value, scale=2**40, noise_std=2**-30):
    # 정수 인코딩
    scaled = int(value * scale)
    
    # 노이즈 추가
    noise = np.random.normal(0, noise_std * scale)
    scaled_noisy = scaled + int(noise)
    
    # 디코딩
    decoded = scaled_noisy / scale
    
    return decoded, abs(decoded - value)

# 계수 테스트
coefficients = {
    'large': 0.05,
    'medium': 0.005,
    'small': 0.0005,
    'tiny': 0.00005,
}

for name, coef in coefficients.items():
    decoded, error = ckks_simulate(coef)
    rel_error = error / abs(coef) * 100
    print(f"{name:10s}  Coef: {coef:.6f}  Decoded: {decoded:.6f}  "
          f"Error: {error:.2e}  ({rel_error:.2f}%)")
```

### 예상 결과
```
large      Coef: 0.050000  Decoded: 0.050001  Error: 9.1e-07  (0.00%)  ✅
medium     Coef: 0.005000  Decoded: 0.005001  Error: 9.1e-07  (0.02%)  ✅
small      Coef: 0.000500  Decoded: 0.000501  Error: 9.1e-07  (0.18%)  ⚠️
tiny       Coef: 0.000050  Decoded: 0.000051  Error: 9.1e-07  (1.82%)  ❌
```

**결론:**
- **10^-2 이상**: 안전 ✅
- **10^-3**: 허용 가능 ⚠️
- **10^-4 이하**: 위험 ❌

---

## 참고: CKKS Parameters

```go
// Current Settings
LogN:            14,           // Ring degree: 2^14 = 16384
LogQ:            []int{60, 40, 40, 60},  // 4 levels
LogDefaultScale: 40,           // Scale: 2^40 ≈ 1.1e12

// Precision
Effective Precision: ~10^-10 to 10^-12
Relative Error:      ~2^-40 ≈ 9e-13
```

**계수 크기 가이드라인:**
- **Optimal**: 0.01 ~ 1.0 (10^-2 ~ 10^0)
- **Acceptable**: 0.001 ~ 10.0 (10^-3 ~ 10^1)
- **Risky**: < 0.001 or > 10.0
