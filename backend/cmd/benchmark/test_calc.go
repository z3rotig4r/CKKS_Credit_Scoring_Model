package main
import (
"fmt"
"math"
)
func main() {
	logit := -1.4137 + (-0.2502*25) + (0.0137*0.15) + (0.0124*0.20) + (-0.0427*5000) + (0.0063*50000)
	fmt.Printf("Logit: %.6f\n", logit)
	score := 1.0 / (1.0 + math.Exp(-logit))
	fmt.Printf("Score: %.6f\n", score)
}
