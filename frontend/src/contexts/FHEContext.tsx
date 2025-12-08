import React, { createContext, useContext, useState, useEffect, useCallback } from 'react';
import wasmLoader from '../services/wasmLoader';
import { creditAPI } from '../services/apiService';
import { indexedDBService } from '../services/indexedDBService';
// Parallel encryption via Web Workers doesn't provide speedup for WASM due to overhead
const isWebWorkerSupported = () => false; // Disabled

export interface TimingStats {
  keygenTime?: number;
  encryptionTime?: number;
  decryptionTime?: number;
  inferenceTime?: number;      // Server-side inference time (from backend)
  networkTime?: number;        // Network round-trip time (total - server time)
}

interface FHEContextType {
  wasmLoaded: boolean;
  isReady: boolean;
  loading: boolean;
  error: string | null;
  hasStoredKeys: boolean;
  timings: TimingStats;
  encrypt: (value: number) => Promise<Uint8Array>;
  encryptMultiple: (values: number[], onProgress?: (current: number, total: number) => void) => Promise<Uint8Array[]>;
  decrypt: (ciphertext: Uint8Array) => Promise<number>;
  sendToBackend: (encryptedFeatures: Uint8Array[]) => Promise<Uint8Array>;
  generateKeys: (password?: string) => Promise<void>;
  loadStoredKeys: (password: string) => Promise<void>;
  deleteStoredKeys: () => Promise<void>;
  supportsParallelEncryption: boolean;
}

const FHEContext = createContext<FHEContextType | undefined>(undefined);

export const useFHE = (): FHEContextType => {
  const context = useContext(FHEContext);
  if (!context) {
    throw new Error('useFHE must be used within FHEProvider');
  }
  return context;
};

interface Keys {
  secretKey: Uint8Array | null;
  publicKey: Uint8Array | null;
  relinearizationKey: Uint8Array | null;
}

export const FHEProvider: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const [wasmLoaded, setWasmLoaded] = useState(false);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [keys, setKeys] = useState<Keys>({ secretKey: null, publicKey: null, relinearizationKey: null });
  const [hasStoredKeys, setHasStoredKeys] = useState(false);
  const [timings, setTimings] = useState<TimingStats>({});
  const [lastActivity, setLastActivity] = useState<number>(Date.now());
  const [supportsParallelEncryption] = useState(isWebWorkerSupported());

  // Idle timeout: 5분 후 sk 자동 폐기
  const IDLE_TIMEOUT = 5 * 60 * 1000; // 5 minutes

  // Initialize WASM and check for stored keys
  useEffect(() => {
    const initialize = async () => {
      try {
        setLoading(true);
        setError(null);

        // Initialize IndexedDB
        await indexedDBService.init();

        // Check if keys are stored
        const keysExist = await indexedDBService.hasKeys();
        setHasStoredKeys(keysExist);
        console.log(`🔑 Stored keys exist: ${keysExist}`);

        // Load WASM module
        console.log('🔄 Loading WASM module...');
        const startTime = performance.now();
        await wasmLoader.load();
        const loadTime = performance.now() - startTime;
        console.log(`✅ WASM loaded in ${loadTime.toFixed(2)}ms`);
        setWasmLoaded(true);

        console.log('✅ FHE system initialized successfully');
      } catch (err: any) {
        console.error('❌ FHE initialization failed:', err);
        setError(err.message || 'Failed to initialize FHE system');
      } finally {
        setLoading(false);
      }
    };

    initialize();
  }, []);

  // Idle timeout 체크
  useEffect(() => {
    const checkIdle = setInterval(() => {
      if (keys.secretKey && Date.now() - lastActivity > IDLE_TIMEOUT) {
        console.warn('⏰ Idle timeout: Clearing secret key from memory');
        setKeys((prev) => ({ ...prev, secretKey: null }));
        alert('Session expired due to inactivity. Please load keys again.');
      }
    }, 30000); // 30초마다 체크

    return () => clearInterval(checkIdle);
  }, [keys.secretKey, lastActivity, IDLE_TIMEOUT]);

  // Activity tracker
  const updateActivity = useCallback(() => {
    setLastActivity(Date.now());
  }, []);

  const generateKeys = useCallback(
    async (password?: string) => {
      if (!wasmLoaded) {
        throw new Error('WASM not loaded');
      }

      console.log('🔄 Generating FHE keys...');
      
      // Log WASM parameters first
      try {
        const paramsInfo = await wasmLoader.getParamsInfo() as any;
        console.log('📊 WASM Parameters before keygen:', paramsInfo);
        if (paramsInfo.MaxLevel !== 5) {
          console.error('❌ CRITICAL: MaxLevel is not 5! WASM may be cached.');
          console.error('   Please hard refresh (Ctrl+Shift+R) or clear browser cache.');
        }
      } catch (e) {
        console.warn('⚠️ Could not get params info:', e);
      }
      
      const startTime = performance.now();

      const keyPair = await wasmLoader.callFunction('fheKeygen');

      // Generate relinearization key
      console.log('🔄 Generating relinearization key...');
      const rlkStartTime = performance.now();
      const relinearizationKey = await wasmLoader.callFunction('fheGenRelinearizationKey', keyPair.secretKey);
      const rlkTime = performance.now() - rlkStartTime;
      console.log(`⏱️ Relinearization key generation time: ${rlkTime.toFixed(2)}ms`);

      const endTime = performance.now();
      const keygenTime = endTime - startTime;

      console.log(`⏱️ Total key generation time: ${keygenTime.toFixed(2)}ms`);

      setKeys({
        secretKey: keyPair.secretKey,
        publicKey: keyPair.publicKey,
        relinearizationKey,
      });

      setTimings((prev) => ({ ...prev, keygenTime }));

      // Save to IndexedDB if password provided
      if (password) {
        await indexedDBService.saveKeys(
          {
            secretKey: keyPair.secretKey,
            publicKey: keyPair.publicKey,
            relinearizationKey,
          },
          password
        );
        setHasStoredKeys(true);
        console.log('💾 Keys saved to IndexedDB');
      }
    },
    [wasmLoaded]
  );

  const loadStoredKeys = useCallback(
    async (password: string) => {
      try {
        console.log('🔄 Loading keys from IndexedDB...');
        const storedKeys = await indexedDBService.loadKeys(password);

        if (!storedKeys) {
          throw new Error('No keys found in storage');
        }

        setKeys({
          secretKey: storedKeys.secretKey,
          publicKey: storedKeys.publicKey,
          relinearizationKey: storedKeys.relinearizationKey || null,
        });

        console.log('✅ Keys loaded successfully');
      } catch (err: any) {
        console.error('❌ Failed to load keys:', err);
        throw new Error(`Failed to load keys: ${err.message}`);
      }
    },
    []
  );

  const deleteStoredKeys = useCallback(async () => {
    await indexedDBService.deleteKeys();
    setHasStoredKeys(false);
    console.log('🗑️ Keys deleted from storage');
  }, []);

  const encrypt = useCallback(
    async (value: number): Promise<Uint8Array> => {
      if (!wasmLoaded || !keys.publicKey) {
        throw new Error('FHE system not ready');
      }

      updateActivity(); // Activity 갱신

      try {
        const startTime = performance.now();
        console.log(`🔐 Encrypting value ${value.toFixed(4)}...`);

        const ciphertext = await wasmLoader.callFunction(
          'fheEncrypt',
          keys.publicKey,
          value
        );

        const endTime = performance.now();
        const encryptTime = endTime - startTime;
        console.log(`✅ Encrypted in ${encryptTime.toFixed(2)}ms (size: ${ciphertext.byteLength} bytes)`);

        setTimings((prev) => ({
          ...prev,
          encryptionTime: (prev.encryptionTime || 0) + encryptTime,
        }));

        return ciphertext as Uint8Array;
      } catch (err: any) {
        console.error('Encryption failed:', err);
        throw new Error(`Encryption failed: ${err.message}`);
      }
    },
    [wasmLoaded, keys.publicKey, updateActivity]
  );

  const encryptMultiple = useCallback(
    async (values: number[], onProgress?: (current: number, total: number) => void): Promise<Uint8Array[]> => {
      if (!wasmLoaded || !keys.publicKey) {
        throw new Error('FHE system not ready');
      }

      updateActivity();

      try {
        const startTime = performance.now();
        
        // Just use sequential encryption with proper progress tracking
        // Web Workers don't provide speedup for WASM operations due to overhead
        console.log(`🔄 Encrypting ${values.length} features...`);
        const encryptedFeatures: Uint8Array[] = [];
        
        for (let i = 0; i < values.length; i++) {
          const featureStartTime = performance.now();
          const ciphertext = await wasmLoader.callFunction(
            'fheEncrypt',
            keys.publicKey,
            values[i]
          );
          const featureEndTime = performance.now();
          console.log(`  Feature ${i+1}/${values.length}: ${(featureEndTime - featureStartTime).toFixed(2)}ms`);
          
          encryptedFeatures.push(ciphertext as Uint8Array);
          
          if (onProgress) {
            onProgress(i + 1, values.length);
          }
        }
        
        const endTime = performance.now();
        const totalTime = endTime - startTime;
        
        console.log(`✅ Encrypted ${values.length} features in ${totalTime.toFixed(2)}ms`);
        console.log(`⚡ Average: ${(totalTime / values.length).toFixed(2)}ms per feature`);
        
        setTimings((prev) => ({
          ...prev,
          encryptionTime: totalTime,
        }));
        
        return encryptedFeatures;
      } catch (err: any) {
        console.error('Multiple encryption failed:', err);
        throw new Error(`Multiple encryption failed: ${err.message}`);
      }
    },
    [wasmLoaded, keys.publicKey, updateActivity, encrypt]
  );

  const decrypt = useCallback(
    async (ciphertext: Uint8Array): Promise<number> => {
      if (!wasmLoaded || !keys.secretKey) {
        throw new Error('FHE system not ready');
      }

      updateActivity(); // Activity 갱신

      // sk를 로컬 변수로 복사 (사용 후 명시적 제거 준비)
      const secretKeyForDecryption = keys.secretKey;

      try {
        const startTime = performance.now();

        const plaintext = await wasmLoader.callFunction(
          'fheDecrypt',
          secretKeyForDecryption,
          ciphertext
        );

        const endTime = performance.now();
        const decryptTime = endTime - startTime;

        console.log(`⏱️ Decryption time: ${decryptTime.toFixed(2)}ms`);
        console.log(`🔍 Raw decrypted value: ${plaintext}`);

        // Clamp to valid probability range [0, 1]
        // Backend should return sigmoid output in [0, 1], but clamp for safety
        let clampedValue = plaintext as number;
        if (clampedValue < 0 || clampedValue > 1) {
          console.warn(`⚠️ Decrypted value ${clampedValue} out of range [0,1], clamping...`);
          clampedValue = Math.max(0, Math.min(1, clampedValue));
        }

        console.log(`✅ Final score (probability): ${clampedValue.toFixed(6)}`);

        setTimings((prev) => ({ ...prev, decryptionTime: decryptTime }));

        return clampedValue;
      } catch (err: any) {
        console.error('Decryption failed:', err);
        throw new Error(`Decryption failed: ${err.message}`);
      } finally {
        // sk 메모리 즉시 폐기 (Best effort - GC는 JS 엔진이 결정)
        // TypedArray의 경우 버퍼를 0으로 덮어쓰기
        if (secretKeyForDecryption && secretKeyForDecryption.fill) {
          secretKeyForDecryption.fill(0);
        }
        console.log('🔒 Secret key reference cleared from memory');
      }
    },
    [wasmLoaded, keys.secretKey, updateActivity]
  );

  const sendToBackend = useCallback(
    async (encryptedFeatures: Uint8Array[]): Promise<Uint8Array> => {
      try {
        if (!keys.relinearizationKey) {
          throw new Error('Relinearization key not available');
        }

        console.log('🔄 Sending to backend with RLK for inference...');
        const startTime = performance.now();

        const result = await creditAPI.computeScore(encryptedFeatures, keys.relinearizationKey);

        const endTime = performance.now();
        const totalRoundTrip = endTime - startTime;
        const serverInferenceTime = result.serverInferenceTime;
        const networkTime = totalRoundTrip - serverInferenceTime;

        console.log(`⏱️ Total round-trip: ${totalRoundTrip.toFixed(2)}ms`);
        console.log(`⏱️ Server inference time: ${serverInferenceTime.toFixed(2)}ms`);
        console.log(`⏱️ Network time: ${networkTime.toFixed(2)}ms`);

        setTimings((prev) => ({ ...prev, inferenceTime: serverInferenceTime, networkTime }));

        return result.encryptedScore;
      } catch (err: any) {
        console.error('Backend communication failed:', err);
        throw new Error(`Backend error: ${err.message}`);
      }
    },
    [keys.relinearizationKey]
  );

  const value: FHEContextType = {
    wasmLoaded,
    isReady: wasmLoaded && keys.publicKey !== null && keys.secretKey !== null && keys.relinearizationKey !== null,
    loading,
    error,
    hasStoredKeys,
    timings,
    encrypt,
    encryptMultiple,
    decrypt,
    sendToBackend,
    generateKeys,
    loadStoredKeys,
    deleteStoredKeys,
    supportsParallelEncryption,
  };

  return <FHEContext.Provider value={value}>{children}</FHEContext.Provider>;
};
