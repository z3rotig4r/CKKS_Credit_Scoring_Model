package main

import (
	"fmt"
	"math"
	"time"

	"github.com/tuneinsight/lattigo/v6/core/rlwe"
	"github.com/tuneinsight/lattigo/v6/schemes/ckks"
	"github.com/z3rotig4r/ckks_credit/backend/sigmoid"
)

func benchmarkSigmoidApproximations(params ckks.Parameters) {
	fmt.Println("\n\n📈 Sigmoid Approximation Methods Comparison")
	fmt.Println("============================================")

	// Initialize approximation methods
	methods := []sigmoid.Approximation{
		sigmoid.NewChebyshevApprox(3),
		sigmoid.NewChebyshevApprox(5),
		sigmoid.NewChebyshevApprox(7),
		sigmoid.NewMinimaxApprox(3),
		sigmoid.NewMinimaxApprox(5),
		sigmoid.NewMinimaxApprox(7),
		sigmoid.NewCompositeApprox(3),
		sigmoid.NewCreditScoringApprox(3), // ✅ Used in production
	}

	// Test points covering typical credit scoring logit range
	testPoints := []float64{
		-8.0, -6.0, -4.0, -3.0, -2.0, -1.5, -1.0, -0.5,
		0.0, 0.5, 1.0, 1.5, 2.0, 3.0, 4.0, 6.0, 8.0,
	}

	fmt.Printf("\nTest Range: [%.1f, %.1f] with %d points\n", testPoints[0], testPoints[len(testPoints)-1], len(testPoints))
	fmt.Println("(Typical credit scoring logits: -3 to 0)")
	fmt.Println()

	// Initialize CKKS components
	kgen := ckks.NewKeyGenerator(params)
	sk := kgen.GenSecretKeyNew()
	rlk := kgen.GenRelinearizationKeyNew(sk)
	evk := rlwe.NewMemEvaluationKeySet(rlk)

	encoder := ckks.NewEncoder(params)
	encryptor := ckks.NewEncryptor(params, sk)
	decryptor := ckks.NewDecryptor(params, sk)
	evaluator := ckks.NewEvaluator(params, evk)

	// Results table header
	fmt.Println("Results:")
	fmt.Println("--------")
	fmt.Printf("%-20s | %-12s | %-12s | %-12s | %-10s | %-8s\n",
		"Method", "Mean Error", "Max Error", "Std Dev", "Time (ms)", "Depth")
	fmt.Println(string(make([]byte, 100)))

	type Result struct {
		method   sigmoid.Approximation
		meanErr  float64
		maxErr   float64
		stdDev   float64
		duration time.Duration
	}

	results := make([]Result, 0, len(methods))

	for _, method := range methods {
		var totalError float64
		var maxError float64
		var errors []float64
		startTime := time.Now()
		actualSlots := params.MaxSlots() / 2 // Use actual usable slots

		for _, x := range testPoints {
			// Encrypt
			// Use actual slots (MaxSlots/2) to match client encryption
			values := make([]complex128, actualSlots)
			for i := range values {
				values[i] = complex(x, 0)
			}
			pt := ckks.NewPlaintext(params, params.MaxLevel())
			encoder.Encode(values, pt)
			ct, _ := encryptor.EncryptNew(pt)

			// Evaluate sigmoid approximation
			resultCt, err := method.Evaluate(evaluator, ct, params)
			if err != nil {
				fmt.Printf("Error evaluating %s: %v\n", method.Name(), err)
				continue
			}

			// Decrypt
			ptResult := decryptor.DecryptNew(resultCt)
			valuesResult := make([]complex128, actualSlots)
			encoder.Decode(ptResult, valuesResult)

			approxValue := real(valuesResult[0])
			trueValue := 1.0 / (1.0 + math.Exp(-x))
			errorVal := math.Abs(approxValue - trueValue)

			totalError += errorVal
			errors = append(errors, errorVal)
			if errorVal > maxError {
				maxError = errorVal
			}
		}

		duration := time.Since(startTime)
		meanErr := totalError / float64(len(testPoints))

		// Calculate standard deviation
		var variance float64
		for _, err := range errors {
			variance += math.Pow(err-meanErr, 2)
		}
		stdDev := math.Sqrt(variance / float64(len(errors)))

		results = append(results, Result{
			method:   method,
			meanErr:  meanErr,
			maxErr:   maxError,
			stdDev:   stdDev,
			duration: duration,
		})

		fmt.Printf("%-20s | %.6e | %.6e | %.6e | %10.2f | %8d\n",
			method.Name(),
			meanErr,
			maxError,
			stdDev,
			float64(duration.Microseconds())/1000.0,
			method.RequiredDepth())
	}

	fmt.Println(string(make([]byte, 100)))

	// Find best methods
	bestAccuracy := results[0]
	bestSpeed := results[0]
	bestBalanced := results[0]

	for _, r := range results {
		if r.meanErr < bestAccuracy.meanErr {
			bestAccuracy = r
		}
		if r.duration < bestSpeed.duration {
			bestSpeed = r
		}
		// Balanced score: weighted combination of accuracy and speed
		balancedScore := func(res Result) float64 {
			// Normalize: lower is better
			// Weight: 70% accuracy, 30% speed
			normalizedErr := res.meanErr / 0.01 // Target ~1% error
			normalizedTime := float64(res.duration.Microseconds()) / 1000000.0
			return 0.7*normalizedErr + 0.3*normalizedTime
		}
		if balancedScore(r) < balancedScore(bestBalanced) {
			bestBalanced = r
		}
	}

	// Recommendations
	fmt.Println("\n🏆 Recommendations:")
	fmt.Println("-------------------")
	fmt.Printf("Best Accuracy:  %s (Mean Error: %.6e, Max Error: %.6e)\n",
		bestAccuracy.method.Name(), bestAccuracy.meanErr, bestAccuracy.maxErr)
	fmt.Printf("Fastest:        %s (Time: %.2f ms, Mean Error: %.6e)\n",
		bestSpeed.method.Name(), float64(bestSpeed.duration.Microseconds())/1000.0, bestSpeed.meanErr)
	fmt.Printf("Best Balanced:  %s (Error: %.6e, Time: %.2f ms)\n",
		bestBalanced.method.Name(), bestBalanced.meanErr, float64(bestBalanced.duration.Microseconds())/1000.0)

	// Credit scoring specific recommendation
	fmt.Println("\n💡 For Credit Scoring:")
	fmt.Println("   Typical logit range: [-3, 0]")
	fmt.Println("   Required accuracy: <1% error")

	// Find best for credit scoring range
	creditTestPoints := []float64{-3.0, -2.5, -2.0, -1.5, -1.0, -0.5, 0.0}
	var bestForCredit Result
	minCreditError := math.MaxFloat64
	actualSlots := params.MaxSlots() / 2 // Use actual usable slots

	for _, r := range results {
		var creditError float64
		for _, x := range creditTestPoints {
			values := make([]complex128, actualSlots)
			for i := range values {
				values[i] = complex(x, 0)
			}
			pt := ckks.NewPlaintext(params, params.MaxLevel())
			encoder.Encode(values, pt)
			ct, _ := encryptor.EncryptNew(pt)
			resultCt, _ := r.method.Evaluate(evaluator, ct, params)
			ptResult := decryptor.DecryptNew(resultCt)
			valuesResult := make([]complex128, actualSlots)
			encoder.Decode(ptResult, valuesResult)
			approxValue := real(valuesResult[0])
			trueValue := 1.0 / (1.0 + math.Exp(-x))
			creditError += math.Abs(approxValue - trueValue)
		}
		avgCreditError := creditError / float64(len(creditTestPoints))
		if avgCreditError < minCreditError {
			minCreditError = avgCreditError
			bestForCredit = r
		}
	}

	fmt.Printf("   Recommended:       %s (Error on credit range: %.6e)\n",
		bestForCredit.method.Name(), minCreditError)
	fmt.Printf("   Depth required:    %d levels\n", bestForCredit.method.RequiredDepth())

	// Error analysis by range
	fmt.Println("\n📊 Error Analysis by Input Range:")
	fmt.Println("----------------------------------")
	ranges := []struct {
		name string
		min  float64
		max  float64
		desc string
	}{
		{"Large Negative", -8.0, -4.0, "High default risk"},
		{"Credit Scoring", -3.0, 0.0, "Typical range"},
		{"Small Values", -1.0, 1.0, "Near 0.5"},
		{"Large Positive", 4.0, 8.0, "Low default risk"},
	}

	for _, rng := range ranges {
		fmt.Printf("\n%s [%.1f, %.1f] - %s:\n", rng.name, rng.min, rng.max, rng.desc)
		for _, r := range results {
			var rangeError float64
			count := 0
			for _, x := range testPoints {
				if x >= rng.min && x <= rng.max {
					values := make([]complex128, actualSlots)
					for i := range values {
						values[i] = complex(x, 0)
					}
					pt := ckks.NewPlaintext(params, params.MaxLevel())
					encoder.Encode(values, pt)
					ct, _ := encryptor.EncryptNew(pt)
					resultCt, _ := r.method.Evaluate(evaluator, ct, params)
					ptResult := decryptor.DecryptNew(resultCt)
					valuesResult := make([]complex128, actualSlots)
					encoder.Decode(ptResult, valuesResult)
					approxValue := real(valuesResult[0])
					trueValue := 1.0 / (1.0 + math.Exp(-x))
					rangeError += math.Abs(approxValue - trueValue)
					count++
				}
			}
			if count > 0 {
				avgError := rangeError / float64(count)
				status := "✅"
				if avgError > 0.01 {
					status = "⚠️"
				}
				if avgError > 0.05 {
					status = "❌"
				}
				fmt.Printf("  %s %-18s: %.6e\n", status, r.method.Name(), avgError)
			}
		}
	}
}

func main() {
	// CKKS parameters - using optimized LogN=13
	params, err := ckks.NewParametersFromLiteral(ckks.ParametersLiteral{
		LogN:            13,
		LogQ:            []int{60, 40, 40, 40, 40, 60},
		LogP:            []int{61},
		LogDefaultScale: 40,
	})
	if err != nil {
		panic(err)
	}

	fmt.Println("🧪 CKKS Sigmoid Approximation Benchmark")
	fmt.Println("========================================")
	fmt.Printf("CKKS Parameters: LogN=%d, MaxLevel=%d, MaxSlots=%d\n",
		params.LogN(), params.MaxLevel(), params.MaxSlots())

	benchmarkSigmoidApproximations(params)
}
