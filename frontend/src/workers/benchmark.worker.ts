/**
 * Web Worker Performance Benchmark
 * Tests real-world performance of parallel vs sequential encryption
 */

/* eslint-disable no-restricted-globals */
import wasmLoader from '../services/wasmLoader';

let wasmModule: any = null;
let publicKey: Uint8Array | null = null;

self.onmessage = async (e: MessageEvent) => {
  const { cmd, data } = e.data;

  try {
    switch (cmd) {
      case 'init':
        // Initialize WASM module
        await wasmLoader.load();
        wasmModule = wasmLoader;
        self.postMessage({ cmd: 'init', success: true });
        break;

      case 'setPublicKey':
        publicKey = data.publicKey;
        self.postMessage({ cmd: 'setPublicKey', success: true });
        break;

      case 'encrypt':
        if (!publicKey) {
          throw new Error('Public key not set');
        }
        const { value, index } = data;
        const startTime = performance.now();
        const encrypted = await wasmModule.callFunction('fheEncrypt', publicKey, value);
        const endTime = performance.now();
        
        self.postMessage({
          cmd: 'encrypt',
          result: encrypted,
          index,
          time: endTime - startTime
        });
        break;

      default:
        throw new Error(`Unknown command: ${cmd}`);
    }
  } catch (error: any) {
    self.postMessage({
      cmd: 'error',
      error: error.message || String(error)
    });
  }
};
