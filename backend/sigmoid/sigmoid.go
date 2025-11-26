package sigmoid

import (
	"math"

	"github.com/tuneinsight/lattigo/v6/core/rlwe"
	"github.com/tuneinsight/lattigo/v6/schemes/ckks"
)

// Approximation represents a sigmoid approximation method
type Approximation interface {
	Name() string
	Evaluate(evaluator *ckks.Evaluator, ct *rlwe.Ciphertext, params ckks.Parameters) (*rlwe.Ciphertext, error)
	RequiredDepth() int
}

// ChebyshevApprox implements Chebyshev polynomial approximation
type ChebyshevApprox struct {
	Degree int
	coeffs []float64
}

// NewChebyshevApprox creates a new Chebyshev approximation
// Range: [-8, 8], degrees: 3, 5, 7
func NewChebyshevApprox(degree int) *ChebyshevApprox {
	var coeffs []float64

	switch degree {
	case 3:
		// 3rd degree Chebyshev approximation
		coeffs = []float64{0.5, 0.25, 0.0, -0.03125}
	case 5:
		// 5th degree Chebyshev approximation
		coeffs = []float64{0.5, 0.25, 0.0, -0.03125, 0.0, 0.003906}
	case 7:
		// 7th degree Chebyshev approximation
		coeffs = []float64{0.5, 0.25, 0.0, -0.03125, 0.0, 0.003906, 0.0, -0.000488}
	default:
		degree = 3
		coeffs = []float64{0.5, 0.25, 0.0, -0.03125}
	}

	return &ChebyshevApprox{
		Degree: degree,
		coeffs: coeffs,
	}
}

func (c *ChebyshevApprox) Name() string {
	return "Chebyshev-" + string(rune(c.Degree+'0'))
}

func (c *ChebyshevApprox) RequiredDepth() int {
	return c.Degree
}

func (c *ChebyshevApprox) Evaluate(evaluator *ckks.Evaluator, ct *rlwe.Ciphertext, params ckks.Parameters) (*rlwe.Ciphertext, error) {
	// Compute polynomial using Horner's method
	// p(x) = c0 + c1*x + c2*x^2 + ... + cn*x^n

	encoder := ckks.NewEncoder(params)

	// Start with the highest degree coefficient
	n := len(c.coeffs) - 1
	result := ct.CopyNew()

	// Scale by last coefficient
	if c.coeffs[n] != 0 {
		constPt := ckks.NewPlaintext(params, result.Level())
		values := make([]complex128, params.MaxSlots())
		for i := range values {
			values[i] = complex(c.coeffs[n], 0)
		}
		encoder.Encode(values, constPt)
		evaluator.Mul(result, constPt, result)
		evaluator.Rescale(result, result)
	}

	// Horner's method: iteratively compute result = result*x + c[i]
	for i := n - 1; i >= 0; i-- {
		if i < n-1 {
			// Multiply by x
			evaluator.Mul(result, ct, result)
			evaluator.Rescale(result, result)
		}

		// Add coefficient c[i]
		if c.coeffs[i] != 0 {
			constPt := ckks.NewPlaintext(params, result.Level())
			values := make([]complex128, params.MaxSlots())
			for j := range values {
				values[j] = complex(c.coeffs[i], 0)
			}
			encoder.Encode(values, constPt)
			evaluator.Add(result, constPt, result)
		}
	}

	return result, nil
}

// MinimaxApprox implements minimax polynomial approximation
type MinimaxApprox struct {
	Degree int
	coeffs []float64
}

// NewMinimaxApprox creates a new minimax approximation
func NewMinimaxApprox(degree int) *MinimaxApprox {
	var coeffs []float64

	switch degree {
	case 3:
		// 3rd degree minimax approximation for sigmoid on [-8, 8]
		coeffs = []float64{0.5, 0.2159198, 0.0, -0.0082176}
	case 5:
		// 5th degree minimax approximation
		coeffs = []float64{0.5, 0.2380952, 0.0, -0.0154321, 0.0, 0.0006588}
	case 7:
		// 7th degree minimax approximation
		coeffs = []float64{0.5, 0.2471169, 0.0, -0.0195740, 0.0, 0.0015314, 0.0, -0.0000451}
	default:
		degree = 3
		coeffs = []float64{0.5, 0.2159198, 0.0, -0.0082176}
	}

	return &MinimaxApprox{
		Degree: degree,
		coeffs: coeffs,
	}
}

func (m *MinimaxApprox) Name() string {
	return "Minimax-" + string(rune(m.Degree+'0'))
}

func (m *MinimaxApprox) RequiredDepth() int {
	return m.Degree
}

func (m *MinimaxApprox) Evaluate(evaluator *ckks.Evaluator, ct *rlwe.Ciphertext, params ckks.Parameters) (*rlwe.Ciphertext, error) {
	encoder := ckks.NewEncoder(params)

	// Start with the highest degree coefficient
	n := len(m.coeffs) - 1
	result := ct.CopyNew()

	// Scale by last coefficient
	if m.coeffs[n] != 0 {
		constPt := ckks.NewPlaintext(params, result.Level())
		values := make([]complex128, params.MaxSlots())
		for i := range values {
			values[i] = complex(m.coeffs[n], 0)
		}
		encoder.Encode(values, constPt)
		evaluator.Mul(result, constPt, result)
		evaluator.Rescale(result, result)
	}

	// Horner's method
	for i := n - 1; i >= 0; i-- {
		if i < n-1 {
			evaluator.Mul(result, ct, result)
			evaluator.Rescale(result, result)
		}

		if m.coeffs[i] != 0 {
			constPt := ckks.NewPlaintext(params, result.Level())
			values := make([]complex128, params.MaxSlots())
			for j := range values {
				values[j] = complex(m.coeffs[i], 0)
			}
			encoder.Encode(values, constPt)
			evaluator.Add(result, constPt, result)
		}
	}

	return result, nil
}

// CompositeApprox uses piecewise approximation
type CompositeApprox struct {
	Degree int
}

func NewCompositeApprox(degree int) *CompositeApprox {
	return &CompositeApprox{Degree: degree}
}

func (c *CompositeApprox) Name() string {
	return "Composite-" + string(rune(c.Degree+'0'))
}

func (c *CompositeApprox) RequiredDepth() int {
	return c.Degree + 2
}

func (c *CompositeApprox) Evaluate(evaluator *ckks.Evaluator, ct *rlwe.Ciphertext, params ckks.Parameters) (*rlwe.Ciphertext, error) {
	// Composite approximation: σ(x) ≈ 0.5 + 0.5 * tanh(x/2)
	// tanh(x/2) ≈ x/2 - (x/2)^3/3 + ...

	encoder := ckks.NewEncoder(params)

	// First scale by 0.5 (x/2)
	halfPt := ckks.NewPlaintext(params, ct.Level())
	values := make([]complex128, params.MaxSlots())
	for i := range values {
		values[i] = complex(0.5, 0)
	}
	encoder.Encode(values, halfPt)

	scaledCt, _ := evaluator.MulNew(ct, halfPt)
	evaluator.Rescale(scaledCt, scaledCt)

	// Compute x^3
	x2, _ := evaluator.MulRelinNew(scaledCt, scaledCt)
	evaluator.Rescale(x2, x2)

	x3, _ := evaluator.MulRelinNew(x2, scaledCt)
	evaluator.Rescale(x3, x3)

	// Compute (x/2) - (x/2)^3/3
	thirdPt := ckks.NewPlaintext(params, x3.Level())
	for i := range values {
		values[i] = complex(1.0/3.0, 0)
	}
	encoder.Encode(values, thirdPt)

	x3Scaled, _ := evaluator.MulNew(x3, thirdPt)
	evaluator.Rescale(x3Scaled, x3Scaled)

	tanhApprox, _ := evaluator.SubNew(scaledCt, x3Scaled)

	// Multiply by 0.5
	result, _ := evaluator.MulNew(tanhApprox, halfPt)
	evaluator.Rescale(result, result)

	// Add 0.5
	encoder.Encode(values, halfPt)
	evaluator.Add(result, halfPt, result)

	return result, nil
}

// BenchmarkResult stores benchmark information
type BenchmarkResult struct {
	Method     string
	Accuracy   float64 // Mean absolute error
	Duration   int64   // Microseconds
	MaxError   float64
	TestPoints int
}

// Benchmark tests all approximation methods
func Benchmark(approxMethods []Approximation, params ckks.Parameters) []BenchmarkResult {
	results := make([]BenchmarkResult, 0, len(approxMethods))

	// Test points
	testPoints := []float64{-8, -6, -4, -2, -1, -0.5, 0, 0.5, 1, 2, 4, 6, 8}

	kgen := ckks.NewKeyGenerator(params)
	sk := kgen.GenSecretKeyNew()
	encoder := ckks.NewEncoder(params)
	encryptor := ckks.NewEncryptor(params, sk)
	decryptor := ckks.NewDecryptor(params, sk)
	evaluator := ckks.NewEvaluator(params, nil)

	for _, method := range approxMethods {
		var totalError float64
		var maxError float64

		for _, x := range testPoints {
			// Encrypt
			values := make([]complex128, params.MaxSlots())
			values[0] = complex(x, 0)
			pt := ckks.NewPlaintext(params, params.MaxLevel())
			encoder.Encode(values, pt)
			ct, _ := encryptor.EncryptNew(pt)

			// Evaluate
			result, _ := method.Evaluate(evaluator, ct, params)

			// Decrypt
			ptResult := decryptor.DecryptNew(result)
			valuesResult := make([]complex128, params.MaxSlots())
			encoder.Decode(ptResult, valuesResult)

			approxValue := real(valuesResult[0])
			trueValue := 1.0 / (1.0 + math.Exp(-x))
			err := math.Abs(approxValue - trueValue)

			totalError += err
			if err > maxError {
				maxError = err
			}
		}

		results = append(results, BenchmarkResult{
			Method:     method.Name(),
			Accuracy:   totalError / float64(len(testPoints)),
			MaxError:   maxError,
			TestPoints: len(testPoints),
		})
	}

	return results
}
