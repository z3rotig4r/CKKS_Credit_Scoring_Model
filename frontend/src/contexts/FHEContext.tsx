import React, { createContext, useContext, useState, useEffect, useCallback } from 'react';
import wasmLoader from '../services/wasmLoader';
import { creditAPI } from '../services/apiService';
import { indexedDBService, StoredKeys } from '../services/indexedDBService';

export interface TimingStats {
  keygenTime?: number;
  encryptionTime?: number;
  decryptionTime?: number;
  inferenceTime?: number;
}

interface FHEContextType {
  isReady: boolean;
  loading: boolean;
  error: string | null;
  hasStoredKeys: boolean;
  timings: TimingStats;
  encrypt: (value: number) => Promise<Uint8Array>;
  decrypt: (ciphertext: Uint8Array) => Promise<number>;
  sendToBackend: (encryptedFeatures: Uint8Array[]) => Promise<Uint8Array>;
  generateKeys: (password?: string) => Promise<void>;
  loadStoredKeys: (password: string) => Promise<void>;
  deleteStoredKeys: () => Promise<void>;
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
}

export const FHEProvider: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const [wasmLoaded, setWasmLoaded] = useState(false);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [keys, setKeys] = useState<Keys>({ secretKey: null, publicKey: null });
  const [hasStoredKeys, setHasStoredKeys] = useState(false);
  const [timings, setTimings] = useState<TimingStats>({});
  const [lastActivity, setLastActivity] = useState<number>(Date.now());

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
      const startTime = performance.now();

      const keyPair = await wasmLoader.callFunction('fheKeygen');

      const endTime = performance.now();
      const keygenTime = endTime - startTime;

      console.log(`⏱️ Key generation time: ${keygenTime.toFixed(2)}ms`);

      setKeys({
        secretKey: keyPair.secretKey,
        publicKey: keyPair.publicKey,
      });

      setTimings((prev) => ({ ...prev, keygenTime }));

      // Save to IndexedDB if password provided
      if (password) {
        await indexedDBService.saveKeys(
          {
            secretKey: keyPair.secretKey,
            publicKey: keyPair.publicKey,
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

        const ciphertext = await wasmLoader.callFunction(
          'fheEncrypt',
          keys.publicKey,
          value
        );

        const endTime = performance.now();
        const encryptTime = endTime - startTime;

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

        setTimings((prev) => ({ ...prev, decryptionTime: decryptTime }));

        return plaintext as number;
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
        console.log('🔄 Sending to backend for inference...');
        const startTime = performance.now();

        const encryptedScore = await creditAPI.computeScore(encryptedFeatures);

        const endTime = performance.now();
        const inferenceTime = endTime - startTime;

        console.log(`⏱️ Backend inference time: ${inferenceTime.toFixed(2)}ms`);

        setTimings((prev) => ({ ...prev, inferenceTime }));

        return encryptedScore;
      } catch (err: any) {
        console.error('Backend communication failed:', err);
        throw new Error(`Backend error: ${err.message}`);
      }
    },
    []
  );

  const value: FHEContextType = {
    isReady: wasmLoaded && keys.publicKey !== null && keys.secretKey !== null,
    loading,
    error,
    hasStoredKeys,
    timings,
    encrypt,
    decrypt,
    sendToBackend,
    generateKeys,
    loadStoredKeys,
    deleteStoredKeys,
  };

  return <FHEContext.Provider value={value}>{children}</FHEContext.Provider>;
};
