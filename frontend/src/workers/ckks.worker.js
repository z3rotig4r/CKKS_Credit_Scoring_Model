// Web Worker for CKKS encryption operations
// This runs in a separate thread to enable parallel feature encryption

let wasmModule = null;
let isInitialized = false;

// Initialize WASM module when worker starts
self.addEventListener('message', async (e) => {
  const { cmd, data, id } = e.data;

  try {
    // Lazy load WASM module on first use
    if (!isInitialized && cmd !== 'ping') {
      await initializeWASM();
    }

    switch (cmd) {
      case 'ping':
        // Health check
        self.postMessage({ id, cmd: 'pong', success: true });
        break;

      case 'init':
        // Initialize WASM with specific config
        await initializeWASM();
        self.postMessage({ id, cmd: 'init', success: true });
        break;

      case 'encrypt':
        // Encrypt a single feature value
        const { value, publicKey } = data;
        const encrypted = await encryptValue(value, publicKey);
        self.postMessage({ id, cmd: 'encrypt', result: encrypted, success: true });
        break;

      case 'decrypt':
        // Decrypt a ciphertext
        const { ciphertext, secretKey } = data;
        const decrypted = await decryptValue(ciphertext, secretKey);
        self.postMessage({ id, cmd: 'decrypt', result: decrypted, success: true });
        break;

      case 'terminate':
        // Clean up and close worker
        cleanup();
        self.postMessage({ id, cmd: 'terminate', success: true });
        self.close();
        break;

      default:
        throw new Error(`Unknown command: ${cmd}`);
    }
  } catch (error) {
    self.postMessage({
      id,
      cmd,
      success: false,
      error: error.message || 'Unknown error in worker'
    });
  }
});

/**
 * Initialize WASM module
 * Loads wasm_exec.js and the Go WASM binary
 */
async function initializeWASM() {
  if (isInitialized) return;

  try {
    // Load Go WASM runtime
    importScripts('/wasm_exec.js');
    
    // Initialize Go runtime
    const go = new Go();
    const wasmPath = '/ckks.wasm';
    
    // Fetch and instantiate WASM module
    const result = await WebAssembly.instantiateStreaming(
      fetch(wasmPath),
      go.importObject
    );
    
    // Run Go WASM
    go.run(result.instance);
    
    // Wait for Go to expose functions
    await new Promise((resolve) => {
      const checkReady = setInterval(() => {
        if (self.encryptValue && self.decryptValue && self.generateKeys) {
          clearInterval(checkReady);
          resolve();
        }
      }, 50);
    });
    
    wasmModule = self;
    isInitialized = true;
    
    console.log('[Worker] WASM initialized successfully');
  } catch (error) {
    console.error('[Worker] WASM initialization failed:', error);
    throw new Error(`WASM init failed: ${error.message}`);
  }
}

/**
 * Encrypt a single value
 * @param {number} value - The plaintext value to encrypt
 * @param {string} publicKey - Base64 encoded public key
 * @returns {Promise<string>} Base64 encoded ciphertext
 */
async function encryptValue(value, publicKey) {
  if (!wasmModule || !wasmModule.encryptValue) {
    throw new Error('WASM module not initialized');
  }
  
  try {
    const encrypted = wasmModule.encryptValue(value, publicKey);
    return encrypted;
  } catch (error) {
    throw new Error(`Encryption failed: ${error.message}`);
  }
}

/**
 * Decrypt a ciphertext
 * @param {string} ciphertext - Base64 encoded ciphertext
 * @param {string} secretKey - Base64 encoded secret key
 * @returns {Promise<number>} Decrypted value
 */
async function decryptValue(ciphertext, secretKey) {
  if (!wasmModule || !wasmModule.decryptValue) {
    throw new Error('WASM module not initialized');
  }
  
  try {
    const decrypted = wasmModule.decryptValue(ciphertext, secretKey);
    return decrypted;
  } catch (error) {
    throw new Error(`Decryption failed: ${error.message}`);
  }
}

/**
 * Cleanup resources before terminating worker
 */
function cleanup() {
  wasmModule = null;
  isInitialized = false;
  console.log('[Worker] Cleaned up resources');
}
