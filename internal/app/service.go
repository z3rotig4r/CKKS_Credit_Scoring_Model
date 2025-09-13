package app

import (
	"fmt"
	"github.com/tuneinsight/lattigo/v6/core/rlwe"
	"github.com/tuneinsight/lattigo/v6/schemes/ckks"
)

type CreditScoringService struct {
	params    ckks.Parameters
	scorer    *CreditScorer
	inference *InferenceEngine
}

func NewCreditScoringService() (*CreditScoringService, error) {
	paramsLit := ckks.ParametersLiternal{
		LogN:            12,
		LogQ:            []int{38, 32},
		LogP:            []int{39},
		LogDefaultScale: 32,
	}

	params, err := ckks.NewParametersFromLiteral(paramsLit)
	if err != nil {
		return nil, fmt.Errorf("CKKS ParaGen Failed!: %v", err)
	}

	scorer, err := NewCreditScorer(params)
	if err != nil {
		return nil, fmt.Errorf("CreditScorer Gen Failed!: %v", err)
	}

	inference, err := NewIn

	return &CreditScoringService{
		params:    params,
		scorer:    scorer,
		inference: inference,
	}, nil
}
