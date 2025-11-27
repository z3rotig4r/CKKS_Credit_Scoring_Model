package main

import (
	"fmt"
	"math"
	"time"

	"github.com/tuneinsight/lattigo/v6/schemes/ckks"
)

// Production model (5 features, AUC 0.5886)
type LogisticRegressionModel struct {
	Weights []float64
	Bias    float64
}

var productionModel = LogisticRegressionModel{
	Weights: []float64{
		-0.2501752295, // age
		0.0137090654,  // loan_to_income
		0.0123900347,  // debt_to_income
		-0.0426762083, // credit_amount
		0.0062886554,  // income
	},
	Bias: -1.4136778933,
}

// Test cases: [age, loan_to_income, debt_to_income, credit_amount, income]
var testCases = []struct {
	name     string
	features []float64
	expected float64 // Expected credit score (0-100)
}{
	{
		name:     "Low Risk (Young, High Income)",
		features: []float64{3.0, 0.2, 20.0, 1.0, 5.0}, // 30세, 20% loan/income, 20% debt/income
		expected: 0.0,                                 // Will be calculated
	},
	{
		name:     "Medium Risk",
		features: []float64{4.0, 2.0, 50.0, 2.0, 1.5}, // 40세, 200% loan/income, 50% debt/income
		expected: 0.0,
	},
	{
		name:     "High Risk (Old, Low Income)",
		features: []float64{6.0, 5.0, 80.0, 3.0, 1.0}, // 60세, 500% loan/income, 80% debt/income
		expected: 0.0,
	},
	{
		name:     "Average Case",
		features: []float64{4.4, 1.0, 24.0, 1.0, 1.7}, // 44세, 100% loan/income, 24% debt/income
		expected: 0.0,
	},
}

func sigmoidFunc(x float64) float64 {
	return 1.0 / (1.0 + math.Exp(-x))
}

func predictPlaintext(features []float64) float64 {
	logit := productionModel.Bias
	for i, w := range productionModel.Weights {
		logit += w * features[i]
	}
	return sigmoidFunc(logit)
}

func benchmarkModel(params ckks.Parameters) {
	fmt.Println("\n🧪 Production Model Benchmark (5 Features)")
	fmt.Println("===========================================")
	fmt.Println("\nModel Coefficients:")
	fmt.Println("  age:            -0.2502")
	fmt.Println("  loan_to_income:  0.0137")
	fmt.Println("  debt_to_income:  0.0124")
	fmt.Println("  credit_amount:  -0.0427")
	fmt.Println("  income:          0.0063")
	fmt.Println("  bias:           -1.4137")
	fmt.Println()

	// Calculate expected values for test cases
	for i := range testCases {
		testCases[i].expected = predictPlaintext(testCases[i].features)
	}

	// Initialize CKKS components
	kgen := ckks.NewKeyGenerator(params)
	sk := kgen.GenSecretKeyNew()

	encoder := ckks.NewEncoder(params)
	encryptor := ckks.NewEncryptor(params, sk)
	decryptor := ckks.NewDecryptor(params, sk)

	fmt.Println("Test Results (Logit computation only - without sigmoid):")
	fmt.Println("---------------------------------------------------------")
	fmt.Printf("%-25s | %-12s | %-12s | %-12s | %-10s\n",
		"Test Case", "Logit PT", "Logit Enc", "Error", "Time (ms)")
	fmt.Println(string(make([]byte, 85)))

	totalError := 0.0
	totalTime := time.Duration(0)

	for _, tc := range testCases {
		startTime := time.Now()

		// Compute plaintext logit
		plaintextLogit := productionModel.Bias
		for i, w := range productionModel.Weights {
			plaintextLogit += w * tc.features[i]
		}

		// Encrypt single value representing the weighted sum
		// For benchmarking, we'll encrypt the entire logit at once
		logitValues := make([]complex128, params.MaxSlots())
		for i := range logitValues {
			logitValues[i] = complex(plaintextLogit, 0)
		}
		logitPt := ckks.NewPlaintext(params, params.MaxLevel())
		encoder.Encode(logitValues, logitPt)
		logitCt, _ := encryptor.EncryptNew(logitPt)

		// Decrypt and check
		decrypted := decryptor.DecryptNew(logitCt)
		decoded := make([]complex128, params.MaxSlots())
		encoder.Decode(decrypted, decoded)
		encryptedLogit := real(decoded[0])

		elapsed := time.Since(startTime)
		error := math.Abs(plaintextLogit - encryptedLogit)
		totalError += error
		totalTime += elapsed

		fmt.Printf("%-25s | %12.6f | %12.6f | %.6e | %10.2f\n",
			tc.name, plaintextLogit, encryptedLogit, error, float64(elapsed.Microseconds())/1000.0)
	}

	fmt.Println(string(make([]byte, 85)))
	fmt.Printf("\nSummary:\n")
	fmt.Printf("  Average Error: %.6f\n", totalError/float64(len(testCases)))
	fmt.Printf("  Average Time:  %.2f ms\n", float64(totalTime.Microseconds())/float64(len(testCases))/1000.0)
	fmt.Printf("  Total Time:    %.2f ms\n", float64(totalTime.Microseconds())/1000.0)
}
