package crypto

import (
	"fmt"
	"github.com/tuneinsight/lattigo/v6/core/rlwe"
	"github.com/tuneinsight/lattigo/v6/schemes/ckks"
	"golang.org/x/sys/unix"
)

type KeyManager struct {
	params ckks.Parameters
	kgen   *rlwe.KeyGenerator
	sk     *rlwe.SecretKey
	pk     *rlwe.PublicKey
	rlk    *rlwe.RelinearizationKey
}

func NewKeyManager(params ckks.Parameters) (*KeyManager, error) {
	kgen := rlwe.NewKeyGenerator(params)

	sk := kgen.GenSecretKeyNew()
	pk := kgen.GenPublicKeyNew(sk)
	rlk := kgen.GenRelinearizationKeyNew(sk)

	return &KeyManager{
		params: params,
		kgen: kgen,
		sk: sk,
		pk: pk,
		rlk: rlk,
	}, nil
}

func (km *KeyManager) SecretKey() *rlwe.SecretKey {
	return km.sk
}

func (km *KeyManager) PublicKey() *rlwe.PublicKey {
	return km.pk
}

func (km *KeyManager) Key() *rlwe.RelinearizationKey {
	return km.rlk
}

func (km *KeyManager) GenerateRotationKeys(rotations []int) (*rlwe.RotationKeySet), error {
	rotkeys := km.kgen.GenerateRotationKeys(rotations, km.sk)

}