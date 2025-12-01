/**
 * Crypto Worker - CKKS 암호화 연산을 백그라운드 스레드에서 처리
 * 
 * 장점:
 * - UI 블로킹 없이 암호화/복호화 수행
 * - 메인 스레드의 응답성 유지
 * - SharedArrayBuffer로 Zero-Copy 통신 지원
 */

// WASM 실행 환경 설정
let wasmModule = null;
let wasmReady = false;
let sharedMemory = null; // SharedArrayBuffer 지원 시 사용

/**
 * WASM 모듈 초기화
 */
async function initializeWASM() {
  try {
    // wasm_exec.js 스크립트 로드
    importScripts('/wasm_exec.js');
    
    // Go WASM 초기화
    const go = new Go();
    const result = await WebAssembly.instantiateStreaming(
      fetch('/main.wasm'),
      go.importObject
    );
    
    wasmModule = result.instance;
    go.run(wasmModule);
    
    wasmReady = true;
    
    self.postMessage({
      type: 'WASM_READY',
      success: true,
      message: 'WASM module loaded in worker',
      timestamp: Date.now()
    });
  } catch (error) {
    self.postMessage({
      type: 'WASM_ERROR',
      success: false,
      error: error.message,
      stack: error.stack
    });
  }
}

/**
 * SharedArrayBuffer 지원 확인 및 초기화
 */
function initializeSharedMemory(size) {
  try {
    if (typeof SharedArrayBuffer !== 'undefined') {
      sharedMemory = new SharedArrayBuffer(size);
      return {
        supported: true,
        buffer: sharedMemory,
        size: size
      };
    }
    return { supported: false };
  } catch (error) {
    console.warn('[Worker] SharedArrayBuffer not available:', error);
    return { supported: false };
  }
}

/**
 * 메시지 핸들러
 */
self.addEventListener('message', async (event) => {
  const { type, data, id } = event.data;
  
  try {
    switch (type) {
      case 'INIT_WASM':
        await initializeWASM();
        break;
        
      case 'CHECK_SHARED_MEMORY':
        const sharedMemResult = initializeSharedMemory(data.size || 100 * 1024 * 1024); // 100MB default
        self.postMessage({
          type: 'SHARED_MEMORY_STATUS',
          id,
          ...sharedMemResult
        });
        break;
        
      case 'GENERATE_KEYS':
        await handleGenerateKeys(data, id);
        break;
        
      case 'ENCRYPT_FEATURES':
        await handleEncryptFeatures(data, id);
        break;
        
      case 'DECRYPT_RESULT':
        await handleDecryptResult(data, id);
        break;
        
      case 'ENCRYPT_SINGLE':
        await handleEncryptSingle(data, id);
        break;
        
      default:
        throw new Error(`Unknown message type: ${type}`);
    }
  } catch (error) {
    self.postMessage({
      type: 'ERROR',
      id,
      error: error.message,
      stack: error.stack
    });
  }
});

/**
 * 키 생성 처리 (백그라운드에서 수행)
 */
async function handleGenerateKeys(data, id) {
  if (!wasmReady) {
    throw new Error('WASM not initialized. Call INIT_WASM first.');
  }
  
  const startTime = performance.now();
  
  // WASM 함수 호출
  const keys = await window.generateKeys();
  
  const duration = performance.now() - startTime;
  
  // Zero-Copy: SharedArrayBuffer 사용 시
  if (sharedMemory && data.useSharedMemory) {
    // SharedArrayBuffer에 키 데이터 복사
    const sharedView = new Uint8Array(sharedMemory);
    
    // 메타데이터 저장
    const metadata = {
      skOffset: 0,
      skLength: keys.secretKey.length,
      pkOffset: keys.secretKey.length,
      pkLength: keys.publicKey.length,
      rlkOffset: keys.secretKey.length + keys.publicKey.length,
      rlkLength: keys.relinearizationKey.length
    };
    
    // 데이터 복사
    sharedView.set(new Uint8Array(keys.secretKey), metadata.skOffset);
    sharedView.set(new Uint8Array(keys.publicKey), metadata.pkOffset);
    sharedView.set(new Uint8Array(keys.relinearizationKey), metadata.rlkOffset);
    
    self.postMessage({
      type: 'KEYS_GENERATED',
      id,
      useSharedMemory: true,
      metadata,
      duration
    });
  } else {
    // 일반 전송 (ArrayBuffer 복사)
    self.postMessage({
      type: 'KEYS_GENERATED',
      id,
      keys: {
        secretKey: keys.secretKey,
        publicKey: keys.publicKey,
        relinearizationKey: keys.relinearizationKey
      },
      duration
    }, [keys.secretKey, keys.publicKey, keys.relinearizationKey]); // Transferable objects
  }
}

/**
 * 특성 암호화 처리 (백그라운드에서 수행)
 */
async function handleEncryptFeatures(data, id) {
  if (!wasmReady) {
    throw new Error('WASM not initialized');
  }
  
  const { features, publicKey, useSharedMemory } = data;
  const startTime = performance.now();
  
  // 진행 상태 리포트
  const reportProgress = (current, total) => {
    self.postMessage({
      type: 'ENCRYPTION_PROGRESS',
      id,
      current,
      total,
      percentage: Math.round((current / total) * 100)
    });
  };
  
  const encryptedFeatures = [];
  
  for (let i = 0; i < features.length; i++) {
    const encrypted = await window.encrypt(features[i], publicKey);
    encryptedFeatures.push(encrypted);
    reportProgress(i + 1, features.length);
  }
  
  const duration = performance.now() - startTime;
  
  if (useSharedMemory && sharedMemory) {
    // Zero-Copy: SharedArrayBuffer 사용
    const sharedView = new Uint8Array(sharedMemory);
    let offset = 0;
    const metadata = [];
    
    for (const encrypted of encryptedFeatures) {
      const data = new Uint8Array(encrypted);
      sharedView.set(data, offset);
      metadata.push({ offset, length: data.length });
      offset += data.length;
    }
    
    self.postMessage({
      type: 'FEATURES_ENCRYPTED',
      id,
      useSharedMemory: true,
      metadata,
      duration,
      avgTimePerFeature: duration / features.length
    });
  } else {
    // 일반 전송
    self.postMessage({
      type: 'FEATURES_ENCRYPTED',
      id,
      encryptedFeatures,
      duration,
      avgTimePerFeature: duration / features.length
    }, encryptedFeatures); // Transferable
  }
}

/**
 * 결과 복호화 처리
 */
async function handleDecryptResult(data, id) {
  if (!wasmReady) {
    throw new Error('WASM not initialized');
  }
  
  const { encryptedResult, secretKey } = data;
  const startTime = performance.now();
  
  const result = await window.decrypt(encryptedResult, secretKey);
  const duration = performance.now() - startTime;
  
  self.postMessage({
    type: 'RESULT_DECRYPTED',
    id,
    result,
    duration
  });
}

/**
 * 단일 값 암호화 (테스트용)
 */
async function handleEncryptSingle(data, id) {
  if (!wasmReady) {
    throw new Error('WASM not initialized');
  }
  
  const { value, publicKey } = data;
  const startTime = performance.now();
  
  const encrypted = await window.encrypt(value, publicKey);
  const duration = performance.now() - startTime;
  
  self.postMessage({
    type: 'SINGLE_ENCRYPTED',
    id,
    encrypted,
    duration
  }, [encrypted]);
}

/**
 * Worker 시작 시 초기화
 */
self.postMessage({
  type: 'WORKER_READY',
  timestamp: Date.now(),
  sharedArrayBufferSupported: typeof SharedArrayBuffer !== 'undefined'
});

console.log('[Crypto Worker] Started successfully');
