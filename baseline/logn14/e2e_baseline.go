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
		Age:         0.4, // 43 years old
		Income:      0.5, // $100k/year
		LoanAmount:  0.4, // $40k loan
		CreditScore: 0.6, // 630 score
		DebtRatio:   0.4, // 40% debt ratio
		Expected:    "Average",
	},
	{
		Name:        "Young Professional - Low History",
		Age:         0.15, // 27 years old
		Income:      0.6,  // $120k/year
		LoanAmount:  0.5,  // $50k loan
		CreditScore: 0.55, // 602 score
		DebtRatio:   0.35, // 35% debt ratio
		Expected:    "Average",
	},
}

func main() {
	fmt.Println("🧪 CKKS Credit Scoring E2E Test - BASELINE (LogN=14)")
	fmt.Println("=====================================================")
	fmt.Println()

	// CKKS parameters - BASELINE (LogN=14)
	params, err := ckks.NewParametersFromLiteral(ckks.ParametersLiteral{
		LogN:            14,
		LogQ:            []int{60, 40, 40, 40, 40, 60}, // SAME 6 levels as optimized for fair comparison
		LogP:            []int{61},
		LogDefaultScale: 40,
	})
	if err != nil {
		panic(err)
	}

	fmt.Printf("📊 CKKS Parameters: LogN=%d, MaxLevel=%d, MaxSlots=%d\n\n",
		params.LogN(), params.MaxLevel(), params.MaxSlots())

	// Generate keys
	fmt.Println("🔑 Generating keys...")
	startKeygen := time.Now()
	kgen := ckks.NewKeyGenerator(params)
	sk := kgen.GenSecretKeyNew()
	rlk := kgen.GenRelinearizationKeyNew(sk)
	keygenTime := time.Since(startKeygen)
	fmt.Printf("✅ Keys generated in %.2fms\n\n", float64(keygenTime.Microseconds())/1000.0)

	// Serialize RLK
	rlkBytes, _ := rlk.MarshalBinary()
	rlkB64 := base64.StdEncoding.EncodeToString(rlkBytes)
	fmt.Printf("📦 Relinearization key serialized: %d bytes\n\n", len(rlkBytes))

	encoder := ckks.NewEncoder(params)
	encryptor := ckks.NewEncryptor(params, sk)
	decryptor := ckks.NewDecryptor(params, sk)

	backendURL := "http://localhost:8080/api/inference"

	passed := 0
	failed := 0
	totalE2E := 0.0
	totalNetwork := 0.0

	for i, tc := range testCases {
		fmt.Printf("Test %d/%d: %s\n", i+1, len(testCases), tc.Name)
		fmt.Println(strings.Repeat("-", 60))

		// Encrypt features
		fmt.Println("🔒 Encrypting features...")
		startEnc := time.Now()

		features := []float64{tc.Age, tc.Income, tc.LoanAmount, tc.CreditScore, tc.DebtRatio}
		ciphertexts := make([]string, len(features))
		totalEncSize := 0

		for j, f := range features {
			values := make([]complex128, params.MaxSlots())
			for k := range values {
				values[k] = complex(f, 0)
			}
			pt := ckks.NewPlaintext(params, params.MaxLevel())
			encoder.Encode(values, pt)
			ct, _ := encryptor.EncryptNew(pt)

			ctBytes, _ := ct.MarshalBinary()
			ciphertexts[j] = base64.StdEncoding.EncodeToString(ctBytes)

			fmt.Printf("  Feature %d: %.4f → %d bytes (%.2f KB, Level=%d)\n",
				j+1, f, len(ctBytes), float64(len(ctBytes))/1024, ct.Level())
			totalEncSize += len(ctBytes)
		}

		encTime := time.Since(startEnc)
		fmt.Printf("✅ Encryption completed in %.2fms (Total: %.2f KB)\n\n",
			float64(encTime.Microseconds())/1000.0, float64(totalEncSize)/1024)

		// Send to backend
		fmt.Println("📡 Sending to backend...")
		requestPayload := map[string]interface{}{
			"encryptedFeatures":  ciphertexts,
			"relinearizationKey": rlkB64,
		}
		requestJSON, _ := json.Marshal(requestPayload)
		requestSize := len(requestJSON)

		startBackend := time.Now()
		resp, err := http.Post(backendURL, "application/json", bytes.NewBuffer(requestJSON))
		if err != nil {
			fmt.Printf("❌ Backend request failed: %v\n\n", err)
			failed++
			continue
		}

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			fmt.Printf("❌ Backend returned error %d: %s\n\n", resp.StatusCode, string(body))
			resp.Body.Close()
			failed++
			continue
		}

		var response map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&response)
		resp.Body.Close()
		backendTime := time.Since(startBackend)

		fmt.Printf("✅ Backend inference completed in %.2fms (Request: %.2f KB)\n\n",
			float64(backendTime.Microseconds())/1000.0, float64(requestSize)/1024)

		// Decrypt result
		fmt.Println("🔓 Decrypting result...")
		startDec := time.Now()

		encryptedScore := response["encryptedScore"].(string)
		resultBytes, _ := base64.StdEncoding.DecodeString(encryptedScore)
		resultCt := &rlwe.Ciphertext{}
		resultCt.UnmarshalBinary(resultBytes)

		resultPt := decryptor.DecryptNew(resultCt)
		resultValues := make([]complex128, params.MaxSlots())
		encoder.Decode(resultPt, resultValues)
		score := real(resultValues[0])

		decTime := time.Since(startDec)
		responseSize := len(resultBytes)
		fmt.Printf("✅ Decryption completed in %.2fms (Response: %.2f KB)\n\n",
			float64(decTime.Microseconds())/1000.0, float64(responseSize)/1024)

		// Calculate metrics
		e2eTime := float64(encTime.Microseconds()+backendTime.Microseconds()+decTime.Microseconds()) / 1000.0
		networkSize := float64(requestSize+responseSize) / 1024

		// Determine rating
		rating := "Unknown"
		if score >= 0.7 {
			rating = "Excellent"
		} else if score >= 0.6 {
			rating = "Good"
		} else if score >= 0.45 {
			rating = "Fair"
		} else {
			rating = "Poor"
		}

		fmt.Println("📊 Results:")
		fmt.Printf("  Raw Score: %f\n", score)
		fmt.Printf("  Probability: %.2f%%\n", score*100)
		fmt.Printf("  Credit Rating: %s\n", rating)
		fmt.Printf("  Expected: %s\n", tc.Expected)
		fmt.Printf("  Total E2E Time: %.2fms\n", e2eTime)
		fmt.Printf("  Total Network: %.2f KB\n", networkSize)

		totalE2E += e2eTime
		totalNetwork += networkSize

		// Validate score range
		if score >= 0 && score <= 1 {
			fmt.Println("✅ PASS - Score in valid range [0, 1]")
			passed++
		} else {
			fmt.Println("❌ FAIL - Score out of range!")
			failed++
		}
		fmt.Println()
	}

	// Summary
	fmt.Println()
	fmt.Println(strings.Repeat("=", 60))
	fmt.Printf("🎯 Test Summary: %d/%d passed (%.1f%%)\n", passed, len(testCases), float64(passed)/float64(len(testCases))*100)
	if failed > 0 {
		fmt.Printf("❌ %d tests failed\n", failed)
	}
	fmt.Println()
	fmt.Printf("⚡ Average E2E Time: %.2fms\n", totalE2E/float64(len(testCases)))
	fmt.Printf("📦 Average Network: %.2f KB\n", totalNetwork/float64(len(testCases)))
	fmt.Println(strings.Repeat("=", 60))
}
