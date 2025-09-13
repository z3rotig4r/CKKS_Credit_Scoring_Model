package app

import (
	"fmt"
	"github.com/tuneinsight/lattigo/v6/core/rlwe"
	"github.com/tuneinsight/lattigo/v6/schemes/ckks"
)

type InferenceEngine struct {
	params    ckks.Parameters
	encoder   *ckks.Encoder
	evaluator *ckks.Evaluator
	scorer    *CreditScorer
}

func NewInferenceEngine(params ckks.Parameters, scorer *CreditScorer) (*InferenceEngine, error) {
	encoder := ckks.NewEncoder(params)
	evaluator := ckks.NewEvaluator(params, nil)

	return &InferenceEngine{
		params:    params,
		encoder:   encoder,
		evaluator: evaluator,
		scorer:    scorer,
	}, nil
}

func (ie *InferenceEngine) InferCreditScore(encryptedFeatures []*rlwe.Ciphertext) (*rlwe.Ciphertext, error) {
	if len(encryptedFeatures) != len(ie.scorer.weights) {
		return nil, fmt.Errorf("feature 개수 불일치: expected %d, got %d", len(ie.scorer.weights), len(encryptedFeatures))
	}

	biasPlaintext := ckks.NewPlaintext(ie.params, ie.params.MaxLevel())
	if err := ie.encoder.Encode([]float64{ie.scorer.bias}, biasPlaintext); err != nil {
		return nil, fmt.Errorf("Bias Encoding Failed!: %v", err)
	}

	result := encryptedFeatures[0].CopyNew()
	weightsPlaintext := ckks.NewPlaintext(ie.params, ie.params.MaxLevel())
	if err := ie.encoder.Encode(result, weightsPlaintext); err != nil {
		return nil, fmt.Errorf("Weights Encoding Failed!: %v", err)
	}

	if err := ie.evaluator.Mul(result, weightsPlaintext, result); err != nil {
		return nil, fmt.Errorf("Mul (feature * weight) Failed!: %v", err)
	}

	for i := 1; i < len(encryptedFeatures); i++ {
		if err := ie.encoder.Encode([]float64{ie.scorer.weights[i]}, weightsPlaintext); err != nil {
			return nil, fmt.Errorf("Weights %d Encoding Failed!: %v", i, err)
		}

		temp := encryptedFeatures[i].CopyNew()
		if err := ie.evaluator.Mul(temp, weightsPlaintext, temp); err != nil {
			return nil, fmt.Errorf("Mul %d (feature * weight) Failed!: %v", i, err)
		}

		if err := ie.evaluator.Add(result, temp, result); err != nil {
			return nil, fmt.Errorf("Add %d (feature * weight) Failed!: %v", i, err)
		}
	}

	if err := ie.evaluator.Mul(result, biasPlaintext, result); err != nil {
		return nil, fmt.Errorf("Bias addition Failed!: %v", err)
	}

	return result, nil
}
