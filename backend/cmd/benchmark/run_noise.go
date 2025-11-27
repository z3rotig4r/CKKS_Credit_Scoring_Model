package main

import (
	"fmt"
	"log"

	"github.com/tuneinsight/lattigo/v6/schemes/ckks"
)

const separator = "========================================================================"

func main() {
	fmt.Println(separator)
	fmt.Println("🔬 CKKS NOISE ANALYSIS BENCHMARK")
	fmt.Println("   Testing: Logistic Regression with FULL Sigmoid Transformation")
	fmt.Println("   Model: 5 features (age, loan_to_income, debt_to_income, credit_amount, income)")
	fmt.Println(separator)

	// CKKS Parameters (same as production: LogN=13, MaxLevel=5)
	params, err := ckks.NewParametersFromLiteral(ckks.ParametersLiteral{
		LogN:            13,
		LogQ:            []int{60, 40, 40, 40, 40, 60}, // MaxLevel=5
		LogP:            []int{61},
		LogDefaultScale: 40,
	})

	if err != nil {
		log.Fatalf("Failed to create CKKS parameters: %v", err)
	}

	fmt.Printf("\n⚙️  CKKS Configuration:\n")
	fmt.Printf("   LogN:            %d (Ring dimension: %d)\n", params.LogN(), params.N())
	fmt.Printf("   LogQ:            %v\n", params.LogQ())
	fmt.Printf("   LogP:            %v\n", params.LogP())
	fmt.Printf("   LogDefaultScale: %d (Scale: 2^%d)\n", params.LogDefaultScale(), params.LogDefaultScale())
	fmt.Printf("   MaxLevel:        %d\n", params.MaxLevel())
	fmt.Printf("   MaxSlots:        %d\n", params.MaxSlots())

	// Run noise benchmark with FULL sigmoid
	benchmarkNoiseWithSigmoid(params)

	fmt.Println("\n" + separator)
	fmt.Println("✅ All benchmarks completed successfully")
	fmt.Println(separator)
}
