package crypto

import (
	"fmt"

	"github.com/tuneinsight/lattigo/v6/core/rlwe"
	"github.com/tuneinsight/lattigo/v6/schemes/ckks"
)

type Encryptor struct {
	params    ckks.Parameters
	encoder   *ckks.Encoder
	encryptor *rlwe.Encryptor
}

// NewEncryptor 암호화 객체 생성
func NewEncryptor(params ckks.Parameters, pk *rlwe.PublicKey) *Encryptor {
	encoder := ckks.NewEncoder(params)
	encryptor := rlwe.NewEncryptor(params, pk)

	return &Encryptor{
		params:    params,
		encoder:   encoder,
		encryptor: encryptor,
	}
}

// float64 값 암호화
func (e *Encryptor) EncryptFloat64(value float64) (*rlwe.Ciphertext, error) {
	plaintext := ckks.NewPlaintext(e.params, e.params.MaxLevel())

	values := []float64{value}
	if err := e.encoder.Encode(values, plaintext); err != nil {
		return nil, fmt.Errorf("encoding failed: %v", err)
	}

	// encryption
	ciphertext, err := e.encryptor.EncryptNew(plaintext)
	if err != nil {
		return nil, fmt.Errorf("encryption failed: %v", err)
	}
	return ciphertext, nil
}

func (e *Encryptor) EncryptFloat64Slice(values []float64) (*rlwe.Ciphertext, error) {
	plaintext := ckks.NewPlaintext(e.params, e.params.MaxLevel())
	if err := e.encoder.Encode(values, plaintext); err != nil {
		return nil, fmt.Errorf("encoding failed: %v", err)
	}
	ciphertext, err := e.encryptor.EncryptNew(plaintext)
	if err != nil {
		return nil, fmt.Errorf("encryption failed: %v", err)
	}
	return ciphertext, nil
}

// EncryptFinanceData 재무 데이터 암호화
func (e *Encryptor) EncryptFinanceData(income, debtRatio, creditHistory, employment float64) ([]*rlwe.Ciphertext, error) {
	features := []float64{income, debtRatio, creditHistory, employment}
	var encryptedFeatures []*rlwe.Ciphertext

	for i, feature := range features {
		encrypted, err := e.EncryptFloat64(feature)
		if err != nil {
			return nil, fmt.Errorf("feature %d encryption failed: %v", i, err)
		}
		encryptedFeatures = append(encryptedFeatures, encrypted)
	}

	return encryptedFeatures, nil
}
