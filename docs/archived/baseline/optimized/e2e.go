package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/tuneinsight/lattigo/v6/core/rlwe"
	"github.com/tuneinsight/lattigo/v6/schemes/ckks"
)

// Test data representing realistic credit applicants
type TestCase struct {
	Name        string
	Age         float64 // Normalized: 18-80 → 0.0-1.0
	Income      float64 // Normalized: 0-200k → 0.0-1.0
	LoanAmount  float64 // Normalized: 0-100k → 0.0-1.0
	CreditScore float64 // Normalized: 300-850 → 0.0-1.0
	DebtRatio   float64 // Normalized: 0.0-1.0
	Expected    string  // Expected credit rating
}

var testCases = []TestCase{
	{
		Name:        "Good Credit - High Income",
		Age:         0.5,  // 49 years old
		Income:      0.75, // $150k/year
		LoanAmount:  0.3,  // $30k loan
		CreditScore: 0.85, // 762 score
		DebtRatio:   0.2,  // 20% debt ratio
		Expected:    "Good",
	},
	{
		Name:        "Poor Credit - Low Income",
		Age:         0.3, // 36 years old
		Income:      0.2, // $40k/year
		LoanAmount:  0.6, // $60k loan
		CreditScore: 0.3, // 465 score
		DebtRatio:   0.8, // 80% debt ratio
		Expected:    "Poor",
	},
	{
		Name:        "Excellent Credit - High Income Low Debt",
		Age:         0.6,  // 55 years old
		Income:      0.95, // $190k/year
		LoanAmount:  0.2,  // $20k loan
		CreditScore: 0.95, // 822 score
		DebtRatio:   0.1,  // 10% debt ratio
		Expected:    "Excellent",
	},
	{
		Name:        "Average Credit - Middle Income",
		Age:         0.4, // 42 years old
		Income:      0.5, // $100k/year
		LoanAmount:  0.4, // $40k loan
		CreditScore: 0.6, // 630 score
		DebtRatio:   0.4, // 40% debt ratio
		Expected:    "Fair",
	},
	{
		Name:        "Young Professional - Low History",
		Age:         0.15, // 27 years old
		Income:      0.6,  // $120k/year
		LoanAmount:  0.5,  // $50k loan
		CreditScore: 0.55, // 602 score
		DebtRatio:   0.35, // 35% debt ratio
		Expected:    "Fair",
	},
}

type InferenceRequest struct {
	EncryptedFeatures  []string `json:"encryptedFeatures"`
	RelinearizationKey string   `json:"relinearizationKey"`
}

type InferenceResponse struct {
	EncryptedScore string `json:"encryptedScore"`
	Timestamp      int64  `json:"timestamp"`
}

func main() {
	fmt.Println("🧪 CKKS Credit Scoring E2E Test")
	fmt.Println("================================\n")

	// Initialize CKKS parameters (must match backend and WASM)
	params, err := ckks.NewParametersFromLiteral(ckks.ParametersLiteral{
		LogN: 14,
		LogQ:            []int{60, 40, 40, 40, 40, 60}, // MaxLevel=5
		LogP:            []int{61},
		LogDefaultScale: 40,
	})
	if err != nil {
		panic(fmt.Sprintf("Failed to create parameters: %v", err))
	}

	fmt.Printf("📊 CKKS Parameters: LogN=%d, MaxLevel=%d, MaxSlots=%d\n\n",
		params.LogN(), params.MaxLevel(), params.MaxSlots())

	// Generate keys
	fmt.Println("🔑 Generating keys...")
	startKeygen := time.Now()
	kgen := rlwe.NewKeyGenerator(params)
	sk := kgen.GenSecretKeyNew()
	pk := kgen.GenPublicKeyNew(sk)
	rlk := kgen.GenRelinearizationKeyNew(sk)
	keygenTime := time.Since(startKeygen)
	fmt.Printf("✅ Keys generated in %.2fms\n\n", float64(keygenTime.Microseconds())/1000.0)

	// Initialize encoder, encryptor, decryptor
	encoder := ckks.NewEncoder(params)
	encryptor := ckks.NewEncryptor(params, pk)
	decryptor := ckks.NewDecryptor(params, sk)

	// Serialize RLK for backend
	rlkBytes, err := rlk.MarshalBinary()
	if err != nil {
		panic(fmt.Sprintf("Failed to serialize RLK: %v", err))
	}
	rlkBase64 := base64.StdEncoding.EncodeToString(rlkBytes)
	fmt.Printf("📦 Relinearization key serialized: %d bytes\n\n", len(rlkBytes))

	// Run test cases
	passCount := 0
	failCount := 0

	// Aggregate metrics
	var totalEncryptTime, totalBackendTime, totalDecryptTime time.Duration
	var totalCiphertextSize, totalNetworkSize int64

	for i, tc := range testCases {
		fmt.Printf("Test %d/%d: %s\n", i+1, len(testCases), tc.Name)
		fmt.Println(strings.Repeat("-", 60))

		// Prepare features
		features := []float64{
			tc.Age,
			tc.Income,
			tc.LoanAmount,
			tc.CreditScore,
			tc.DebtRatio,
		}

		// Encrypt features
		fmt.Println("🔒 Encrypting features...")
		startEncrypt := time.Now()
		encryptedFeatures := make([]string, len(features))
		var encryptedBytesTotal int
		for j, value := range features {
			// Create plaintext with value in first slot
			values := make([]complex128, params.MaxSlots())
			values[0] = complex(value, 0)
			pt := ckks.NewPlaintext(params, params.MaxLevel())
			encoder.Encode(values, pt)

			// Encrypt
			ct, err := encryptor.EncryptNew(pt)
			if err != nil {
				panic(fmt.Sprintf("Encryption failed: %v", err))
			}

			// Serialize
			ctBytes, err := ct.MarshalBinary()
			if err != nil {
				panic(fmt.Sprintf("Serialization failed: %v", err))
			}

			encryptedFeatures[j] = base64.StdEncoding.EncodeToString(ctBytes)
			encryptedBytesTotal += len(ctBytes)
			fmt.Printf("  Feature %d: %.4f → %d bytes (%.2f KB, Level=%d)\n",
				j+1, value, len(ctBytes), float64(len(ctBytes))/1024.0, ct.Level())
		}
		encryptTime := time.Since(startEncrypt)
		totalEncryptTime += encryptTime
		totalCiphertextSize += int64(encryptedBytesTotal)
		fmt.Printf("✅ Encryption completed in %.2fms (Total: %.2f KB)\n\n",
			float64(encryptTime.Microseconds())/1000.0, float64(encryptedBytesTotal)/1024.0)

		// Send to backend
		fmt.Println("📡 Sending to backend...")
		startBackend := time.Now()

		reqBody := InferenceRequest{
			EncryptedFeatures:  encryptedFeatures,
			RelinearizationKey: rlkBase64,
		}
		reqJSON, _ := json.Marshal(reqBody)
		requestSize := len(reqJSON)
		totalNetworkSize += int64(requestSize)

		resp, err := http.Post("http://localhost:8080/api/inference",
			"application/json", bytes.NewBuffer(reqJSON))
		if err != nil {
			fmt.Printf("❌ Backend request failed: %v\n\n", err)
			failCount++
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			fmt.Printf("❌ Backend returned error %d: %s\n\n", resp.StatusCode, string(body))
			failCount++
			continue
		}

		var inferenceResp InferenceResponse
		if err := json.NewDecoder(resp.Body).Decode(&inferenceResp); err != nil {
			fmt.Printf("❌ Failed to decode response: %v\n\n", err)
			failCount++
			continue
		}

		backendTime := time.Since(startBackend)
		totalBackendTime += backendTime
		fmt.Printf("✅ Backend inference completed in %.2fms (Request: %.2f KB)\n\n",
			float64(backendTime.Microseconds())/1000.0, float64(requestSize)/1024.0)

		// Decrypt result
		fmt.Println("🔓 Decrypting result...")
		startDecrypt := time.Now()

		scoreBytes, err := base64.StdEncoding.DecodeString(inferenceResp.EncryptedScore)
		if err != nil {
			fmt.Printf("❌ Failed to decode score: %v\n\n", err)
			failCount++
			continue
		}

		scoreCt := new(rlwe.Ciphertext)
		if err := scoreCt.UnmarshalBinary(scoreBytes); err != nil {
			fmt.Printf("❌ Failed to deserialize score: %v\n\n", err)
			failCount++
			continue
		}

		scorePt := decryptor.DecryptNew(scoreCt)
		scoreValues := make([]complex128, params.MaxSlots())
		encoder.Decode(scorePt, scoreValues)
		score := real(scoreValues[0])

		decryptTime := time.Since(startDecrypt)
		totalDecryptTime += decryptTime
		responseSize := len(scoreBytes)
		totalNetworkSize += int64(responseSize)
		fmt.Printf("✅ Decryption completed in %.2fms (Response: %.2f KB)\n",
			float64(decryptTime.Microseconds())/1000.0, float64(responseSize)/1024.0)

		// Evaluate result
		rating := getCreditRating(score)
		totalTime := encryptTime + backendTime + decryptTime

		fmt.Printf("\n📊 Results:\n")
		fmt.Printf("  Raw Score: %.6f\n", score)
		fmt.Printf("  Probability: %.2f%%\n", score*100)
		fmt.Printf("  Credit Rating: %s\n", rating)
		fmt.Printf("  Expected: %s\n", tc.Expected)
		fmt.Printf("  Total E2E Time: %.2fms\n", float64(totalTime.Microseconds())/1000.0)
		fmt.Printf("  Total Network: %.2f KB\n", float64(requestSize+responseSize)/1024.0)

		// Validate
		if score >= 0 && score <= 1 {
			fmt.Println("✅ PASS - Score in valid range [0, 1]")
			passCount++
		} else {
			fmt.Printf("❌ FAIL - Score %.6f out of range!\n", score)
			failCount++
		}

		fmt.Println()
	}

	// Summary
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Printf("🎯 Test Summary: %d/%d passed (%.1f%%)\n",
		passCount, len(testCases), float64(passCount)/float64(len(testCases))*100)
	if failCount > 0 {
		fmt.Printf("❌ %d tests failed\n", failCount)
	}

	// Aggregate Performance Metrics
	if passCount > 0 {
		fmt.Println("\n" + strings.Repeat("=", 60))
		fmt.Println("📊 Aggregate Performance Metrics (Baseline)")
		fmt.Println(strings.Repeat("=", 60))
		fmt.Printf("Key Generation:     %.2fms\n", float64(keygenTime.Microseconds())/1000.0)
		fmt.Printf("Avg Encryption:     %.2fms per test\n", float64(totalEncryptTime.Microseconds())/1000.0/float64(passCount))
		fmt.Printf("Avg Backend:        %.2fms per test\n", float64(totalBackendTime.Microseconds())/1000.0/float64(passCount))
		fmt.Printf("Avg Decryption:     %.2fms per test\n", float64(totalDecryptTime.Microseconds())/1000.0/float64(passCount))
		fmt.Printf("Total E2E Time:     %.2fms avg per test\n",
			float64(totalEncryptTime.Microseconds()+totalBackendTime.Microseconds()+totalDecryptTime.Microseconds())/1000.0/float64(passCount))
		fmt.Println(strings.Repeat("-", 60))
		fmt.Printf("Avg Ciphertext Size: %.2f KB per test (5 features)\n", float64(totalCiphertextSize)/1024.0/float64(passCount))
		fmt.Printf("Avg Network Traffic: %.2f KB per test (upload + download)\n", float64(totalNetworkSize)/1024.0/float64(passCount))
		fmt.Printf("RLK Size:            %.2f MB (sent once per test)\n", float64(len(rlkBytes))/1024.0/1024.0)
		fmt.Println(strings.Repeat("=", 60))
	}
}

func getCreditRating(score float64) string {
	if score >= 0.8 {
		return "Excellent"
	} else if score >= 0.6 {
		return "Good"
	} else if score >= 0.4 {
		return "Fair"
	} else if score >= 0.2 {
		return "Needs Improvement"
	}
	return "Poor"
}
