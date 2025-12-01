package main

import (
	"encoding/json"
	"fmt"
	"syscall/js"

	"github.com/tuneinsight/lattigo/v6/core/rlwe"
	"github.com/tuneinsight/lattigo/v6/schemes/ckks"
)

var (
	params  ckks.Parameters
	encoder *ckks.Encoder // ⚡ 전역 인코더 캐시 (재사용)
)

func init() {
	// CKKS 파라미터 초기화 (LogN=13, LogSlots=12, Scale=2^40, LogQ=[60,40,40,40,40,60])
	// MaxLevel=5 to provide sufficient depth for sigmoid evaluation
	var err error
	params, err = ckks.NewParametersFromLiteral(ckks.ParametersLiteral{
		LogN:            13,
		LogQ:            []int{60, 40, 40, 40, 40, 60},
		LogP:            []int{61},
		LogDefaultScale: 40,
	})
	if err != nil {
		panic(fmt.Sprintf("Failed to create CKKS parameters: %v", err))
	}

	// ⚡ Encoder 사전 생성 (100ms 절약!)
	encoder = ckks.NewEncoder(params)
	fmt.Println("✅ CKKS Encoder initialized (cached for reuse)")
	fmt.Println("🔖 WASM Version: 2.0-ENCODER-CACHED (2025-12-01-22:45)")
}

// keygenWrapper: FHE 키 쌍 생성
func keygenWrapper(this js.Value, args []js.Value) interface{} {
	// Promise 생성
	handler := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		resolve := args[0]
		reject := args[1]

		go func() {
			defer func() {
				if r := recover(); r != nil {
					errorConstructor := js.Global().Get("Error")
					errorObject := errorConstructor.New(fmt.Sprintf("Keygen failed: %v", r))
					reject.Invoke(errorObject)
				}
			}()

			// 키 생성
			kgen := ckks.NewKeyGenerator(params)
			sk := kgen.GenSecretKeyNew()
			pk := kgen.GenPublicKeyNew(sk)

			// 직렬화
			skBytes, err := sk.MarshalBinary()
			if err != nil {
				errorConstructor := js.Global().Get("Error")
				errorObject := errorConstructor.New(fmt.Sprintf("Failed to marshal secret key: %v", err))
				reject.Invoke(errorObject)
				return
			}

			pkBytes, err := pk.MarshalBinary()
			if err != nil {
				errorConstructor := js.Global().Get("Error")
				errorObject := errorConstructor.New(fmt.Sprintf("Failed to marshal public key: %v", err))
				reject.Invoke(errorObject)
				return
			}

			// JavaScript Uint8Array로 변환
			skArray := js.Global().Get("Uint8Array").New(len(skBytes))
			js.CopyBytesToJS(skArray, skBytes)

			pkArray := js.Global().Get("Uint8Array").New(len(pkBytes))
			js.CopyBytesToJS(pkArray, pkBytes)

			// 결과 객체 생성
			result := js.Global().Get("Object").New()
			result.Set("secretKey", skArray)
			result.Set("publicKey", pkArray)

			resolve.Invoke(result)
		}()

		return nil
	})

	promiseConstructor := js.Global().Get("Promise")
	return promiseConstructor.New(handler)
}

// genRelinearizationKeyWrapper: Relinearization Key 생성
func genRelinearizationKeyWrapper(this js.Value, args []js.Value) interface{} {
	if len(args) != 1 {
		return js.Global().Get("Error").New("genRelinearizationKey requires 1 argument: secretKey (Uint8Array)")
	}

	skArray := args[0]

	handler := js.FuncOf(func(this js.Value, promiseArgs []js.Value) interface{} {
		resolve := promiseArgs[0]
		reject := promiseArgs[1]

		go func() {
			defer func() {
				if r := recover(); r != nil {
					errorConstructor := js.Global().Get("Error")
					errorObject := errorConstructor.New(fmt.Sprintf("GenRelinearizationKey failed: %v", r))
					reject.Invoke(errorObject)
				}
			}()

			// Secret Key 역직렬화
			skBytes := make([]byte, skArray.Get("length").Int())
			js.CopyBytesToGo(skBytes, skArray)

			sk := &rlwe.SecretKey{}
			if err := sk.UnmarshalBinary(skBytes); err != nil {
				errorConstructor := js.Global().Get("Error")
				errorObject := errorConstructor.New(fmt.Sprintf("Failed to unmarshal secret key: %v", err))
				reject.Invoke(errorObject)
				return
			}

			// Relinearization Key 생성
			kgen := ckks.NewKeyGenerator(params)
			rlk := kgen.GenRelinearizationKeyNew(sk)

			// 직렬화
			rlkBytes, err := rlk.MarshalBinary()
			if err != nil {
				errorConstructor := js.Global().Get("Error")
				errorObject := errorConstructor.New(fmt.Sprintf("Failed to marshal relinearization key: %v", err))
				reject.Invoke(errorObject)
				return
			}

			// JavaScript Uint8Array로 변환
			rlkArray := js.Global().Get("Uint8Array").New(len(rlkBytes))
			js.CopyBytesToJS(rlkArray, rlkBytes)

			resolve.Invoke(rlkArray)
		}()

		return nil
	})

	promiseConstructor := js.Global().Get("Promise")
	return promiseConstructor.New(handler)
}

// genGaloisKeysWrapper: Galois Keys 생성 (회전 및 conjugation 포함)
func genGaloisKeysWrapper(this js.Value, args []js.Value) interface{} {
	if len(args) < 1 {
		return js.Global().Get("Error").New("genGaloisKeys requires at least 1 argument: secretKey (Uint8Array), optional: galoisElements (Array)")
	}

	skArray := args[0]
	var galoisElements []uint64

	// 갈루아 요소가 지정된 경우
	if len(args) > 1 && !args[1].IsUndefined() && !args[1].IsNull() {
		galElsJS := args[1]
		length := galElsJS.Length()
		galoisElements = make([]uint64, length)
		for i := 0; i < length; i++ {
			galoisElements[i] = uint64(galElsJS.Index(i).Int())
		}
	}

	handler := js.FuncOf(func(this js.Value, promiseArgs []js.Value) interface{} {
		resolve := promiseArgs[0]
		reject := promiseArgs[1]

		go func() {
			defer func() {
				if r := recover(); r != nil {
					errorConstructor := js.Global().Get("Error")
					errorObject := errorConstructor.New(fmt.Sprintf("GenGaloisKeys failed: %v", r))
					reject.Invoke(errorObject)
				}
			}()

			// Secret Key 역직렬화
			skBytes := make([]byte, skArray.Get("length").Int())
			js.CopyBytesToGo(skBytes, skArray)

			sk := &rlwe.SecretKey{}
			if err := sk.UnmarshalBinary(skBytes); err != nil {
				errorConstructor := js.Global().Get("Error")
				errorObject := errorConstructor.New(fmt.Sprintf("Failed to unmarshal secret key: %v", err))
				reject.Invoke(errorObject)
				return
			}

			// Galois Keys 생성
			kgen := ckks.NewKeyGenerator(params)
			var gks []*rlwe.GaloisKey

			if len(galoisElements) > 0 {
				// 특정 갈루아 요소에 대한 키 생성
				gks = kgen.GenGaloisKeysNew(galoisElements, sk)
			} else {
				// 기본 회전 키들 생성 (1, 2, 4, 8, ...)
				logSlots := params.LogMaxSlots()
				galoisElements = make([]uint64, 0)
				for i := 0; i < logSlots; i++ {
					galoisElements = append(galoisElements, params.GaloisElement(1<<i))
					galoisElements = append(galoisElements, params.GaloisElement(-(1 << i)))
				}
				gks = kgen.GenGaloisKeysNew(galoisElements, sk)
			}

			// 개별 직렬화 후 JSON 배열로 반환
			result := js.Global().Get("Array").New()
			for _, gk := range gks {
				gkBytes, err := gk.MarshalBinary()
				if err != nil {
					errorConstructor := js.Global().Get("Error")
					errorObject := errorConstructor.New(fmt.Sprintf("Failed to marshal Galois key: %v", err))
					reject.Invoke(errorObject)
					return
				}
				gkArray := js.Global().Get("Uint8Array").New(len(gkBytes))
				js.CopyBytesToJS(gkArray, gkBytes)
				result.Call("push", gkArray)
			}

			resolve.Invoke(result)
		}()

		return nil
	})

	promiseConstructor := js.Global().Get("Promise")
	return promiseConstructor.New(handler)
}

// genRotationKeysWrapper: 특정 회전을 위한 Rotation Keys 생성
func genRotationKeysWrapper(this js.Value, args []js.Value) interface{} {
	if len(args) != 2 {
		return js.Global().Get("Error").New("genRotationKeys requires 2 arguments: secretKey (Uint8Array), rotations (Array of numbers)")
	}

	skArray := args[0]
	rotationsJS := args[1]

	// 회전 인덱스 배열 변환
	length := rotationsJS.Length()
	rotations := make([]int, length)
	for i := 0; i < length; i++ {
		rotations[i] = rotationsJS.Index(i).Int()
	}

	handler := js.FuncOf(func(this js.Value, promiseArgs []js.Value) interface{} {
		resolve := promiseArgs[0]
		reject := promiseArgs[1]

		go func() {
			defer func() {
				if r := recover(); r != nil {
					errorConstructor := js.Global().Get("Error")
					errorObject := errorConstructor.New(fmt.Sprintf("GenRotationKeys failed: %v", r))
					reject.Invoke(errorObject)
				}
			}()

			// Secret Key 역직렬화
			skBytes := make([]byte, skArray.Get("length").Int())
			js.CopyBytesToGo(skBytes, skArray)

			sk := &rlwe.SecretKey{}
			if err := sk.UnmarshalBinary(skBytes); err != nil {
				errorConstructor := js.Global().Get("Error")
				errorObject := errorConstructor.New(fmt.Sprintf("Failed to unmarshal secret key: %v", err))
				reject.Invoke(errorObject)
				return
			}

			// 회전에 대한 갈루아 요소 계산
			galoisElements := make([]uint64, len(rotations))
			for i, rot := range rotations {
				galoisElements[i] = params.GaloisElement(rot)
			}

			// Rotation Keys 생성
			kgen := ckks.NewKeyGenerator(params)
			rotKeys := kgen.GenGaloisKeysNew(galoisElements, sk)

			// 개별 직렬화 후 배열로 반환
			result := js.Global().Get("Array").New()
			for _, rk := range rotKeys {
				rkBytes, err := rk.MarshalBinary()
				if err != nil {
					errorConstructor := js.Global().Get("Error")
					errorObject := errorConstructor.New(fmt.Sprintf("Failed to marshal rotation key: %v", err))
					reject.Invoke(errorObject)
					return
				}
				rkArray := js.Global().Get("Uint8Array").New(len(rkBytes))
				js.CopyBytesToJS(rkArray, rkBytes)
				result.Call("push", rkArray)
			}

			resolve.Invoke(result)
		}()

		return nil
	})

	promiseConstructor := js.Global().Get("Promise")
	return promiseConstructor.New(handler)
}

// genConjugationKeyWrapper: Conjugation Key 생성
func genConjugationKeyWrapper(this js.Value, args []js.Value) interface{} {
	if len(args) != 1 {
		return js.Global().Get("Error").New("genConjugationKey requires 1 argument: secretKey (Uint8Array)")
	}

	skArray := args[0]

	handler := js.FuncOf(func(this js.Value, promiseArgs []js.Value) interface{} {
		resolve := promiseArgs[0]
		reject := promiseArgs[1]

		go func() {
			defer func() {
				if r := recover(); r != nil {
					errorConstructor := js.Global().Get("Error")
					errorObject := errorConstructor.New(fmt.Sprintf("GenConjugationKey failed: %v", r))
					reject.Invoke(errorObject)
				}
			}()

			// Secret Key 역직렬화
			skBytes := make([]byte, skArray.Get("length").Int())
			js.CopyBytesToGo(skBytes, skArray)

			sk := &rlwe.SecretKey{}
			if err := sk.UnmarshalBinary(skBytes); err != nil {
				errorConstructor := js.Global().Get("Error")
				errorObject := errorConstructor.New(fmt.Sprintf("Failed to unmarshal secret key: %v", err))
				reject.Invoke(errorObject)
				return
			}

			// Conjugation Key 생성 (갈루아 요소 -1 사용)
			kgen := ckks.NewKeyGenerator(params)
			conjKey := kgen.GenGaloisKeyNew(params.GaloisElementForComplexConjugation(), sk)

			// 직렬화
			conjKeyBytes, err := conjKey.MarshalBinary()
			if err != nil {
				errorConstructor := js.Global().Get("Error")
				errorObject := errorConstructor.New(fmt.Sprintf("Failed to marshal conjugation key: %v", err))
				reject.Invoke(errorObject)
				return
			}

			// JavaScript Uint8Array로 변환
			conjKeyArray := js.Global().Get("Uint8Array").New(len(conjKeyBytes))
			js.CopyBytesToJS(conjKeyArray, conjKeyBytes)

			resolve.Invoke(conjKeyArray)
		}()

		return nil
	})

	promiseConstructor := js.Global().Get("Promise")
	return promiseConstructor.New(handler)
}

// genAllKeysWrapper: 모든 평가 키 한번에 생성 (SK, PK, RLK, Galois Keys)
func genAllKeysWrapper(this js.Value, args []js.Value) interface{} {
	handler := js.FuncOf(func(this js.Value, promiseArgs []js.Value) interface{} {
		resolve := promiseArgs[0]
		reject := promiseArgs[1]

		go func() {
			defer func() {
				if r := recover(); r != nil {
					errorConstructor := js.Global().Get("Error")
					errorObject := errorConstructor.New(fmt.Sprintf("GenAllKeys failed: %v", r))
					reject.Invoke(errorObject)
				}
			}()

			// 키 생성기
			kgen := ckks.NewKeyGenerator(params)

			// 1. Secret Key & Public Key
			sk := kgen.GenSecretKeyNew()
			pk := kgen.GenPublicKeyNew(sk)

			// 2. Relinearization Key
			rlk := kgen.GenRelinearizationKeyNew(sk)

			// 3. Galois Keys (회전 + Conjugation)
			logSlots := params.LogMaxSlots()
			galEls := make([]uint64, 0)

			// 2의 거듭제곱 회전 (1, 2, 4, 8, ...)
			for i := 0; i < logSlots; i++ {
				galEls = append(galEls, params.GaloisElement(1<<i))
				galEls = append(galEls, params.GaloisElement(-(1 << i)))
			}
			// Conjugation
			galEls = append(galEls, params.GaloisElementForComplexConjugation())

			gks := kgen.GenGaloisKeysNew(galEls, sk)

			// 직렬화
			skBytes, _ := sk.MarshalBinary()
			pkBytes, _ := pk.MarshalBinary()
			rlkBytes, _ := rlk.MarshalBinary()

			// JavaScript Uint8Array로 변환
			skArray := js.Global().Get("Uint8Array").New(len(skBytes))
			js.CopyBytesToJS(skArray, skBytes)

			pkArray := js.Global().Get("Uint8Array").New(len(pkBytes))
			js.CopyBytesToJS(pkArray, pkBytes)

			rlkArray := js.Global().Get("Uint8Array").New(len(rlkBytes))
			js.CopyBytesToJS(rlkArray, rlkBytes)

			// Galois Keys 배열
			gksArrayJS := js.Global().Get("Array").New()
			for _, gk := range gks {
				gkBytes, _ := gk.MarshalBinary()
				gkArray := js.Global().Get("Uint8Array").New(len(gkBytes))
				js.CopyBytesToJS(gkArray, gkBytes)
				gksArrayJS.Call("push", gkArray)
			}

			// 결과 객체 생성
			result := js.Global().Get("Object").New()
			result.Set("secretKey", skArray)
			result.Set("publicKey", pkArray)
			result.Set("relinearizationKey", rlkArray)
			result.Set("galoisKeys", gksArrayJS)

			resolve.Invoke(result)
		}()

		return nil
	})

	promiseConstructor := js.Global().Get("Promise")
	return promiseConstructor.New(handler)
}

// encryptWrapper: 평문 암호화
func encryptWrapper(this js.Value, args []js.Value) interface{} {
	if len(args) != 2 {
		return js.Global().Get("Error").New("encryptWrapper requires 2 arguments: publicKey (Uint8Array), plaintext (number)")
	}

	// 외부 스코프에 인자 저장
	pkArray := args[0]
	plaintext := args[1].Float()

	// Promise 생성
	handler := js.FuncOf(func(this js.Value, promiseArgs []js.Value) interface{} {
		resolve := promiseArgs[0]
		reject := promiseArgs[1]

		go func() {
			// ⏱️ 전체 시작 시간
			totalStart := js.Global().Get("performance").Call("now").Float()

			defer func() {
				if r := recover(); r != nil {
					errorConstructor := js.Global().Get("Error")
					errorObject := errorConstructor.New(fmt.Sprintf("Encrypt failed: %v", r))
					reject.Invoke(errorObject)
				}
			}()

			// Public Key 역직렬화
			t0 := js.Global().Get("performance").Call("now").Float()
			pkBytes := make([]byte, pkArray.Get("length").Int())
			js.CopyBytesToGo(pkBytes, pkArray)

			pk := &rlwe.PublicKey{}
			if err := pk.UnmarshalBinary(pkBytes); err != nil {
				errorConstructor := js.Global().Get("Error")
				errorObject := errorConstructor.New(fmt.Sprintf("Failed to unmarshal public key: %v", err))
				reject.Invoke(errorObject)
				return
			}
			t1 := js.Global().Get("performance").Call("now").Float()
			fmt.Printf("  [Go] PK unmarshal: %.2fms\n", t1-t0)

			// 평문 인코딩 (complex128로 변환)
			// ⚡ 전역 encoder 재사용 (캐싱 효과!)
			t2 := js.Global().Get("performance").Call("now").Float()
			values := make([]complex128, params.MaxSlots())
			values[0] = complex(plaintext, 0) // 실수를 복소수로 변환
			pt := ckks.NewPlaintext(params, params.MaxLevel())
			if err := encoder.Encode(values, pt); err != nil {
				errorConstructor := js.Global().Get("Error")
				errorObject := errorConstructor.New(fmt.Sprintf("Failed to encode plaintext: %v", err))
				reject.Invoke(errorObject)
				return
			}
			t3 := js.Global().Get("performance").Call("now").Float()
			fmt.Printf("  [Go] Encoding: %.2fms\n", t3-t2)

			// 암호화
			t4 := js.Global().Get("performance").Call("now").Float()
			encryptor := ckks.NewEncryptor(params, pk)
			t5 := js.Global().Get("performance").Call("now").Float()
			fmt.Printf("  [Go] Encryptor creation: %.2fms\n", t5-t4)

			ct, err := encryptor.EncryptNew(pt)
			t6 := js.Global().Get("performance").Call("now").Float()
			fmt.Printf("  [Go] Actual encryption: %.2fms\n", t6-t5)
			if err != nil {
				errorConstructor := js.Global().Get("Error")
				errorObject := errorConstructor.New(fmt.Sprintf("Failed to encrypt: %v", err))
				reject.Invoke(errorObject)
				return
			}

			// 암호문 직렬화
			t7 := js.Global().Get("performance").Call("now").Float()
			ctBytes, err := ct.MarshalBinary()
			t8 := js.Global().Get("performance").Call("now").Float()
			fmt.Printf("  [Go] CT marshal: %.2fms\n", t8-t7)
			if err != nil {
				errorConstructor := js.Global().Get("Error")
				errorObject := errorConstructor.New(fmt.Sprintf("Failed to marshal ciphertext: %v", err))
				reject.Invoke(errorObject)
				return
			}

			// JavaScript Uint8Array로 변환
			t9 := js.Global().Get("performance").Call("now").Float()
			ctArray := js.Global().Get("Uint8Array").New(len(ctBytes))
			js.CopyBytesToJS(ctArray, ctBytes)
			t10 := js.Global().Get("performance").Call("now").Float()
			fmt.Printf("  [Go] JS copy: %.2fms\n", t10-t9)

			totalEnd := js.Global().Get("performance").Call("now").Float()
			fmt.Printf("  [Go] ⚡ TOTAL: %.2fms\n", totalEnd-totalStart)

			resolve.Invoke(ctArray)
		}()

		return nil
	})

	promiseConstructor := js.Global().Get("Promise")
	return promiseConstructor.New(handler)
}

// decryptWrapper: 암호문 복호화
func decryptWrapper(this js.Value, args []js.Value) interface{} {
	if len(args) != 2 {
		return js.Global().Get("Error").New("decryptWrapper requires 2 arguments: secretKey (Uint8Array), ciphertext (Uint8Array)")
	}

	// 외부 스코프에 인자 저장
	skArray := args[0]
	ctArray := args[1]

	// Promise 생성
	handler := js.FuncOf(func(this js.Value, promiseArgs []js.Value) interface{} {
		resolve := promiseArgs[0]
		reject := promiseArgs[1]

		go func() {
			defer func() {
				if r := recover(); r != nil {
					errorConstructor := js.Global().Get("Error")
					errorObject := errorConstructor.New(fmt.Sprintf("Decrypt failed: %v", r))
					reject.Invoke(errorObject)
				}
			}()

			// Secret Key 역직렬화
			skBytes := make([]byte, skArray.Get("length").Int())
			js.CopyBytesToGo(skBytes, skArray)

			sk := &rlwe.SecretKey{}
			if err := sk.UnmarshalBinary(skBytes); err != nil {
				errorConstructor := js.Global().Get("Error")
				errorObject := errorConstructor.New(fmt.Sprintf("Failed to unmarshal secret key: %v", err))
				reject.Invoke(errorObject)
				return
			}

			// Ciphertext 역직렬화 - 레벨을 미리 지정하지 말고 UnmarshalBinary가 자동으로 설정하게 함
			ctBytes := make([]byte, ctArray.Get("length").Int())
			js.CopyBytesToGo(ctBytes, ctArray)

			// 빈 암호문 생성 후 역직렬화 (레벨은 자동으로 복원됨)
			ct := new(rlwe.Ciphertext)
			if err := ct.UnmarshalBinary(ctBytes); err != nil {
				errorConstructor := js.Global().Get("Error")
				errorObject := errorConstructor.New(fmt.Sprintf("Failed to unmarshal ciphertext: %v", err))
				reject.Invoke(errorObject)
				return
			}

			// 복호화
			decryptor := ckks.NewDecryptor(params, sk)
			pt := decryptor.DecryptNew(ct)

			// 디코딩
			// ⚡ 전역 encoder 재사용
			values := make([]complex128, params.MaxSlots())
			if err := encoder.Decode(pt, values); err != nil {
				errorConstructor := js.Global().Get("Error")
				errorObject := errorConstructor.New(fmt.Sprintf("Failed to decode plaintext: %v", err))
				reject.Invoke(errorObject)
				return
			}

			// 첫 번째 값 반환 (실수부만)
			result := real(values[0])
			resolve.Invoke(js.ValueOf(result))
		}()

		return nil
	})

	promiseConstructor := js.Global().Get("Promise")
	return promiseConstructor.New(handler)
}

// encryptVectorWrapper: 벡터 (여러 값) 한번에 암호화 - SIMD packing
func encryptVectorWrapper(this js.Value, args []js.Value) interface{} {
	if len(args) != 2 {
		return js.Global().Get("Error").New("encryptVector requires 2 arguments: publicKey (Uint8Array), values (Array of numbers)")
	}

	pkArray := args[0]
	valuesJS := args[1]

	handler := js.FuncOf(func(this js.Value, promiseArgs []js.Value) interface{} {
		resolve := promiseArgs[0]
		reject := promiseArgs[1]

		go func() {
			defer func() {
				if r := recover(); r != nil {
					errorConstructor := js.Global().Get("Error")
					errorObject := errorConstructor.New(fmt.Sprintf("EncryptVector failed: %v", r))
					reject.Invoke(errorObject)
				}
			}()

			// Public Key 역직렬화
			pkBytes := make([]byte, pkArray.Get("length").Int())
			js.CopyBytesToGo(pkBytes, pkArray)

			pk := &rlwe.PublicKey{}
			if err := pk.UnmarshalBinary(pkBytes); err != nil {
				errorConstructor := js.Global().Get("Error")
				errorObject := errorConstructor.New(fmt.Sprintf("Failed to unmarshal public key: %v", err))
				reject.Invoke(errorObject)
				return
			}

			// JavaScript array를 Go slice로 변환
			length := valuesJS.Length()
			if length > params.MaxSlots() {
				errorConstructor := js.Global().Get("Error")
				errorObject := errorConstructor.New(fmt.Sprintf("Vector size %d exceeds MaxSlots %d", length, params.MaxSlots()))
				reject.Invoke(errorObject)
				return
			}

			values := make([]complex128, params.MaxSlots())
			for i := 0; i < length; i++ {
				values[i] = complex(valuesJS.Index(i).Float(), 0)
			}

			// 평문 인코딩
			// ⚡ 전역 encoder 재사용
			pt := ckks.NewPlaintext(params, params.MaxLevel())
			if err := encoder.Encode(values, pt); err != nil {
				errorConstructor := js.Global().Get("Error")
				errorObject := errorConstructor.New(fmt.Sprintf("Failed to encode plaintext vector: %v", err))
				reject.Invoke(errorObject)
				return
			}

			// 암호화
			encryptor := ckks.NewEncryptor(params, pk)
			ct, err := encryptor.EncryptNew(pt)
			if err != nil {
				errorConstructor := js.Global().Get("Error")
				errorObject := errorConstructor.New(fmt.Sprintf("Failed to encrypt vector: %v", err))
				reject.Invoke(errorObject)
				return
			}

			// 직렬화
			ctBytes, err := ct.MarshalBinary()
			if err != nil {
				errorConstructor := js.Global().Get("Error")
				errorObject := errorConstructor.New(fmt.Sprintf("Failed to marshal ciphertext: %v", err))
				reject.Invoke(errorObject)
				return
			}

			// JavaScript Uint8Array로 변환
			ctArray := js.Global().Get("Uint8Array").New(len(ctBytes))
			js.CopyBytesToJS(ctArray, ctBytes)

			resolve.Invoke(ctArray)
		}()

		return nil
	})

	promiseConstructor := js.Global().Get("Promise")
	return promiseConstructor.New(handler)
}

// getParamsInfo: 파라미터 정보 반환
func getParamsInfo(this js.Value, args []js.Value) interface{} {
	info := map[string]interface{}{
		"LogN":         params.LogN(),
		"LogQ":         params.LogQ(),
		"LogP":         params.LogP(),
		"MaxLevel":     params.MaxLevel(),
		"MaxSlots":     params.MaxSlots(),
		"DefaultScale": params.DefaultScale().Float64(),
		"RingType":     params.RingType().String(),
	}

	jsonBytes, err := json.Marshal(info)
	if err != nil {
		return js.Global().Get("Error").New(fmt.Sprintf("Failed to marshal params info: %v", err))
	}

	return js.ValueOf(string(jsonBytes))
}

func main() {
	fmt.Println("Lattigo CKKS Wasm module initialized")
	fmt.Printf("Parameters: LogN=%d, LogQ=%v, MaxLevel=%d, MaxSlots=%d\n",
		params.LogN(), params.LogQ(), params.MaxLevel(), params.MaxSlots())

	// JavaScript 전역 객체에 함수 등록
	js.Global().Set("fheKeygen", js.FuncOf(keygenWrapper))
	js.Global().Set("fheEncrypt", js.FuncOf(encryptWrapper))
	js.Global().Set("fheEncryptVector", js.FuncOf(encryptVectorWrapper))
	js.Global().Set("fheDecrypt", js.FuncOf(decryptWrapper))
	js.Global().Set("fheGetParamsInfo", js.FuncOf(getParamsInfo))

	// 추가 키 생성 함수들
	js.Global().Set("fheGenRelinearizationKey", js.FuncOf(genRelinearizationKeyWrapper))
	js.Global().Set("fheGenGaloisKeys", js.FuncOf(genGaloisKeysWrapper))
	js.Global().Set("fheGenRotationKeys", js.FuncOf(genRotationKeysWrapper))
	js.Global().Set("fheGenConjugationKey", js.FuncOf(genConjugationKeyWrapper))
	js.Global().Set("fheGenAllKeys", js.FuncOf(genAllKeysWrapper))

	fmt.Println("FHE functions exposed to JavaScript:")
	fmt.Println("  - fheKeygen()")
	fmt.Println("  - fheEncrypt(publicKey, plaintext)")
	fmt.Println("  - fheEncryptVector(publicKey, [values])")
	fmt.Println("  - fheDecrypt(secretKey, ciphertext)")
	fmt.Println("  - fheGetParamsInfo()")
	fmt.Println("  - fheGenRelinearizationKey(secretKey)")
	fmt.Println("  - fheGenGaloisKeys(secretKey, [galoisElements])")
	fmt.Println("  - fheGenRotationKeys(secretKey, [rotations])")
	fmt.Println("  - fheGenConjugationKey(secretKey)")
	fmt.Println("  - fheGenAllKeys()")

	// 프로그램이 종료되지 않도록 무한 대기
	select {}
}
