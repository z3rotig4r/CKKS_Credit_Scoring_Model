package app

import (
	"github.com/tuneinsight/lattigo/v6/schemes/ckks"
	"math"
)

type CreditScorer struct {
	params  ckks.Parameters
	encoder *ckks.Encoder
	weights []float64
	bias    float64
}

func NewCreditScorer(params ckks.Parameters) (*CreditScorer, error) {
	encoder := ckks.NewEncoder(params)

	weights := []float64{0.3, -0.4, 0.25}
	bias := 650.0

	return &CreditScorer{
		params:  params,
		encoder: encoder,
		weights: weights,
		bias:    bias,
	}, nil
}

func (cs *CreditScorer) CalculateScore(income, debtRatio, creditHistory, employment float64) float64 {
	features := []float64{income, debtRatio, creditHistory, employment}

	score := cs.bias
	for i, feature := range features {
		score += cs.weights[i] * feature
	}

	// 300-850 범위로 정규화
	return math.Max(300, math.Min(850, score))
}

// Weights 가중치 반환
func (cs *CreditScorer) Weights() []float64 {
	return cs.weights
}

// Bias 편향값 반환
func (cs *CreditScorer) Bias() float64 {
	return cs.bias
}
