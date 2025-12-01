/**
 * Web Worker Manager
 * 
 * Crypto Worker를 관리하고 메시지 통신을 담당하는 서비스
 */

class WorkerManager {
  constructor() {
    this.worker = null;
    this.messageId = 0;
    this.pendingRequests = new Map();
    this.sharedMemorySupported = false;
    this.sharedBuffer = null;
    this.isInitialized = false;
  }

  /**
   * Worker 초기화
   */
  async initialize() {
    if (this.isInitialized) {
      console.log('[WorkerManager] Already initialized');
      return;
    }

    return new Promise((resolve, reject) => {
      try {
        // Worker 생성
        this.worker = new Worker('/crypto.worker.js');
        
        // 메시지 핸들러 등록
        this.worker.addEventListener('message', this.handleMessage.bind(this));
        this.worker.addEventListener('error', this.handleError.bind(this));
        
        // Worker 준비 대기
        const initTimeout = setTimeout(() => {
          reject(new Error('Worker initialization timeout'));
        }, 10000);
        
        const initHandler = (event) => {
          if (event.data.type === 'WORKER_READY') {
            clearTimeout(initTimeout);
            this.sharedMemorySupported = event.data.sharedArrayBufferSupported;
            console.log(`[WorkerManager] Worker ready. SharedArrayBuffer: ${this.sharedMemorySupported}`);
            
            // WASM 초기화 요청
            this.sendMessage({ type: 'INIT_WASM' })
              .then(() => {
                this.isInitialized = true;
                resolve();
              })
              .catch(reject);
          }
        };
        
        this.worker.addEventListener('message', initHandler, { once: true });
        
      } catch (error) {
        reject(error);
      }
    });
  }

  /**
   * SharedArrayBuffer 초기화 확인
   */
  async initializeSharedMemory(size = 100 * 1024 * 1024) {
    if (!this.sharedMemorySupported) {
      console.warn('[WorkerManager] SharedArrayBuffer not supported');
      return false;
    }

    try {
      const result = await this.sendMessage({
        type: 'CHECK_SHARED_MEMORY',
        data: { size }
      });

      if (result.supported) {
        this.sharedBuffer = new SharedArrayBuffer(size);
        console.log(`[WorkerManager] SharedArrayBuffer initialized: ${size} bytes`);
        return true;
      }
    } catch (error) {
      console.error('[WorkerManager] SharedArrayBuffer initialization failed:', error);
    }
    
    return false;
  }

  /**
   * 메시지 전송 (Promise 기반)
   */
  sendMessage(message, transfer = []) {
    return new Promise((resolve, reject) => {
      if (!this.worker) {
        reject(new Error('Worker not initialized'));
        return;
      }

      const id = this.messageId++;
      const messageWithId = { ...message, id };

      // 응답 대기 등록
      this.pendingRequests.set(id, { resolve, reject });

      // 타임아웃 설정 (30초)
      const timeout = setTimeout(() => {
        this.pendingRequests.delete(id);
        reject(new Error(`Worker request timeout: ${message.type}`));
      }, 30000);

      // 타임아웃을 저장하여 나중에 클리어
      this.pendingRequests.get(id).timeout = timeout;

      // 메시지 전송
      if (transfer.length > 0) {
        this.worker.postMessage(messageWithId, transfer);
      } else {
        this.worker.postMessage(messageWithId);
      }
    });
  }

  /**
   * Worker로부터 메시지 수신
   */
  handleMessage(event) {
    const { type, id, error, ...data } = event.data;

    // 진행 상태 메시지는 별도 처리
    if (type === 'ENCRYPTION_PROGRESS') {
      this.onProgressCallback?.(data);
      return;
    }

    // ID가 있는 응답 메시지 처리
    if (id !== undefined && this.pendingRequests.has(id)) {
      const { resolve, reject, timeout } = this.pendingRequests.get(id);
      clearTimeout(timeout);
      this.pendingRequests.delete(id);

      if (error) {
        reject(new Error(error));
      } else {
        resolve({ type, ...data });
      }
      return;
    }

    // 일반 메시지 (이벤트 핸들러로 전달)
    console.log('[WorkerManager] Received:', type, data);
  }

  /**
   * Worker 에러 핸들러
   */
  handleError(error) {
    console.error('[WorkerManager] Worker error:', error);
    
    // 모든 대기 중인 요청을 에러로 처리
    for (const [id, { reject, timeout }] of this.pendingRequests) {
      clearTimeout(timeout);
      reject(new Error('Worker error'));
    }
    this.pendingRequests.clear();
  }

  /**
   * 진행 상태 콜백 등록
   */
  setProgressCallback(callback) {
    this.onProgressCallback = callback;
  }

  /**
   * 키 생성 (Worker에서 수행)
   */
  async generateKeys(useSharedMemory = true) {
    const result = await this.sendMessage({
      type: 'GENERATE_KEYS',
      data: { useSharedMemory: useSharedMemory && this.sharedMemorySupported }
    });

    if (result.useSharedMemory && this.sharedBuffer) {
      // Zero-Copy: SharedArrayBuffer에서 데이터 읽기
      const { metadata } = result;
      const sharedView = new Uint8Array(this.sharedBuffer);

      return {
        secretKey: sharedView.slice(metadata.skOffset, metadata.skOffset + metadata.skLength),
        publicKey: sharedView.slice(metadata.pkOffset, metadata.pkOffset + metadata.pkLength),
        relinearizationKey: sharedView.slice(metadata.rlkOffset, metadata.rlkOffset + metadata.rlkLength),
        duration: result.duration,
        zeroCoopy: true
      };
    } else {
      // 일반 복사
      return {
        ...result.keys,
        duration: result.duration,
        zeroCopy: false
      };
    }
  }

  /**
   * 특성 암호화 (Worker에서 수행)
   */
  async encryptFeatures(features, publicKey, useSharedMemory = true) {
    const result = await this.sendMessage({
      type: 'ENCRYPT_FEATURES',
      data: {
        features,
        publicKey,
        useSharedMemory: useSharedMemory && this.sharedMemorySupported
      }
    }, [publicKey.buffer]); // publicKey를 transfer

    if (result.useSharedMemory && this.sharedBuffer) {
      // Zero-Copy: SharedArrayBuffer에서 암호문 읽기
      const { metadata } = result;
      const sharedView = new Uint8Array(this.sharedBuffer);
      
      const encryptedFeatures = metadata.map(meta => 
        sharedView.slice(meta.offset, meta.offset + meta.length)
      );

      return {
        encryptedFeatures,
        duration: result.duration,
        avgTimePerFeature: result.avgTimePerFeature,
        zeroCopy: true
      };
    } else {
      return {
        encryptedFeatures: result.encryptedFeatures,
        duration: result.duration,
        avgTimePerFeature: result.avgTimePerFeature,
        zeroCopy: false
      };
    }
  }

  /**
   * 결과 복호화 (Worker에서 수행)
   */
  async decryptResult(encryptedResult, secretKey) {
    const result = await this.sendMessage({
      type: 'DECRYPT_RESULT',
      data: { encryptedResult, secretKey }
    }, [encryptedResult.buffer, secretKey.buffer]);

    return {
      result: result.result,
      duration: result.duration
    };
  }

  /**
   * Worker 종료
   */
  terminate() {
    if (this.worker) {
      this.worker.terminate();
      this.worker = null;
      this.isInitialized = false;
      console.log('[WorkerManager] Worker terminated');
    }
  }

  /**
   * 리소스 정리
   */
  cleanup() {
    this.terminate();
    this.pendingRequests.clear();
    this.sharedBuffer = null;
    this.onProgressCallback = null;
  }
}

// Singleton instance
export const workerManager = new WorkerManager();

export default workerManager;
