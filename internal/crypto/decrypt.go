package crypto

import (
	"fmt"

	"github.com/tuneinsight/lattigo/v6/core/rlwe"
	"github.com/tuneinsight/lattigo/v6/schemes/ckks"
)

type Decryptor struct {
	params    ckks.Parameters
	encoder   *ckks.Encoder
	decryptor *rlwe.Decryptor
}

func NewDecryptor(params ckks.Parameters, sk *rlwe.SecretKey) *Decryptor {
	encoder := ckks.NewEncoder(params)
	decryptor := rlwe.NewDecryptor(params, sk)

	return &Decryptor{
		params:    params,
		encoder:   encoder,
		decryptor: decryptor,
	}
}

// Ciphertext Decryption to float64
func (d *Decryptor) DecryptFloat64(ciphertext *rlwe.Ciphertext) (float64, error) {
	plaintext := d.decryptor.DecryptNew(ciphertext)

	values := make([]float64, d.params.MaxSlots())
	if err := d.encoder.Decode(plaintext, values); err != nil {
		return 0, fmt.Errorf("decoding error: %v", err)
	}
	return values[0], nil
}

// Ciphertext -> float64 slices (Decryption)
func (d *Decryptor) DecryptFloat64Slice(ciphertext *rlwe.Ciphertext) ([]float64, error) {
	plaintext := d.decryptor.DecryptNew(ciphertext)

	values := make([]float64, d.params.MaxSlots())
	if err := d.encoder.Decode(plaintext, values); err != nil {
		return nil, fmt.Errorf("decoding error: %v", err)
	}
	return values, nil
}

func (d *Decryptor) DecryptCreditScore(encryptedScore *rlwe.Ciphertext) (float64, error) {
	score, err := d.DecryptFloat64(encryptedScore)
	if err != nil {
		return 0, fmt.Errorf("credit score decryption error: %v", err)
	}

	if score < 300 {
		score = 300
	} else if score > 850 {
		score = 850
	}
	return score, nil
}
