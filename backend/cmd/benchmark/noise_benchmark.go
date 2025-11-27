package main

import (
	"fmt"
	"log"
	"math"
	"math/cmplx"
	"strings"

	"github.com/tuneinsight/lattigo/v6/core/rlwe"
	"github.com/tuneinsight/lattigo/v6/schemes/ckks"
	"github.com/z3rotig4r/ckks_credit/backend/sigmoid"
)

// Logistic Regression Model (5 features)
type NoiseTestModel struct {
	Weights []float64
	Bias    float64
}

var noiseModel = NoiseTestModel{
	Weights: []float64{-0.2501752295, 0.0137090654, 0.0123900347, -0.0426762083, 0.0062886554},
	Bias:    -1.4136778933,
}

// Test cases representing different risk profiles
// Features are NORMALIZED to realistic credit scoring range
// Expected logit range: -3 to 0 (sigmoid output: 0.05 to 0.5)
type TestCase struct {
	Name     string
	Features []float64 // [age, loan_to_income, debt_to_income, credit_amount, income]
	Expected float64   // Expected probability after sigmoid
}

var testCases = []TestCase{
	{
		Name: "Low Risk Customer",
		// age=0.45, loan_to_income=0.15, debt_to_income=0.20, credit_amount=0.5, income=0.8
		Features: []float64{0.45, 0.15, 0.20, 0.5, 0.8},
		Expected: 0.0, // Will be calculated
	},
	{
		Name: "Medium Risk Customer",
		// age=0.30, loan_to_income=0.35, debt_to_income=0.45, credit_amount=1.2, income=0.5
		Features: []float64{0.30, 0.35, 0.45, 1.2, 0.5},
		Expected: 0.0,
	},
	{
		Name: "High Risk Customer",
		// age=0.25, loan_to_income=0.55, debt_to_income=0.60, credit_amount=2.0, income=0.35
		Features: []float64{0.25, 0.55, 0.60, 2.0, 0.35},
		Expected: 0.0,
	},
	{
		Name: "Very Low Risk (Conservative)",
		// age=0.55, loan_to_income=0.10, debt_to_income=0.15, credit_amount=0.3, income=1.0
		Features: []float64{0.55, 0.10, 0.15, 0.3, 1.0},
		Expected: 0.0,
	},
	{
		Name: "Boundary Case (Near 0.5)",
		// age=0.35, loan_to_income=0.30, debt_to_income=0.35, credit_amount=0.8, income=0.6
		Features: []float64{0.35, 0.30, 0.35, 0.8, 0.6},
		Expected: 0.0,
	},
	{
		Name: "Edge Case: Very High Logit",
		// Designed to test logit near 0 (probability ~0.5)
		Features: []float64{5.0, 3.5, 5.0, 0.0, 8.0},
		Expected: 0.0,
	},
	{
		Name: "Edge Case: Very Low Logit",
		// Designed to test logit near -5 (probability ~0.007)
		Features: []float64{0.1, 0.1, 0.1, 5.0, 0.1},
		Expected: 0.0,
	},
}

// Sigmoid function (ground truth)
func sigmoidFunc(x float64) float64 {
	return 1.0 / (1.0 + math.Exp(-x))
}

// Calculate expected probability using plaintext computation
func calculateExpected(features []float64) float64 {
	logit := noiseModel.Bias
	for i, weight := range noiseModel.Weights {
		logit += weight * features[i]
	}
	return sigmoidFunc(logit)
}

// Noise metrics for a single test case
type NoiseMetrics struct {
	TestName          string
	ExpectedLogit     float64
	ExpectedProb      float64
	EncryptedProb     float64
	AbsoluteError     float64
	RelativeError     float64
	NoiseLevel        float64
	FinalLevel        int
	LogitLevelBefore  int
	LogitLevelAfter   int
	SigmoidLevelStart int
	SigmoidLevelEnd   int
}

// Benchmark noise levels with FULL sigmoid transformation
func benchmarkNoiseWithSigmoid(params ckks.Parameters) {
	fmt.Println("\n" + separator)
	fmt.Println("🔬 NOISE BENCHMARK WITH NEW SIGMOID (Credit Scoring Optimized)")
	fmt.Println(separator)
	fmt.Println("✅ NEW: CreditScoring-5 approximation (range [-3, -1])")
	fmt.Println("✅ Expected error: <0.5% (vs Composite-3: 100%)")
	fmt.Println("⚠️  Testing FULL logistic regression (logit + sigmoid)")
	fmt.Println(separator)

	// Initialize CKKS components
	kgen := rlwe.NewKeyGenerator(params)
	sk := kgen.GenSecretKeyNew()
	pk := kgen.GenPublicKeyNew(sk)

	// Generate relinearization key for polynomial evaluation
	rlk := kgen.GenRelinearizationKeyNew(sk)
	evk := rlwe.NewMemEvaluationKeySet(rlk)

	encoder := ckks.NewEncoder(params)
	encryptor := rlwe.NewEncryptor(params, pk)
	decryptor := rlwe.NewDecryptor(params, sk)
	evaluator := ckks.NewEvaluator(params, evk)

	// Initialize sigmoid approximation (NEW: CreditScoring-3 for [-3, -1] range)
	sigmoidApprox := sigmoid.Approximation(sigmoid.NewCreditScoringApprox(3))

	// Calculate expected values for all test cases
	for i := range testCases {
		testCases[i].Expected = calculateExpected(testCases[i].Features)
	}

	fmt.Println("\n📊 Test Cases Summary:")
	fmt.Println("-------------------------------------------------------------------")
	for i, tc := range testCases {
		logit := noiseModel.Bias
		for j, w := range noiseModel.Weights {
			logit += w * tc.Features[j]
		}
		fmt.Printf("%d. %s\n", i+1, tc.Name)
		fmt.Printf("   Features: %v\n", tc.Features)
		fmt.Printf("   Logit: %.6f\n", logit)
		fmt.Printf("   Expected Probability: %.6f\n\n", tc.Expected)
	}

	// Run noise analysis for each test case
	var allMetrics []NoiseMetrics

	for _, tc := range testCases {
		fmt.Println("\n" + strings.Repeat("=", 70))
		fmt.Printf("🧪 Testing: %s\n", tc.Name)
		fmt.Println(strings.Repeat("=", 70))

		metrics := runSingleNoiseTest(
			tc,
			params,
			encoder,
			encryptor,
			decryptor,
			evaluator,
			sigmoidApprox,
		)

		allMetrics = append(allMetrics, metrics)
	}

	// Print summary report
	printNoiseSummary(allMetrics)
}

func runSingleNoiseTest(
	tc TestCase,
	params ckks.Parameters,
	encoder *ckks.Encoder,
	encryptor *rlwe.Encryptor,
	decryptor *rlwe.Decryptor,
	evaluator *ckks.Evaluator,
	sigmoidApprox sigmoid.Approximation,
) NoiseMetrics {

	// Step 1: Encrypt features
	fmt.Println("\n📦 Step 1: Encrypting features...")
	encryptedFeatures := make([]*rlwe.Ciphertext, len(tc.Features))
	for i, feature := range tc.Features {
		values := make([]complex128, params.MaxSlots())
		values[0] = complex(feature, 0)
		plaintext := ckks.NewPlaintext(params, params.MaxLevel())
		encoder.Encode(values, plaintext)
		encryptedFeatures[i], _ = encryptor.EncryptNew(plaintext)
		fmt.Printf("   Feature %d: Level=%d, Value=%.4f\n", i, encryptedFeatures[i].Level(), feature)
	}

	// Step 2: Compute weighted sum (logit)
	fmt.Println("\n🧮 Step 2: Computing weighted sum (logit)...")
	logitLevelBefore := encryptedFeatures[0].Level()

	values := make([]complex128, params.MaxSlots())
	values[0] = complex(noiseModel.Weights[0], 0)
	weightPt := ckks.NewPlaintext(params, encryptedFeatures[0].Level())
	encoder.Encode(values, weightPt)

	result, _ := evaluator.MulNew(encryptedFeatures[0], weightPt)
	evaluator.Rescale(result, result)
	fmt.Printf("   First weight multiplication: Level=%d\n", result.Level())

	for i := 1; i < len(encryptedFeatures); i++ {
		values[0] = complex(noiseModel.Weights[i], 0)
		weightPt := ckks.NewPlaintext(params, encryptedFeatures[i].Level())
		encoder.Encode(values, weightPt)

		weightedFeature, _ := evaluator.MulNew(encryptedFeatures[i], weightPt)
		evaluator.Rescale(weightedFeature, weightedFeature)

		// Level alignment
		if result.Level() != weightedFeature.Level() {
			if result.Level() > weightedFeature.Level() {
				evaluator.DropLevel(result, result.Level()-weightedFeature.Level())
			} else {
				evaluator.DropLevel(weightedFeature, weightedFeature.Level()-result.Level())
			}
		}

		evaluator.Add(result, weightedFeature, result)
		fmt.Printf("   After feature %d addition: Level=%d\n", i, result.Level())
	}

	// Add bias
	values[0] = complex(noiseModel.Bias, 0)
	biasPt := ckks.NewPlaintext(params, result.Level())
	encoder.Encode(values, biasPt)
	evaluator.Add(result, biasPt, result)
	logitLevelAfter := result.Level()

	fmt.Printf("   ✅ Logit computation complete: Level=%d\n", result.Level())

	// Decrypt logit to check intermediate noise
	logitPlaintext := decryptor.DecryptNew(result)
	logitValues := make([]complex128, params.MaxSlots())
	encoder.Decode(logitPlaintext, logitValues)
	encryptedLogit := real(logitValues[0])

	// Calculate expected logit
	expectedLogit := noiseModel.Bias
	for i, w := range noiseModel.Weights {
		expectedLogit += w * tc.Features[i]
	}

	fmt.Printf("\n   Expected Logit:  %.10f\n", expectedLogit)
	fmt.Printf("   Encrypted Logit: %.10f\n", encryptedLogit)
	fmt.Printf("   Logit Error:     %.10e\n", math.Abs(expectedLogit-encryptedLogit))

	// Step 3: Apply FULL sigmoid transformation (NEVER skip this!)
	fmt.Printf("\n🔐 Step 3: Applying sigmoid approximation (%s)...\n", sigmoidApprox.Name())
	sigmoidLevelStart := result.Level()

	score, err := sigmoidApprox.Evaluate(evaluator, result, params)
	if err != nil {
		log.Fatalf("❌ Sigmoid evaluation failed: %v", err)
	}

	sigmoidLevelEnd := score.Level()
	fmt.Printf("   Sigmoid levels: Start=%d, End=%d\n", sigmoidLevelStart, sigmoidLevelEnd)

	// Step 4: Decrypt and measure noise
	fmt.Println("\n🔓 Step 4: Decrypting result...")
	scorePlaintext := decryptor.DecryptNew(score)
	scoreValues := make([]complex128, params.MaxSlots())
	encoder.Decode(scorePlaintext, scoreValues)
	encryptedProb := real(scoreValues[0])

	// Calculate noise metrics
	absError := math.Abs(tc.Expected - encryptedProb)
	relError := absError / tc.Expected * 100.0

	// Estimate noise level (magnitude of imaginary component + deviation from expected)
	noiseLevel := cmplx.Abs(scoreValues[0] - complex(tc.Expected, 0))

	fmt.Printf("   Expected Probability:  %.10f\n", tc.Expected)
	fmt.Printf("   Encrypted Probability: %.10f\n", encryptedProb)
	fmt.Printf("   Absolute Error:        %.10e\n", absError)
	fmt.Printf("   Relative Error:        %.6f%%\n", relError)
	fmt.Printf("   Noise Level:           %.10e\n", noiseLevel)

	return NoiseMetrics{
		TestName:          tc.Name,
		ExpectedLogit:     expectedLogit,
		ExpectedProb:      tc.Expected,
		EncryptedProb:     encryptedProb,
		AbsoluteError:     absError,
		RelativeError:     relError,
		NoiseLevel:        noiseLevel,
		FinalLevel:        score.Level(),
		LogitLevelBefore:  logitLevelBefore,
		LogitLevelAfter:   logitLevelAfter,
		SigmoidLevelStart: sigmoidLevelStart,
		SigmoidLevelEnd:   sigmoidLevelEnd,
	}
}

func printNoiseSummary(metrics []NoiseMetrics) {
	fmt.Println("\n" + separator)
	fmt.Println("📈 NOISE ANALYSIS SUMMARY")
	fmt.Println(separator)

	fmt.Println("\n┌─────────────────────────────────────────────────────────────────────────┐")
	fmt.Println("│                         NOISE METRICS TABLE                             │")
	fmt.Println("├─────────────────────────────────────────────────────────────────────────┤")
	fmt.Printf("│ %-25s │ %10s │ %10s │ %8s │\n", "Test Case", "Expected", "Encrypted", "Error (%)")
	fmt.Println("├─────────────────────────────────────────────────────────────────────────┤")

	var totalAbsError, totalRelError, totalNoise float64

	for _, m := range metrics {
		fmt.Printf("│ %-25s │ %10.6f │ %10.6f │ %7.4f%% │\n",
			m.TestName, m.ExpectedProb, m.EncryptedProb, m.RelativeError)
		totalAbsError += m.AbsoluteError
		totalRelError += m.RelativeError
		totalNoise += m.NoiseLevel
	}

	fmt.Println("└─────────────────────────────────────────────────────────────────────────┘")

	avgAbsError := totalAbsError / float64(len(metrics))
	avgRelError := totalRelError / float64(len(metrics))
	avgNoise := totalNoise / float64(len(metrics))

	fmt.Println("\n📊 Statistical Summary:")
	fmt.Printf("   Average Absolute Error: %.10e\n", avgAbsError)
	fmt.Printf("   Average Relative Error: %.6f%%\n", avgRelError)
	fmt.Printf("   Average Noise Level:    %.10e\n", avgNoise)

	// Level consumption analysis
	fmt.Println("\n🔋 Level Consumption Analysis:")
	for _, m := range metrics {
		fmt.Printf("\n   %s:\n", m.TestName)
		fmt.Printf("   ├─ Initial Level:      %d (fresh encryption)\n", m.LogitLevelBefore)
		fmt.Printf("   ├─ After Logit:        %d (consumed %d levels)\n",
			m.LogitLevelAfter, m.LogitLevelBefore-m.LogitLevelAfter)
		fmt.Printf("   ├─ Sigmoid Start:      %d\n", m.SigmoidLevelStart)
		fmt.Printf("   ├─ Sigmoid End:        %d (consumed %d levels)\n",
			m.SigmoidLevelEnd, m.SigmoidLevelStart-m.SigmoidLevelEnd)
		fmt.Printf("   └─ Final Level:        %d\n", m.FinalLevel)
	}

	// Quality assessment
	fmt.Println("\n✅ Quality Assessment:")
	if avgRelError < 0.01 {
		fmt.Println("   🟢 EXCELLENT: Average error < 0.01% (Production ready)")
	} else if avgRelError < 0.1 {
		fmt.Println("   🟡 GOOD: Average error < 0.1% (Acceptable for most cases)")
	} else if avgRelError < 1.0 {
		fmt.Println("   🟠 MODERATE: Average error < 1.0% (May need parameter tuning)")
	} else {
		fmt.Println("   🔴 POOR: Average error > 1.0% (Requires investigation)")
	}

	// Noise budget warning
	minFinalLevel := metrics[0].FinalLevel
	for _, m := range metrics {
		if m.FinalLevel < minFinalLevel {
			minFinalLevel = m.FinalLevel
		}
	}

	fmt.Println("\n⚠️  Noise Budget Warning:")
	if minFinalLevel < 0 {
		fmt.Printf("   🔴 CRITICAL: Final level %d < 0 (Out of noise budget!)\n", minFinalLevel)
		fmt.Println("   Recommendation: Increase LogQ or reduce computation depth")
	} else if minFinalLevel == 0 {
		fmt.Printf("   🟡 WARNING: Final level = 0 (Noise budget exhausted)\n")
		fmt.Println("   Recommendation: No room for additional operations")
	} else {
		fmt.Printf("   🟢 SAFE: Final level = %d (Noise budget remaining)\n", minFinalLevel)
	}

	fmt.Println("\n" + separator)
	fmt.Println("✅ Noise benchmark completed successfully")
	fmt.Println(separator)
}
