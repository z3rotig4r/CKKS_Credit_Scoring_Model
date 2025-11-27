/**
 * Parallel Encryption Service using Web Workers
 * 
 * Provides parallel encryption of credit scoring features for 5x speedup.
 * Falls back to sequential encryption if Web Workers unavailable.
 */

import { getWorkerPool } from './workerPool';

/**
 * Check if Web Workers are supported
 */
export function isWebWorkerSupported() {
  return typeof Worker !== 'undefined';
}

/**
 * Encrypt multiple features in parallel using Web Workers
 * 
 * @param {number[]} features - Array of feature values to encrypt
 * @param {string} publicKey - Base64 encoded public key
 * @param {Function} onProgress - Optional progress callback (featureIndex, total)
 * @returns {Promise<string[]>} Array of encrypted features (base64)
 */
export async function encryptFeaturesParallel(features, publicKey, onProgress = null) {
  if (!isWebWorkerSupported()) {
    console.warn('[ParallelEncryption] Web Workers not supported, falling back to sequential');
    return encryptFeaturesSequential(features, publicKey, onProgress);
  }

  try {
    const startTime = performance.now();
    const workerPool = getWorkerPool();
    
    console.log(`[ParallelEncryption] Encrypting ${features.length} features in parallel`);
    
    // Create encryption tasks for all features
    const encryptionPromises = features.map((value, index) => 
      workerPool.execute('encrypt', { value, publicKey })
        .then(encrypted => {
          if (onProgress) {
            onProgress(index + 1, features.length);
          }
          return encrypted;
        })
    );
    
    // Wait for all encryptions to complete
    const encryptedFeatures = await Promise.all(encryptionPromises);
    
    const elapsedTime = performance.now() - startTime;
    console.log(`[ParallelEncryption] Completed in ${elapsedTime.toFixed(2)}ms`);
    console.log(`[ParallelEncryption] Average: ${(elapsedTime / features.length).toFixed(2)}ms per feature`);
    
    return encryptedFeatures;
    
  } catch (error) {
    console.error('[ParallelEncryption] Error:', error);
    console.warn('[ParallelEncryption] Falling back to sequential encryption');
    return encryptFeaturesSequential(features, publicKey, onProgress);
  }
}

/**
 * Encrypt features sequentially (fallback)
 * 
 * @param {number[]} features - Array of feature values
 * @param {string} publicKey - Base64 encoded public key
 * @param {Function} onProgress - Optional progress callback
 * @returns {Promise<string[]>} Array of encrypted features
 */
async function encryptFeaturesSequential(features, publicKey, onProgress = null) {
  const startTime = performance.now();
  const encryptedFeatures = [];
  
  console.log(`[SequentialEncryption] Encrypting ${features.length} features sequentially`);
  
  // Import WASM module (assuming it's available globally)
  const { encryptValue } = await import('./wasmLoader');
  
  for (let i = 0; i < features.length; i++) {
    const encrypted = await encryptValue(features[i], publicKey);
    encryptedFeatures.push(encrypted);
    
    if (onProgress) {
      onProgress(i + 1, features.length);
    }
  }
  
  const elapsedTime = performance.now() - startTime;
  console.log(`[SequentialEncryption] Completed in ${elapsedTime.toFixed(2)}ms`);
  console.log(`[SequentialEncryption] Average: ${(elapsedTime / features.length).toFixed(2)}ms per feature`);
  
  return encryptedFeatures;
}

/**
 * Decrypt a value using Web Worker
 * 
 * @param {string} ciphertext - Base64 encoded ciphertext
 * @param {string} secretKey - Base64 encoded secret key
 * @returns {Promise<number>} Decrypted value
 */
export async function decryptValueParallel(ciphertext, secretKey) {
  if (!isWebWorkerSupported()) {
    const { decryptValue } = await import('./wasmLoader');
    return decryptValue(ciphertext, secretKey);
  }

  try {
    const workerPool = getWorkerPool();
    return await workerPool.execute('decrypt', { ciphertext, secretKey });
  } catch (error) {
    console.error('[ParallelDecryption] Error:', error);
    const { decryptValue } = await import('./wasmLoader');
    return decryptValue(ciphertext, secretKey);
  }
}

/**
 * Benchmark parallel vs sequential encryption
 * 
 * @param {number[]} testFeatures - Features to use for benchmarking
 * @param {string} publicKey - Public key for encryption
 * @returns {Promise<Object>} Benchmark results
 */
export async function benchmarkEncryption(testFeatures, publicKey) {
  console.log('[Benchmark] Starting encryption benchmark...');
  
  // Warm up (exclude from timing)
  await encryptFeaturesSequential([testFeatures[0]], publicKey);
  if (isWebWorkerSupported()) {
    await encryptFeaturesParallel([testFeatures[0]], publicKey);
  }
  
  // Sequential benchmark
  const seqStart = performance.now();
  await encryptFeaturesSequential(testFeatures, publicKey);
  const seqTime = performance.now() - seqStart;
  
  let parTime = null;
  let speedup = null;
  
  // Parallel benchmark (if supported)
  if (isWebWorkerSupported()) {
    const parStart = performance.now();
    await encryptFeaturesParallel(testFeatures, publicKey);
    parTime = performance.now() - parStart;
    speedup = seqTime / parTime;
  }
  
  const results = {
    supported: isWebWorkerSupported(),
    featureCount: testFeatures.length,
    sequential: {
      totalTime: seqTime,
      avgPerFeature: seqTime / testFeatures.length
    },
    parallel: parTime ? {
      totalTime: parTime,
      avgPerFeature: parTime / testFeatures.length,
      speedup: speedup
    } : null
  };
  
  console.log('[Benchmark] Results:', results);
  return results;
}
