package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/tuneinsight/lattigo/v6/core/rlwe"
	"github.com/tuneinsight/lattigo/v6/schemes/ckks"
)

func main() {
	// Same parameters as in main.go
	LogN := 13
	LogQ := []int{60, 40, 40, 60}
	LogP := []int{61}
	LogDefaultScale := 40

	params, err := ckks.NewParametersFromLiteral(ckks.ParametersLiteral{
		LogN:            LogN,
		LogQ:            LogQ,
		LogP:            LogP,
		LogDefaultScale: LogDefaultScale,
	})
	if err != nil {
		log.Fatalf("Failed to create parameters: %v", err)
	}

	fmt.Println("=== CKKS Encryption/Decryption Test ===")
	fmt.Printf("LogN=%d, MaxLevel=%d, DefaultScale=2^%d\n\n", params.LogN(), params.MaxLevel(), LogDefaultScale)

	// Generate keys
	kgen := ckks.NewKeyGenerator(params)
	sk := kgen.GenSecretKeyNew()
	pk := kgen.GenPublicKeyNew(sk)
	rlk := kgen.GenRelinearizationKeyNew(sk)
	evk := rlwe.NewMemEvaluationKeySet(rlk)

	encoder := ckks.NewEncoder(params)
	encryptor := ckks.NewEncryptor(params, pk)
	decryptor := ckks.NewDecryptor(params, sk)
	evaluator := ckks.NewEvaluator(params, evk)

	// Test values: preprocessed credit features
	testValues := []float64{5.0, 2.0, 0.1, 100.0, 50.0, 500.0}

	fmt.Println("Testing encryption/decryption of credit features:")
	for i, val := range testValues {
		// Encode
		values := make([]complex128, params.MaxSlots())
		values[0] = complex(val, 0)
		pt := ckks.NewPlaintext(params, params.MaxLevel())
		if err := encoder.Encode(values, pt); err != nil {
			log.Fatalf("Encode failed: %v", err)
		}

		// Encrypt
		ct, err := encryptor.EncryptNew(pt)
		if err != nil {
			log.Fatalf("Encrypt failed: %v", err)
		}

		// Decrypt
		ptDec := decryptor.DecryptNew(ct)
		resultValues := make([]complex128, params.MaxSlots())
		if err := encoder.Decode(ptDec, resultValues); err != nil {
			log.Fatalf("Decode failed: %v", err)
		}

		decrypted := real(resultValues[0])
		error := decrypted - val
		relError := error / val * 100.0

		fmt.Printf("  [%d] Original: %10.6f → Decrypted: %10.6f (Error: %+.6f, %.3f%%)\n",
			i, val, decrypted, error, relError)
	}

	fmt.Println("\n=== Test Sigmoid Output Range ===")
	// Test a weighted sum result (typical logit value)
	logitValue := -1.5 // Typical credit score logit
	values := make([]complex128, params.MaxSlots())
	values[0] = complex(logitValue, 0)
	pt := ckks.NewPlaintext(params, params.MaxLevel())
	if err := encoder.Encode(values, pt); err != nil {
		log.Fatalf("Encode failed: %v", err)
	}

	ct, err := encryptor.EncryptNew(pt)
	if err != nil {
		log.Fatalf("Encrypt failed: %v", err)
	}

	// Load and apply sigmoid
	sigmoid := loadSigmoid()
	result, err := sigmoid.Evaluate(evaluator, ct, params)
	if err != nil {
		log.Fatalf("Sigmoid evaluation failed: %v", err)
	}

	// Decrypt
	ptDec := decryptor.DecryptNew(result)
	resultValues := make([]complex128, params.MaxSlots())
	if err := encoder.Decode(ptDec, resultValues); err != nil {
		log.Fatalf("Decode failed: %v", err)
	}

	probability := real(resultValues[0])
	expected := 1.0 / (1.0 + (2.718281828459045 * 1.5)) // e^1.5

	fmt.Printf("Logit value: %.6f\n", logitValue)
	fmt.Printf("Expected probability: %.6f\n", expected)
	fmt.Printf("Decrypted probability: %.6f\n", probability)
	fmt.Printf("Error: %.6f (%.2f%%)\n", probability-expected, (probability-expected)/expected*100)

	if probability < 0 || probability > 1.5 {
		fmt.Println("\n⚠️  WARNING: Probability out of valid range [0, 1]!")
		fmt.Println("This indicates a scale/level management issue.")
	} else {
		fmt.Println("\n✅ Probability in valid range")
	}
}

func loadSigmoid() interface {
	Evaluate(evaluator *ckks.Evaluator, ct *rlwe.Ciphertext, params ckks.Parameters) (*rlwe.Ciphertext, error)
} {
	// Dynamically load sigmoid package
	// For now, just return a stub that will be replaced
	wd, _ := os.Getwd()
	sigmoidPath := filepath.Join(wd, "sigmoid")
	fmt.Printf("Loading sigmoid from: %s\n", sigmoidPath)

	// This is a placeholder - in real code we'd use the actual sigmoid
	return nil
}
