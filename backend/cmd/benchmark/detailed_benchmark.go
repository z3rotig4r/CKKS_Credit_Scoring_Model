package main

import (
	"fmt"
	"math"
	"time"

	"github.com/tuneinsight/lattigo/v6/core/rlwe"
	"github.com/tuneinsight/lattigo/v6/schemes/ckks"
)

// Detailed benchmark with homomorphic operations
func benchmarkHomomorphicOps(params ckks.Parameters) {
	fmt.Println("\n\n🔐 Homomorphic Operations Benchmark")
	fmt.Println("====================================")

	// Initialize CKKS components
	kgen := ckks.NewKeyGenerator(params)
	sk := kgen.GenSecretKeyNew()
	rlk := kgen.GenRelinearizationKeyNew(sk)
	evk := rlwe.NewMemEvaluationKeySet(rlk)

	encoder := ckks.NewEncoder(params)
	encryptor := ckks.NewEncryptor(params, sk)
	decryptor := ckks.NewDecryptor(params, sk)
	evaluator := ckks.NewEvaluator(params, evk)

	// Test case: Average user
	features := []float64{4.4, 1.0, 24.0, 1.0, 1.7}
	weights := productionModel.Weights
	bias := productionModel.Bias

	// Expected plaintext result
	expectedLogit := bias
	for i := range weights {
		expectedLogit += weights[i] * features[i]
	}

	fmt.Printf("\nTest Input: age=%.1f, loan/income=%.1f, debt/income=%.1f%%, credit=%.1fM, income=%.1fM\n",
		features[0]*10, features[1], features[2], features[3]/10, features[4]/10)
	fmt.Printf("Expected Logit: %.6f\n\n", expectedLogit)

	// ========== Step 1: Key Generation ==========
	fmt.Println("Step 1: Key Generation")
	startKey := time.Now()
	kgen2 := ckks.NewKeyGenerator(params)
	sk2 := kgen2.GenSecretKeyNew()
	_ = kgen2.GenRelinearizationKeyNew(sk2)
	keyTime := time.Since(startKey)
	fmt.Printf("  Time: %.2f ms\n", float64(keyTime.Microseconds())/1000.0)

	// ========== Step 2: Encryption ==========
	fmt.Println("\nStep 2: Feature Encryption (5 features)")
	startEnc := time.Now()
	featureCts := make([]*rlwe.Ciphertext, len(features))
	for i, val := range features {
		values := make([]complex128, params.MaxSlots())
		for j := range values {
			values[j] = complex(val, 0)
		}
		pt := ckks.NewPlaintext(params, params.MaxLevel())
		encoder.Encode(values, pt)
		featureCts[i], _ = encryptor.EncryptNew(pt)
	}
	encTime := time.Since(startEnc)
	fmt.Printf("  Time: %.2f ms (%.2f ms per feature)\n",
		float64(encTime.Microseconds())/1000.0,
		float64(encTime.Microseconds())/1000.0/float64(len(features)))

	// ========== Step 3: Homomorphic Computation ==========
	fmt.Println("\nStep 3: Homomorphic Weighted Sum")
	startComp := time.Now()

	// Start with bias
	biasValues := make([]complex128, params.MaxSlots())
	for i := range biasValues {
		biasValues[i] = complex(bias, 0)
	}
	biasPt := ckks.NewPlaintext(params, params.MaxLevel())
	encoder.Encode(biasValues, biasPt)
	result, _ := encryptor.EncryptNew(biasPt)

	mulCount := 0
	addCount := 0

	// Add weighted features
	for i, w := range weights {
		weightValues := make([]complex128, params.MaxSlots())
		for j := range weightValues {
			weightValues[j] = complex(w, 0)
		}
		weightPt := ckks.NewPlaintext(params, featureCts[i].Level())
		encoder.Encode(weightValues, weightPt)

		temp := featureCts[i].CopyNew()
		evaluator.Mul(temp, weightPt, temp)
		mulCount++
		evaluator.Rescale(temp, temp)
		evaluator.Add(result, temp, result)
		addCount++
	}

	compTime := time.Since(startComp)
	fmt.Printf("  Time: %.2f ms\n", float64(compTime.Microseconds())/1000.0)
	fmt.Printf("  Operations: %d multiplications, %d additions\n", mulCount, addCount)
	fmt.Printf("  Final ciphertext level: %d (started at %d)\n", result.Level(), params.MaxLevel())

	// ========== Step 4: Decryption ==========
	fmt.Println("\nStep 4: Decryption")
	startDec := time.Now()
	decrypted := decryptor.DecryptNew(result)
	decoded := make([]complex128, params.MaxSlots())
	encoder.Decode(decrypted, decoded)
	encryptedLogit := real(decoded[0])
	decTime := time.Since(startDec)
	fmt.Printf("  Time: %.2f ms\n", float64(decTime.Microseconds())/1000.0)

	// ========== Results ==========
	error := math.Abs(expectedLogit - encryptedLogit)
	fmt.Println("\n" + string(make([]byte, 60)))
	fmt.Println("Results:")
	fmt.Printf("  Expected:  %.10f\n", expectedLogit)
	fmt.Printf("  Encrypted: %.10f\n", encryptedLogit)
	fmt.Printf("  Error:     %.2e (%.6f%%)\n", error, error/math.Abs(expectedLogit)*100)
	fmt.Println()

	// ========== Performance Summary ==========
	totalTime := keyTime + encTime + compTime + decTime
	fmt.Println("Performance Summary:")
	fmt.Printf("  Key Generation:  %6.2f ms (%5.1f%%)\n",
		float64(keyTime.Microseconds())/1000.0,
		float64(keyTime)/float64(totalTime)*100)
	fmt.Printf("  Encryption:      %6.2f ms (%5.1f%%)\n",
		float64(encTime.Microseconds())/1000.0,
		float64(encTime)/float64(totalTime)*100)
	fmt.Printf("  Computation:     %6.2f ms (%5.1f%%)\n",
		float64(compTime.Microseconds())/1000.0,
		float64(compTime)/float64(totalTime)*100)
	fmt.Printf("  Decryption:      %6.2f ms (%5.1f%%)\n",
		float64(decTime.Microseconds())/1000.0,
		float64(decTime)/float64(totalTime)*100)
	fmt.Println("  " + string(make([]byte, 35)))
	fmt.Printf("  Total:           %6.2f ms\n", float64(totalTime.Microseconds())/1000.0)

	// ========== Security Level ==========
	fmt.Println("\nSecurity:")
	fmt.Printf("  Ring Dimension:  %d\n", 1<<params.LogN())
	fmt.Printf("  Modulus Bits:    %.0f\n", params.LogQP())
	fmt.Printf("  Est. Security:   ~128 bits (post-quantum)\n")
}
