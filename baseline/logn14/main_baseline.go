package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
	"github.com/tuneinsight/lattigo/v6/core/rlwe"
	"github.com/tuneinsight/lattigo/v6/schemes/ckks"
	"github.com/z3rotig4r/ckks_credit/backend/sigmoid"
)

const (
	// Maximum ciphertext size: 10MB (보안: 악의적 대용량 데이터 차단)
	MaxCiphertextSize = 10 * 1024 * 1024
)

var (
	params    ckks.Parameters
	evaluator *ckks.Evaluator
	encoder   *ckks.Encoder
	sk        *rlwe.SecretKey
	rlk       *rlwe.RelinearizationKey
)

type LogisticRegressionModel struct {
	Weights []float64
	Bias    float64
}

// Production model trained on 307,499 samples from application_train.csv
// User provides 4 inputs: age, loanAmount, income, monthlyPayment
// Frontend calculates 5 features and sends encrypted to backend
// Backend features: [age/10, loan_to_income, debt_to_income, credit_amount, income/100000]
// AUC-ROC: 0.5886, All coefficients CKKS-safe (0.01 ~ 1.0 range)
// NOTE: EXT_SOURCE_2 제거! 우리가 신용점수를 계산하는 시스템이므로!
var model = LogisticRegressionModel{
	Weights: []float64{
		-0.2501752295, // age (years / 10)
		0.0137090654,  // loan_to_income (loanAmount / income)
		0.0123900347,  // debt_to_income (monthlyPayment / (income/12) * 100)
		-0.0426762083, // credit_amount (loanAmount / 100000)
		0.0062886554,  // income (income / 100000)
	},
	Bias: -1.4136778933,
}

type InferenceRequest struct {
	EncryptedFeatures  []string `json:"encryptedFeatures"`
	RelinearizationKey string   `json:"relinearizationKey"` // Base64-encoded RLK from client
}

type PackedInferenceRequest struct {
	EncryptedVector    string `json:"encryptedVector"`    // Single ciphertext with all features
	RelinearizationKey string `json:"relinearizationKey"` // Base64-encoded RLK from client
	GaloisKey          string `json:"galoisKey"`          // Base64-encoded Galois key for rotations
}

type InferenceResponse struct {
	EncryptedScore string `json:"encryptedScore"`
	Timestamp      int64  `json:"timestamp"`
}

func init() {
	var err error
	params, err = ckks.NewParametersFromLiteral(ckks.ParametersLiteral{
		LogN:            14,                            // BASELINE configuration
		LogQ:            []int{60, 40, 40, 40, 40, 60}, // SAME 6 levels as optimized for fair comparison
		LogP:            []int{61},
		LogDefaultScale: 40,
	})
	if err != nil {
		panic(fmt.Sprintf("Failed to create CKKS parameters: %v", err))
	}

	// Only initialize encoder (evaluator created per-request with client's RLK)
	encoder = ckks.NewEncoder(params)

	log.Printf("CKKS Parameters: LogN=%d, MaxLevel=%d, MaxSlots=%d\n",
		params.LogN(), params.MaxLevel(), params.MaxSlots())
	log.Printf("✅ Backend ready to receive client's relinearization key\n")
}

func enableCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().Unix(),
	})
}

func inferenceHandler(w http.ResponseWriter, r *http.Request) {
	startTotal := time.Now()

	var req InferenceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("❌ ERROR: Failed to decode request body: %v", err)
		http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
		return
	}

	if len(req.EncryptedFeatures) != 5 {
		log.Printf("❌ ERROR: Invalid feature count: expected 5, got %d", len(req.EncryptedFeatures))
		http.Error(w, "Expected 5 encrypted features", http.StatusBadRequest)
		return
	}

	if req.RelinearizationKey == "" {
		log.Printf("❌ ERROR: Missing relinearization key")
		http.Error(w, "Relinearization key required", http.StatusBadRequest)
		return
	}

	log.Printf("📨 Received inference request with %d encrypted features + RLK", len(req.EncryptedFeatures))

	// Deserialize RLK
	rlkBytes, err := base64.StdEncoding.DecodeString(req.RelinearizationKey)
	if err != nil {
		log.Printf("❌ ERROR: Failed to decode RLK: %v", err)
		http.Error(w, fmt.Sprintf("Invalid RLK: %v", err), http.StatusBadRequest)
		return
	}

	rlk := new(rlwe.RelinearizationKey)
	if err := rlk.UnmarshalBinary(rlkBytes); err != nil {
		log.Printf("❌ ERROR: Failed to unmarshal RLK: %v", err)
		http.Error(w, fmt.Sprintf("Invalid RLK data: %v", err), http.StatusBadRequest)
		return
	}

	// Create evaluator with client's RLK
	evk := rlwe.NewMemEvaluationKeySet(rlk)
	evaluator := ckks.NewEvaluator(params, evk)
	log.Printf("✅ Created evaluator with client's relinearization key")

	startDeserialization := time.Now()
	encryptedFeatures := make([]*rlwe.Ciphertext, len(req.EncryptedFeatures))
	var totalBytes int64

	for i, b64Str := range req.EncryptedFeatures {
		// 크기 제한 검증
		if len(b64Str) > MaxCiphertextSize {
			log.Printf("❌ ERROR: Feature %d exceeds size limit: %d bytes", i, len(b64Str))
			http.Error(w, fmt.Sprintf("Feature %d exceeds maximum size", i), http.StatusBadRequest)
			return
		}

		ctBytes, err := base64.StdEncoding.DecodeString(b64Str)
		if err != nil {
			log.Printf("❌ ERROR: Failed to decode feature %d: %v", i, err)
			http.Error(w, fmt.Sprintf("Failed to decode feature %d: %v", i, err), http.StatusBadRequest)
			return
		}

		// 역직렬화 크기 검증
		if len(ctBytes) > MaxCiphertextSize {
			log.Printf("❌ ERROR: Decoded feature %d exceeds size limit: %d bytes", i, len(ctBytes))
			http.Error(w, fmt.Sprintf("Feature %d data too large", i), http.StatusBadRequest)
			return
		}

		// ✅ 올바른 역직렬화: 레벨 자동 복원
		ct := new(rlwe.Ciphertext)
		if err := ct.UnmarshalBinary(ctBytes); err != nil {
			log.Printf("❌ ERROR: Failed to unmarshal ciphertext %d: %v", i, err)
			http.Error(w, fmt.Sprintf("Invalid ciphertext %d: %v", i, err), http.StatusBadRequest)
			return
		}

		// 암호문 유효성 검증
		if ct.Level() < 0 || ct.Level() > params.MaxLevel() {
			log.Printf("❌ ERROR: Invalid ciphertext level %d (max: %d)", ct.Level(), params.MaxLevel())
			http.Error(w, fmt.Sprintf("Invalid ciphertext %d: bad level", i), http.StatusBadRequest)
			return
		}

		encryptedFeatures[i] = ct
		totalBytes += int64(len(ctBytes))
		log.Printf("✅ Feature %d: Level=%d, Size=%d bytes", i, ct.Level(), len(ctBytes))
	}

	deserializationTime := time.Since(startDeserialization)
	log.Printf("⏱️  Deserialization: %.2f ms (total %d bytes)",
		float64(deserializationTime.Microseconds())/1000.0, totalBytes)

	startInference := time.Now()
	result, err := performInference(evaluator, encryptedFeatures)
	if err != nil {
		log.Printf("❌ ERROR: Inference failed: %v", err)
		http.Error(w, fmt.Sprintf("Inference failed: %v", err), http.StatusInternalServerError)
		return
	}
	inferenceTime := time.Since(startInference)
	log.Printf("⏱️  Inference computation: %.2f ms", float64(inferenceTime.Microseconds())/1000.0)

	startSerialization := time.Now()
	resultBytes, err := result.MarshalBinary()
	if err != nil {
		log.Printf("❌ ERROR: Failed to marshal result: %v", err)
		http.Error(w, fmt.Sprintf("Failed to marshal result: %v", err), http.StatusInternalServerError)
		return
	}

	resultB64 := base64.StdEncoding.EncodeToString(resultBytes)
	serializationTime := time.Since(startSerialization)
	log.Printf("⏱️  Serialization: %.2f ms (%d bytes)",
		float64(serializationTime.Microseconds())/1000.0, len(resultBytes))

	response := InferenceResponse{
		EncryptedScore: resultB64,
		Timestamp:      time.Now().Unix(),
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("❌ ERROR: Failed to encode response: %v", err)
		return
	}

	totalTime := time.Since(startTotal)
	log.Printf("✅ Inference completed successfully - Total: %.2f ms (Deser: %.2f ms, Compute: %.2f ms, Ser: %.2f ms)",
		float64(totalTime.Microseconds())/1000.0,
		float64(deserializationTime.Microseconds())/1000.0,
		float64(inferenceTime.Microseconds())/1000.0,
		float64(serializationTime.Microseconds())/1000.0)
}

func performInference(evaluator *ckks.Evaluator, features []*rlwe.Ciphertext) (*rlwe.Ciphertext, error) {
	startAlign := time.Now()

	// 레벨 맞추기: 모든 암호문을 최소 레벨로 통일
	minLevel := features[0].Level()
	for i := 1; i < len(features); i++ {
		if features[i].Level() < minLevel {
			minLevel = features[i].Level()
		}
	}

	for i := 0; i < len(features); i++ {
		if features[i].Level() > minLevel {
			dropCount := features[i].Level() - minLevel
			log.Printf("🔄 Dropping %d level(s) for feature %d", dropCount, i)
			evaluator.DropLevel(features[i], dropCount)
		}
	}

	alignTime := time.Since(startAlign)
	log.Printf("📊 All features aligned to level %d (%.2f ms)", minLevel, float64(alignTime.Microseconds())/1000.0)

	startWeightedSum := time.Now()

	// Use actual slot count (4096 for LogN=13) instead of MaxSlots (8192)
	// Client encrypts with LogSlots=12, so we must match that
	actualSlots := params.MaxSlots() / 2 // 4096 for LogN=13
	values := make([]complex128, actualSlots)
	values[0] = complex(model.Weights[0], 0)
	weightPt := ckks.NewPlaintext(params, features[0].Level())
	encoder.Encode(values, weightPt)

	result, err := evaluator.MulNew(features[0], weightPt)
	if err != nil {
		return nil, fmt.Errorf("multiplication failed: %v", err)
	}
	// ✅ Rescaling 필수: 스케일 정규화
	if err := evaluator.Rescale(result, result); err != nil {
		return nil, fmt.Errorf("rescaling failed: %v", err)
	}
	log.Printf("✅ First weight mul + rescale: Level=%d", result.Level())

	for i := 1; i < len(features); i++ {
		for j := range values {
			values[j] = 0
		}
		values[0] = complex(model.Weights[i], 0)
		weightPt := ckks.NewPlaintext(params, features[i].Level())
		encoder.Encode(values, weightPt)

		weightedFeature, err := evaluator.MulNew(features[i], weightPt)
		if err != nil {
			return nil, fmt.Errorf("multiplication failed at feature %d: %v", i, err)
		}
		// ✅ Rescaling 필수
		if err := evaluator.Rescale(weightedFeature, weightedFeature); err != nil {
			return nil, fmt.Errorf("rescaling failed at feature %d: %v", i, err)
		}

		// 덧셈 전 레벨 맞추기
		if result.Level() != weightedFeature.Level() {
			if result.Level() > weightedFeature.Level() {
				evaluator.DropLevel(result, result.Level()-weightedFeature.Level())
			} else {
				evaluator.DropLevel(weightedFeature, weightedFeature.Level()-result.Level())
			}
		}

		if err := evaluator.Add(result, weightedFeature, result); err != nil {
			return nil, fmt.Errorf("addition failed at feature %d: %v", i, err)
		}
	}

	// Add bias with correct scale matching
	// After rescale, we need to scale bias by the inverse of the rescale factor
	for j := range values {
		values[j] = 0
	}

	// Scale bias value to match the post-rescale scale
	// result.Scale after rescale is DefaultScale / Q[dropped_level]
	scaleFactor := float64(result.Scale.Uint64()) / float64(params.DefaultScale().Uint64())
	values[0] = complex(model.Bias*scaleFactor, 0)

	biasPt := ckks.NewPlaintext(params, result.Level())
	if err := encoder.Encode(values, biasPt); err != nil {
		return nil, fmt.Errorf("bias encoding failed: %v", err)
	}

	if err := evaluator.Add(result, biasPt, result); err != nil {
		return nil, fmt.Errorf("bias addition failed: %v", err)
	}

	weightedSumTime := time.Since(startWeightedSum)
	log.Printf("⏱️  Weighted sum computation: %.2f ms", float64(weightedSumTime.Microseconds())/1000.0)

	startSigmoid := time.Now()
	log.Printf("🔐 Applying sigmoid approximation (CreditScoring-3)...")

	// Log pre-sigmoid noise budget
	logitLevel := result.Level()
	log.Printf("📉 Noise Budget Before Sigmoid: Level=%d/%d (%.1f%% remaining)",
		logitLevel, params.MaxLevel(), float64(logitLevel)/float64(params.MaxLevel())*100.0)

	// Use optimized CreditScoring sigmoid for [-3, -1] range (0.3% error)
	sigmoidApprox := sigmoid.NewCreditScoringApprox(3)
	score, err := sigmoidApprox.Evaluate(evaluator, result, params)
	if err != nil {
		return nil, fmt.Errorf("sigmoid evaluation failed: %v", err)
	}

	// Log post-sigmoid noise budget
	finalLevel := score.Level()
	levelsConsumed := logitLevel - finalLevel
	log.Printf("📉 Noise Budget After Sigmoid: Level=%d/%d (consumed %d levels)",
		finalLevel, params.MaxLevel(), levelsConsumed)

	// Warn if noise budget exhausted
	if finalLevel == 0 {
		log.Printf("⚠️  WARNING: Noise budget exhausted (Level=0). No room for additional operations.")
	} else if finalLevel < 0 {
		log.Printf("🔴 CRITICAL: Noise budget exceeded (Level=%d). Result may be corrupted!", finalLevel)
	}

	// Log estimated noise level based on depth
	estimatedNoise := float64(params.MaxLevel()-finalLevel) / float64(params.MaxLevel()) * 100.0
	log.Printf("📊 Estimated Noise Level: %.1f%% (based on level consumption)", estimatedNoise)

	sigmoidTime := time.Since(startSigmoid)
	log.Printf("⏱️  Sigmoid approximation: %.2f ms", float64(sigmoidTime.Microseconds())/1000.0)

	return score, nil
}

// performPackedInference: 하나의 암호문에 packed된 여러 피처로 추론 수행
// Hadamard product (element-wise multiplication) + Sum 방식 사용
func performPackedInference(evaluator *ckks.Evaluator, packedCt *rlwe.Ciphertext) (*rlwe.Ciphertext, error) {
	startWeightedSum := time.Now()

	// 1. weights를 벡터로 인코딩 (첫 N개 슬롯에 배치)
	values := make([]complex128, params.MaxSlots())
	for i := 0; i < len(model.Weights); i++ {
		values[i] = complex(model.Weights[i], 0)
	}
	weightPt := ckks.NewPlaintext(params, packedCt.Level())
	encoder.Encode(values, weightPt)

	// 2. Hadamard product: weights * features (element-wise)
	weightedCt, err := evaluator.MulNew(packedCt, weightPt)
	if err != nil {
		return nil, fmt.Errorf("hadamard product failed: %v", err)
	}

	// Rescaling
	if err := evaluator.Rescale(weightedCt, weightedCt); err != nil {
		return nil, fmt.Errorf("rescaling failed: %v", err)
	}
	log.Printf("✅ Hadamard product + rescale: Level=%d", weightedCt.Level())

	// 3. Sum first 5 slots using rotations
	// We accumulate: ct[0] + ct[1] + ct[2] + ct[3] + ct[4]
	result := weightedCt.CopyNew()

	// Rotate by 1: add slots [1,2,3,4,...] to result
	rotated1, err := evaluator.RotateNew(weightedCt, 1)
	if err != nil {
		return nil, fmt.Errorf("rotation by 1 failed: %v", err)
	}
	if err := evaluator.Add(result, rotated1, result); err != nil {
		return nil, fmt.Errorf("addition failed: %v", err)
	}

	// Rotate by 2: add slots [2,3,4,5,...] to result
	rotated2, err := evaluator.RotateNew(weightedCt, 2)
	if err != nil {
		return nil, fmt.Errorf("rotation by 2 failed: %v", err)
	}
	if err := evaluator.Add(result, rotated2, result); err != nil {
		return nil, fmt.Errorf("addition failed: %v", err)
	}

	// Rotate by 3: add slots [3,4,5,6,...] to result
	rotated3, err := evaluator.RotateNew(weightedCt, 3)
	if err != nil {
		return nil, fmt.Errorf("rotation by 3 failed: %v", err)
	}
	if err := evaluator.Add(result, rotated3, result); err != nil {
		return nil, fmt.Errorf("addition failed: %v", err)
	}

	// Rotate by 4: add slots [4,5,6,7,...] to result
	rotated4, err := evaluator.RotateNew(weightedCt, 4)
	if err != nil {
		return nil, fmt.Errorf("rotation by 4 failed: %v", err)
	}
	if err := evaluator.Add(result, rotated4, result); err != nil {
		return nil, fmt.Errorf("addition failed: %v", err)
	}

	// Now result[0] = sum of first 5 weighted features

	// 4. Add bias
	biasValues := make([]complex128, params.MaxSlots())
	biasValues[0] = complex(model.Bias, 0)
	biasPt := ckks.NewPlaintext(params, result.Level())
	encoder.Encode(biasValues, biasPt)
	evaluator.Add(result, biasPt, result)

	weightedSumTime := time.Since(startWeightedSum)
	log.Printf("⏱️  Packed weighted sum (Hadamard + Rotate): %.2f ms", float64(weightedSumTime.Microseconds())/1000.0)

	// 5. Apply sigmoid
	startSigmoid := time.Now()
	log.Printf("🔐 Applying sigmoid approximation (CreditScoring-3)...")

	logitLevel := result.Level()
	log.Printf("📉 Noise Budget Before Sigmoid: Level=%d/%d (%.1f%% remaining)",
		logitLevel, params.MaxLevel(), float64(logitLevel)/float64(params.MaxLevel())*100.0)

	sigmoidApprox := sigmoid.NewCreditScoringApprox(3)
	score, err := sigmoidApprox.Evaluate(evaluator, result, params)
	if err != nil {
		return nil, fmt.Errorf("sigmoid evaluation failed: %v", err)
	}

	finalLevel := score.Level()
	levelsConsumed := logitLevel - finalLevel
	log.Printf("📉 Noise Budget After Sigmoid: Level=%d/%d (consumed %d levels)",
		finalLevel, params.MaxLevel(), levelsConsumed)

	if finalLevel == 0 {
		log.Printf("⚠️  WARNING: Noise budget exhausted (Level=0).")
	} else if finalLevel < 0 {
		log.Printf("🔴 CRITICAL: Noise budget exceeded (Level=%d)!", finalLevel)
	}

	sigmoidTime := time.Since(startSigmoid)
	log.Printf("⏱️  Sigmoid approximation: %.2f ms", float64(sigmoidTime.Microseconds())/1000.0)

	return score, nil
}

// packedInferenceHandler: packed ciphertext를 처리하는 핸들러
func packedInferenceHandler(w http.ResponseWriter, r *http.Request) {
	startTotal := time.Now()
	log.Println("📥 Received packed inference request")

	var req PackedInferenceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("❌ ERROR: Failed to decode request: %v", err)
		http.Error(w, fmt.Sprintf("Invalid JSON: %v", err), http.StatusBadRequest)
		return
	}

	// Deserialize RLK
	rlkBytes, err := base64.StdEncoding.DecodeString(req.RelinearizationKey)
	if err != nil {
		log.Printf("❌ ERROR: Failed to decode RLK: %v", err)
		http.Error(w, "Invalid relinearization key encoding", http.StatusBadRequest)
		return
	}

	rlk := new(rlwe.RelinearizationKey)
	if err := rlk.UnmarshalBinary(rlkBytes); err != nil {
		log.Printf("❌ ERROR: Failed to unmarshal RLK: %v", err)
		http.Error(w, "Invalid relinearization key format", http.StatusBadRequest)
		return
	}
	log.Printf("✅ Received RLK: %d bytes", len(rlkBytes))

	// Deserialize Galois keys for rotations
	gkBytes, err := base64.StdEncoding.DecodeString(req.GaloisKey)
	if err != nil {
		log.Printf("❌ ERROR: Failed to decode Galois key: %v", err)
		http.Error(w, "Invalid Galois key encoding", http.StatusBadRequest)
		return
	}

	// Deserialize multiple Galois keys from combined buffer
	var galKeys []*rlwe.GaloisKey
	offset := 0
	for offset < len(gkBytes) {
		// Read length prefix (4 bytes)
		if offset+4 > len(gkBytes) {
			log.Printf("❌ ERROR: Invalid Galois key buffer format")
			http.Error(w, "Invalid Galois key format", http.StatusBadRequest)
			return
		}
		keyLen := int(gkBytes[offset])<<24 | int(gkBytes[offset+1])<<16 | int(gkBytes[offset+2])<<8 | int(gkBytes[offset+3])
		offset += 4

		// Read key data
		if offset+keyLen > len(gkBytes) {
			log.Printf("❌ ERROR: Invalid Galois key data length")
			http.Error(w, "Invalid Galois key format", http.StatusBadRequest)
			return
		}
		gk := new(rlwe.GaloisKey)
		if err := gk.UnmarshalBinary(gkBytes[offset : offset+keyLen]); err != nil {
			log.Printf("❌ ERROR: Failed to unmarshal Galois key: %v", err)
			http.Error(w, "Invalid Galois key format", http.StatusBadRequest)
			return
		}
		galKeys = append(galKeys, gk)
		offset += keyLen
	}
	log.Printf("✅ Received %d Galois keys: %d bytes total", len(galKeys), len(gkBytes))

	// Create evaluator with client's RLK and Galois keys
	evk := rlwe.NewMemEvaluationKeySet(rlk, galKeys...)
	evaluator := ckks.NewEvaluator(params, evk)

	// Deserialize packed ciphertext
	startDeserialization := time.Now()

	if len(req.EncryptedVector) > MaxCiphertextSize {
		log.Printf("❌ ERROR: Packed vector exceeds size limit")
		http.Error(w, "Packed vector exceeds maximum size", http.StatusBadRequest)
		return
	}

	ctBytes, err := base64.StdEncoding.DecodeString(req.EncryptedVector)
	if err != nil {
		log.Printf("❌ ERROR: Failed to decode packed vector: %v", err)
		http.Error(w, "Failed to decode packed vector", http.StatusBadRequest)
		return
	}

	packedCt := new(rlwe.Ciphertext)
	if err := packedCt.UnmarshalBinary(ctBytes); err != nil {
		log.Printf("❌ ERROR: Failed to unmarshal packed ciphertext: %v", err)
		http.Error(w, "Invalid packed ciphertext", http.StatusBadRequest)
		return
	}

	if packedCt.Level() < 0 || packedCt.Level() > params.MaxLevel() {
		log.Printf("❌ ERROR: Invalid ciphertext level %d", packedCt.Level())
		http.Error(w, "Invalid ciphertext level", http.StatusBadRequest)
		return
	}

	deserializationTime := time.Since(startDeserialization)
	log.Printf("✅ Packed Vector: Level=%d, Size=%d bytes", packedCt.Level(), len(ctBytes))
	log.Printf("⏱️  Deserialization: %.2f ms", float64(deserializationTime.Microseconds())/1000.0)

	// Perform inference
	startInference := time.Now()
	result, err := performPackedInference(evaluator, packedCt)
	if err != nil {
		log.Printf("❌ ERROR: Packed inference failed: %v", err)
		http.Error(w, fmt.Sprintf("Packed inference failed: %v", err), http.StatusInternalServerError)
		return
	}
	inferenceTime := time.Since(startInference)
	log.Printf("⏱️  Packed inference computation: %.2f ms", float64(inferenceTime.Microseconds())/1000.0)

	// Serialize result
	startSerialization := time.Now()
	resultBytes, err := result.MarshalBinary()
	if err != nil {
		log.Printf("❌ ERROR: Failed to marshal result: %v", err)
		http.Error(w, "Failed to marshal result", http.StatusInternalServerError)
		return
	}

	resultB64 := base64.StdEncoding.EncodeToString(resultBytes)
	serializationTime := time.Since(startSerialization)
	log.Printf("⏱️  Serialization: %.2f ms (%d bytes)",
		float64(serializationTime.Microseconds())/1000.0, len(resultBytes))

	response := InferenceResponse{
		EncryptedScore: resultB64,
		Timestamp:      time.Now().Unix(),
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("❌ ERROR: Failed to encode response: %v", err)
		return
	}

	totalTime := time.Since(startTotal)
	log.Printf("✅ Packed inference completed - Total: %.2f ms (Deser: %.2f ms, Compute: %.2f ms, Ser: %.2f ms)",
		float64(totalTime.Microseconds())/1000.0,
		float64(deserializationTime.Microseconds())/1000.0,
		float64(inferenceTime.Microseconds())/1000.0,
		float64(serializationTime.Microseconds())/1000.0)
}

func main() {
	router := mux.NewRouter()

	router.HandleFunc("/health", healthHandler).Methods("GET")
	router.HandleFunc("/api/inference", inferenceHandler).Methods("POST", "OPTIONS")
	router.HandleFunc("/api/inference-packed", packedInferenceHandler).Methods("POST", "OPTIONS")

	handler := enableCORS(router)

	port := ":8080"

	// HTTPS 모드 결정: TLS 인증서 파일이 존재하면 HTTPS, 없으면 HTTP
	certFile := "server.crt"
	keyFile := "server.key"
	useHTTPS := fileExists(certFile) && fileExists(keyFile)

	if useHTTPS {
		log.Printf("🔒 Server starting with HTTPS on https://localhost%s", port)
		log.Printf("📊 Model weights: %v, bias: %v", model.Weights, model.Bias)
		log.Printf("🔐 Ready to perform encrypted inference over TLS")
		log.Printf("⚠️  Using self-signed certificate (browsers will show warnings)")

		if err := http.ListenAndServeTLS(port, certFile, keyFile, handler); err != nil {
			log.Fatal(err)
		}
	} else {
		log.Printf("⚠️  Server starting with HTTP on http://localhost%s", port)
		log.Printf("   (No TLS certificates found. Generate with: ./generate_cert.sh)")
		log.Printf("📊 Model weights: %v, bias: %v", model.Weights, model.Bias)
		log.Printf("🔐 Ready to perform encrypted inference")

		if err := http.ListenAndServe(port, handler); err != nil {
			log.Fatal(err)
		}
	}
}

// fileExists 는 파일이 존재하는지 확인
func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}
