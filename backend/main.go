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
)

type LogisticRegressionModel struct {
	Weights []float64
	Bias    float64
}

var model = LogisticRegressionModel{
	Weights: []float64{0.5, 0.3, 0.4, -0.2, 0.35, -0.15},
	Bias:    0.1,
}

type InferenceRequest struct {
	EncryptedFeatures []string `json:"encryptedFeatures"`
}

type InferenceResponse struct {
	EncryptedScore string `json:"encryptedScore"`
	Timestamp      int64  `json:"timestamp"`
}

func init() {
	var err error
	params, err = ckks.NewParametersFromLiteral(ckks.ParametersLiteral{
		LogN:            14,
		LogQ:            []int{60, 40, 40, 60},
		LogP:            []int{61},
		LogDefaultScale: 40,
	})
	if err != nil {
		panic(fmt.Sprintf("Failed to create CKKS parameters: %v", err))
	}

	evaluator = ckks.NewEvaluator(params, nil)
	encoder = ckks.NewEncoder(params)

	log.Printf("CKKS Parameters: LogN=%d, MaxLevel=%d, MaxSlots=%d\n",
		params.LogN(), params.MaxLevel(), params.MaxSlots())
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
	var req InferenceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
		return
	}

	if len(req.EncryptedFeatures) != 6 {
		http.Error(w, "Expected 6 encrypted features", http.StatusBadRequest)
		return
	}

	log.Printf("Received inference request with %d encrypted features", len(req.EncryptedFeatures))

	encryptedFeatures := make([]*rlwe.Ciphertext, len(req.EncryptedFeatures))
	for i, b64Str := range req.EncryptedFeatures {
		// 크기 제한 검증
		if len(b64Str) > MaxCiphertextSize {
			log.Printf("❌ Feature %d exceeds size limit: %d bytes", i, len(b64Str))
			http.Error(w, fmt.Sprintf("Feature %d exceeds maximum size", i), http.StatusBadRequest)
			return
		}

		ctBytes, err := base64.StdEncoding.DecodeString(b64Str)
		if err != nil {
			log.Printf("❌ Failed to decode feature %d: %v", i, err)
			http.Error(w, fmt.Sprintf("Failed to decode feature %d: %v", i, err), http.StatusBadRequest)
			return
		}

		// 역직렬화 크기 검증
		if len(ctBytes) > MaxCiphertextSize {
			log.Printf("❌ Decoded feature %d exceeds size limit: %d bytes", i, len(ctBytes))
			http.Error(w, fmt.Sprintf("Feature %d data too large", i), http.StatusBadRequest)
			return
		}

		// ✅ 올바른 역직렬화: 레벨 자동 복원
		ct := new(rlwe.Ciphertext)
		if err := ct.UnmarshalBinary(ctBytes); err != nil {
			log.Printf("❌ Failed to unmarshal ciphertext %d: %v", i, err)
			http.Error(w, fmt.Sprintf("Invalid ciphertext %d: %v", i, err), http.StatusBadRequest)
			return
		}

		// 암호문 유효성 검증
		if ct.Level() < 0 || ct.Level() > params.MaxLevel() {
			log.Printf("❌ Invalid ciphertext level %d (max: %d)", ct.Level(), params.MaxLevel())
			http.Error(w, fmt.Sprintf("Invalid ciphertext %d: bad level", i), http.StatusBadRequest)
			return
		}

		encryptedFeatures[i] = ct
		log.Printf("✅ Feature %d: Level=%d, Size=%d bytes", i, ct.Level(), len(ctBytes))
	}

	result, err := performInference(encryptedFeatures)
	if err != nil {
		http.Error(w, fmt.Sprintf("Inference failed: %v", err), http.StatusInternalServerError)
		return
	}

	resultBytes, err := result.MarshalBinary()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to marshal result: %v", err), http.StatusInternalServerError)
		return
	}

	resultB64 := base64.StdEncoding.EncodeToString(resultBytes)

	response := InferenceResponse{
		EncryptedScore: resultB64,
		Timestamp:      time.Now().Unix(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)

	log.Printf("Inference completed successfully")
}

func performInference(features []*rlwe.Ciphertext) (*rlwe.Ciphertext, error) {
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

	log.Printf("📊 All features aligned to level %d", minLevel)

	values := make([]complex128, params.MaxSlots())
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

	values[0] = complex(model.Bias, 0)
	biasPt := ckks.NewPlaintext(params, result.Level())
	encoder.Encode(values, biasPt)
	evaluator.Add(result, biasPt, result)

	log.Printf("Applying sigmoid approximation (Chebyshev-5)...")
	sigmoidApprox := sigmoid.NewChebyshevApprox(5)
	score, err := sigmoidApprox.Evaluate(evaluator, result, params)
	if err != nil {
		return nil, fmt.Errorf("sigmoid evaluation failed: %v", err)
	}

	return score, nil
}

func main() {
	router := mux.NewRouter()

	router.HandleFunc("/health", healthHandler).Methods("GET")
	router.HandleFunc("/api/inference", inferenceHandler).Methods("POST", "OPTIONS")

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
