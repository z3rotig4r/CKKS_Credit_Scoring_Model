package main

import (
	"fmt"
	"math"
)

// Sigmoid function
func sigmoid(x float64) float64 {
	return 1.0 / (1.0 + math.Exp(-x))
}

// Chebyshev node (degree n, node k)
func chebyshevNode(k, n int, a, b float64) float64 {
	// Map Chebyshev node from [-1, 1] to [a, b]
	theta := math.Pi * (float64(k) + 0.5) / float64(n+1)
	x := -math.Cos(theta) // Node in [-1, 1]
	// Transform to [a, b]
	return 0.5*(b-a)*x + 0.5*(a+b)
}

// Compute Chebyshev coefficients for sigmoid in [a, b]
func chebyshevCoeffs(degree int, a, b float64) []float64 {
	n := degree
	coeffs := make([]float64, n+1)

	// Evaluate sigmoid at Chebyshev nodes
	fk := make([]float64, n+1)
	for k := 0; k <= n; k++ {
		xk := chebyshevNode(k, n, a, b)
		fk[k] = sigmoid(xk)
	}

	// Compute Chebyshev coefficients using DCT
	for j := 0; j <= n; j++ {
		sum := 0.0
		for k := 0; k <= n; k++ {
			theta := math.Pi * (float64(k) + 0.5) / float64(n+1)
			sum += fk[k] * math.Cos(float64(j)*theta)
		}
		coeffs[j] = 2.0 * sum / float64(n+1)
	}
	coeffs[0] /= 2.0

	return coeffs
}

// Evaluate Chebyshev polynomial
func evalChebyshev(coeffs []float64, x, a, b float64) float64 {
	// Transform x from [a, b] to [-1, 1]
	y := (2.0*x - a - b) / (b - a)

	// Evaluate using Clenshaw's algorithm
	n := len(coeffs) - 1
	if n == 0 {
		return coeffs[0]
	}

	bk2 := 0.0
	bk1 := 0.0

	for k := n; k >= 1; k-- {
		bk := coeffs[k] + 2.0*y*bk1 - bk2
		bk2 = bk1
		bk1 = bk
	}

	return coeffs[0] + y*bk1 - bk2
}

func main() {
	fmt.Println("Chebyshev Approximation for Sigmoid in Credit Scoring Range")
	fmt.Println("===========================================================\n")

	// Credit scoring logit range
	a := -3.0
	b := -1.0

	degrees := []int{3, 5, 7}

	for _, degree := range degrees {
		fmt.Printf("Degree %d Approximation:\n", degree)
		fmt.Println("------------------------")

		coeffs := chebyshevCoeffs(degree, a, b)

		fmt.Println("Chebyshev Coefficients (for range [-3, -1]):")
		for i, c := range coeffs {
			fmt.Printf("  c[%d] = %.10f\n", i, c)
		}

		// Test points
		fmt.Println("\nTest Points:")
		testPoints := []float64{-3.0, -2.5, -2.0, -1.5, -1.0}

		var maxError float64
		var avgError float64

		for _, x := range testPoints {
			expected := sigmoid(x)
			approx := evalChebyshev(coeffs, x, a, b)
			err := math.Abs(expected - approx)
			relErr := err / expected * 100.0

			fmt.Printf("  x=%.1f: expected=%.6f, approx=%.6f, error=%.6f (%.2f%%)\n",
				x, expected, approx, err, relErr)

			if err > maxError {
				maxError = err
			}
			avgError += err
		}
		avgError /= float64(len(testPoints))

		fmt.Printf("\nMax Error: %.8f (%.4f%%)\n", maxError, maxError/0.15*100)
		fmt.Printf("Avg Error: %.8f (%.4f%%)\n\n", avgError, avgError/0.15*100)
	}
}
