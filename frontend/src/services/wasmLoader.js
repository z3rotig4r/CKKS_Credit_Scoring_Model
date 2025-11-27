/**
 * Wasm 모듈 로더 서비스
 * Go WebAssembly 모듈을 로드하고 초기화합니다.
 */

class WasmLoader {
  constructor() {
    this.go = null;
    this.instance = null;
    this.isLoaded = false;
    this.loadPromise = null;
  }

  /**
   * Wasm 모듈 로드 및 초기화
   * @returns {Promise<boolean>} 로드 성공 여부
   */
  async load() {
    // 이미 로드 중이면 기존 Promise 반환
    if (this.loadPromise) {
      return this.loadPromise;
    }

    // 이미 로드 완료되었으면 즉시 true 반환
    if (this.isLoaded) {
      return Promise.resolve(true);
    }

    this.loadPromise = this._doLoad();
    return this.loadPromise;
  }

  async _doLoad() {
    try {
      console.log('🔄 Loading Wasm module...');

      // wasm_exec.js 스크립트 로드
      await this._loadScript('/wasm_exec.js');

      // Go 인스턴스 생성
      if (!window.Go) {
        throw new Error('Go class not found. Make sure wasm_exec.js is loaded correctly.');
      }

      this.go = new window.Go();

      // Wasm 모듈 로드 (with cache busting)
      const cacheBuster = Date.now();
      const response = await fetch(`/main.wasm?v=${cacheBuster}`, { cache: 'no-store' });
      if (!response.ok) {
        throw new Error(`Failed to fetch main.wasm: ${response.status} ${response.statusText}`);
      }
      
      console.log(`🔄 Loading WASM with cache buster: ${cacheBuster}`);

      const wasmBytes = await response.arrayBuffer();
      const result = await WebAssembly.instantiate(wasmBytes, this.go.importObject);
      this.instance = result.instance;

      // Wasm 실행 (비동기로 실행)
      this.go.run(this.instance);

      // FHE 함수들이 등록될 때까지 대기
      console.log('⏳ Waiting for FHE functions to be registered...');
      
      const requiredFunctions = ['fheKeygen', 'fheEncrypt', 'fheDecrypt', 'fheGetParamsInfo'];
      const maxAttempts = 50; // 5초
      let attempts = 0;
      
      while (attempts < maxAttempts) {
        const allFunctionsReady = requiredFunctions.every(fn => typeof window[fn] === 'function');
        if (allFunctionsReady) {
          break;
        }
        await new Promise(resolve => setTimeout(resolve, 100));
        attempts++;
      }

      // 최종 확인
      const missingFunctions = requiredFunctions.filter(fn => typeof window[fn] !== 'function');
      
      if (missingFunctions.length > 0) {
        throw new Error(`FHE functions not available after ${maxAttempts * 100}ms: ${missingFunctions.join(', ')}`);
      }

      this.isLoaded = true;
      console.log('✅ Wasm module loaded successfully!');
      
      // 파라미터 정보 출력
      if (typeof window.fheGetParamsInfo === 'function') {
        const paramsInfo = JSON.parse(window.fheGetParamsInfo());
        console.log('📊 CKKS Parameters:', paramsInfo);
        
        // Check if MaxLevel changed - if so, warn user to regenerate keys
        if (paramsInfo.MaxLevel !== undefined) {
          const expectedMaxLevel = 5;
          if (paramsInfo.MaxLevel !== expectedMaxLevel) {
            console.warn(`⚠️ MaxLevel mismatch: Expected ${expectedMaxLevel}, Got ${paramsInfo.MaxLevel}`);
            console.warn('🔑 You may need to regenerate keys!');
          } else {
            console.log(`✅ MaxLevel correct: ${paramsInfo.MaxLevel}`);
          }
        }
      }

      return true;
    } catch (error) {
      console.error('❌ Failed to load Wasm module:', error);
      this.isLoaded = false;
      this.loadPromise = null;
      throw error;
    }
  }

  /**
   * 외부 스크립트 동적 로드
   * @param {string} src 스크립트 URL
   * @returns {Promise<void>}
   */
  _loadScript(src) {
    return new Promise((resolve, reject) => {
      // 이미 로드되어 있는지 확인
      if (document.querySelector(`script[src="${src}"]`)) {
        resolve();
        return;
      }

      const script = document.createElement('script');
      script.src = src;
      script.onload = () => resolve();
      script.onerror = () => reject(new Error(`Failed to load script: ${src}`));
      document.head.appendChild(script);
    });
  }

  /**
   * Wasm이 로드되었는지 확인
   * @returns {boolean}
   */
  isReady() {
    return this.isLoaded;
  }

  /**
   * FHE 함수 호출 래퍼
   * @param {string} functionName 함수 이름
   * @param  {...any} args 함수 인자
   * @returns {Promise<any>}
   */
  async callFunction(functionName, ...args) {
    if (!this.isLoaded) {
      throw new Error('Wasm module is not loaded yet. Call load() first.');
    }

    if (typeof window[functionName] !== 'function') {
      throw new Error(`Function ${functionName} is not available in Wasm module.`);
    }

    try {
      const startTime = performance.now();
      const result = await window[functionName](...args);
      const endTime = performance.now();
      const elapsed = endTime - startTime;
      
      // Only log if it takes more than 10ms to avoid log spam
      if (elapsed > 10 || functionName === 'fheEncrypt') {
        console.log(`⏱️ WASM ${functionName}: ${elapsed.toFixed(2)}ms`);
      }
      
      return result;
    } catch (error) {
      console.error(`Error calling ${functionName}:`, error);
      throw error;
    }
  }

  /**
   * CKKS 파라미터 정보 가져오기
   * @returns {Promise<Object>}
   */
  async getParamsInfo() {
    if (!this.isLoaded) {
      throw new Error('Wasm module is not loaded yet.');
    }

    const paramsJson = await this.callFunction('fheGetParamsInfo');
    return JSON.parse(paramsJson);
  }
}

// 싱글톤 인스턴스 export
const wasmLoader = new WasmLoader();
export default wasmLoader;
