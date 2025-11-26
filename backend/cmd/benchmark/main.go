package main

import (
	"fmt"
	"time"

	"github.com/tuneinsight/lattigo/v6/schemes/ckks"
	"github.com/z3rotig4r/ckks_credit/backend/sigmoid"
)

func main() {
	// CKKS 파라미터 초기화
	params, err := ckks.NewParametersFromLiteral(ckks.ParametersLiteral{
		LogN:            14,
		LogQ:            []int{60, 40, 40, 60},
		LogP:            []int{61},
		LogDefaultScale: 40,
	})
	if err != nil {
		panic(fmt.Sprintf("Failed to create CKKS parameters: %v", err))
	}

	fmt.Println("🧪 Benchmarking Sigmoid Approximation Methods")
	fmt.Println("=============================================")
	fmt.Printf("Parameters: LogN=%d, MaxLevel=%d, MaxSlots=%d\n\n",
		params.LogN(), params.MaxLevel(), params.MaxSlots())

	// 테스트할 근사 방법들
	methods := []sigmoid.Approximation{
		sigmoid.NewChebyshevApprox(3),
		sigmoid.NewChebyshevApprox(5),
		sigmoid.NewChebyshevApprox(7),
		sigmoid.NewMinimaxApprox(3),
		sigmoid.NewMinimaxApprox(5),
		sigmoid.NewMinimaxApprox(7),
		sigmoid.NewCompositeApprox(3),
	}

	startTime := time.Now()
	results := sigmoid.Benchmark(methods, params)
	totalTime := time.Since(startTime)

	// 결과 출력
	fmt.Println("Results:")
	fmt.Println("--------")
	fmt.Printf("%-20s | %-15s | %-15s | %-10s\n", "Method", "Mean Error", "Max Error", "Depth")
	fmt.Println(string(make([]byte, 80)))

	for i, result := range results {
		fmt.Printf("%-20s | %.10f | %.10f | %d\n",
			result.Method,
			result.Accuracy,
			result.MaxError,
			methods[i].RequiredDepth())
	}

	fmt.Printf("\nTotal benchmark time: %v\n", totalTime)
	fmt.Println("\n📊 Recommendation:")
	fmt.Println("   - For best accuracy: Minimax-7 or Chebyshev-7")
	fmt.Println("   - For balance: Chebyshev-5 or Minimax-5")
	fmt.Println("   - For speed: Chebyshev-3 or Minimax-3")
}
