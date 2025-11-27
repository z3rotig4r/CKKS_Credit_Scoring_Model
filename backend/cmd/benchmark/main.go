package main

import (
	"fmt"

	"github.com/tuneinsight/lattigo/v6/schemes/ckks"
)

func main() {
	// CKKS 파라미터 초기화 (Production: LogN=13, MaxLevel=5)
	params, err := ckks.NewParametersFromLiteral(ckks.ParametersLiteral{
		LogN:            13,
		LogQ:            []int{60, 40, 40, 40, 40, 60}, // MaxLevel=5
		LogP:            []int{61},
		LogDefaultScale: 40,
	})
	if err != nil {
		panic(fmt.Sprintf("Failed to create CKKS parameters: %v", err))
	}

	fmt.Println("╔════════════════════════════════════════════════════════════╗")
	fmt.Println("║   CKKS Credit Scoring - Production Model Benchmark        ║")
	fmt.Println("╚════════════════════════════════════════════════════════════╝")
	fmt.Printf("\n📋 CKKS Parameters:\n")
	fmt.Printf("   LogN:            %d (Ring degree: %d)\n", params.LogN(), 1<<params.LogN())
	fmt.Printf("   LogQ:            %v\n", []int{60, 40, 40, 60})
	fmt.Printf("   MaxLevel:        %d\n", params.MaxLevel())
	fmt.Printf("   MaxSlots:        %d\n", params.MaxSlots())
	fmt.Printf("   Default Scale:   2^%d\n", 40)
	fmt.Println()

	// Run production model benchmark
	benchmarkModel(params)

	// Run detailed homomorphic operations benchmark
	benchmarkHomomorphicOps(params)

	// Run sigmoid approximation comparison benchmark
	benchmarkSigmoidApproximations(params)

	fmt.Println("\n✅ Benchmark Complete!")
}
