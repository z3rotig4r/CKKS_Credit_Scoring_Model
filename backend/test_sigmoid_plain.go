package main

import (
	"fmt"
	"math"
)

func sigmoid(x float64) float64 {
	return 1.0 / (1.0 + math.Exp(-x))
}

func chebyshev3(x float64) float64 {
	// Horner: 0.5 + x(0.25 + x(0 + x(-0.03125)))
	return 0.5 + 0.25*x - 0.03125*x*x*x
}

func chebyshev5(x float64) float64 {
	return 0.5 + 0.25*x - 0.03125*x*x*x + 0.003906*x*x*x*x*x
}

func minimax3(x float64) float64 {
	return 0.5 + 0.2159198*x - 0.0082176*x*x*x
}

func main() {
	fmt.Println("Plain Sigmoid Approximation Test")
	fmt.Println("=================================\n")

	testPoints := []float64{-3, -2, -1, 0, 1, 2, 3}

	fmt.Printf("%-8s | %-12s | %-12s | %-12s | %-12s\n", "x", "True", "Cheby-3", "Cheby-5", "Minimax-3")
	fmt.Println(string(make([]byte, 70)))

	for _, x := range testPoints {
		true_val := sigmoid(x)
		c3 := chebyshev3(x)
		c5 := chebyshev5(x)
		m3 := minimax3(x)

		fmt.Printf("%-8.1f | %.8f | %.8f | %.8f | %.8f\n", x, true_val, c3, c5, m3)
	}

	fmt.Println("\nErrors:")
	fmt.Printf("%-8s | %-12s | %-12s | %-12s\n", "x", "Cheby-3", "Cheby-5", "Minimax-3")
	fmt.Println(string(make([]byte, 50)))

	for _, x := range testPoints {
		true_val := sigmoid(x)
		c3_err := math.Abs(chebyshev3(x) - true_val)
		c5_err := math.Abs(chebyshev5(x) - true_val)
		m3_err := math.Abs(minimax3(x) - true_val)

		fmt.Printf("%-8.1f | %.6e | %.6e | %.6e\n", x, c3_err, c5_err, m3_err)
	}
}
