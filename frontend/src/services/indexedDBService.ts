/**
 * IndexedDB Service for secure key storage with password-based encryption
 * Stores FHE keys encrypted with user password using Web Crypto API
 */

const DB_NAME = 'CKKS_Credit_DB';
const DB_VERSION = 1;
const STORE_NAME = 'fhe_keys';
const SALT_LENGTH = 16;
const IV_LENGTH = 12;
const KEY_LENGTH = 256;

interface KeyData {
  id: string;
  encryptedKey: ArrayBuffer;
  salt: ArrayBuffer;
  iv: ArrayBuffer;
  timestamp: number;
}

export interface StoredKeys {
  secretKey: Uint8Array;
  publicKey: Uint8Array;
  relinearizationKey?: Uint8Array;
  relinKey?: Uint8Array; // Deprecated alias
  galoisKeys?: Uint8Array;
}

class IndexedDBService {
  private db: IDBDatabase | null = null;

  /**
   * Initialize IndexedDB
   */
  async init(): Promise<void> {
    return new Promise((resolve, reject) => {
      const request = indexedDB.open(DB_NAME, DB_VERSION);

      request.onerror = () => reject(new Error('Failed to open IndexedDB'));

      request.onsuccess = () => {
        this.db = request.result;
        resolve();
      };

      request.onupgradeneeded = (event: IDBVersionChangeEvent) => {
        const db = (event.target as IDBOpenDBRequest).result;
        
        if (!db.objectStoreNames.contains(STORE_NAME)) {
          const store = db.createObjectStore(STORE_NAME, { keyPath: 'id' });
          store.createIndex('timestamp', 'timestamp', { unique: false });
        }
      };
    });
  }

  /**
   * Derive encryption key from password using PBKDF2
   */
  private async deriveKey(password: string, salt: Uint8Array): Promise<CryptoKey> {
    const encoder = new TextEncoder();
    const passwordBuffer = encoder.encode(password);

    const baseKey = await window.crypto.subtle.importKey(
      'raw',
      passwordBuffer,
      'PBKDF2',
      false,
      ['deriveBits', 'deriveKey']
    );

    return window.crypto.subtle.deriveKey(
      {
        name: 'PBKDF2',
        salt: salt as BufferSource,
        iterations: 100000,
        hash: 'SHA-256'
      },
      baseKey,
      { name: 'AES-GCM', length: KEY_LENGTH },
      false,
      ['encrypt', 'decrypt']
    );
  }

  /**
   * Encrypt data with password
   */
  private async encryptData(data: Uint8Array, password: string): Promise<{
    encryptedData: ArrayBuffer;
    salt: Uint8Array;
    iv: Uint8Array;
  }> {
    const salt = window.crypto.getRandomValues(new Uint8Array(SALT_LENGTH));
    const iv = window.crypto.getRandomValues(new Uint8Array(IV_LENGTH));

    const key = await this.deriveKey(password, salt);

    const encryptedData = await window.crypto.subtle.encrypt(
      { name: 'AES-GCM', iv: iv as BufferSource },
      key,
      data as BufferSource
    );

    return { encryptedData, salt, iv };
  }

  /**
   * Decrypt data with password
   */
  private async decryptData(
    encryptedData: ArrayBuffer,
    password: string,
    salt: Uint8Array,
    iv: Uint8Array
  ): Promise<Uint8Array> {
    const key = await this.deriveKey(password, salt);

    try {
      const decryptedData = await window.crypto.subtle.decrypt(
        { name: 'AES-GCM', iv: iv as BufferSource },
        key,
        encryptedData
      );

      return new Uint8Array(decryptedData);
    } catch (error) {
      throw new Error('Decryption failed. Wrong password?');
    }
  }

  /**
   * Save keys to IndexedDB with password encryption
   */
  async saveKeys(keys: StoredKeys, password: string): Promise<void> {
    if (!this.db) {
      await this.init();
    }

    // Serialize keys to JSON
    const keysJson = JSON.stringify({
      secretKey: Array.from(keys.secretKey),
      publicKey: Array.from(keys.publicKey),
      relinKey: keys.relinKey ? Array.from(keys.relinKey) : null,
      galoisKeys: keys.galoisKeys ? Array.from(keys.galoisKeys) : null
    });

    const keysBuffer = new TextEncoder().encode(keysJson);

    // Encrypt
    const { encryptedData, salt, iv } = await this.encryptData(
      keysBuffer,
      password
    );

    const keyData: KeyData = {
      id: 'fhe_keys',
      encryptedKey: encryptedData,
      salt: salt.buffer as ArrayBuffer,
      iv: iv.buffer as ArrayBuffer,
      timestamp: Date.now()
    };

    return new Promise((resolve, reject) => {
      const transaction = this.db!.transaction([STORE_NAME], 'readwrite');
      const store = transaction.objectStore(STORE_NAME);
      const request = store.put(keyData);

      request.onsuccess = () => {
        console.log('✅ Keys saved to IndexedDB (encrypted)');
        resolve();
      };

      request.onerror = () => {
        reject(new Error('Failed to save keys to IndexedDB'));
      };
    });
  }

  /**
   * Load keys from IndexedDB with password decryption
   */
  async loadKeys(password: string): Promise<StoredKeys | null> {
    if (!this.db) {
      await this.init();
    }

    return new Promise((resolve, reject) => {
      const transaction = this.db!.transaction([STORE_NAME], 'readonly');
      const store = transaction.objectStore(STORE_NAME);
      const request = store.get('fhe_keys');

      request.onsuccess = async () => {
        const keyData = request.result as KeyData | undefined;

        if (!keyData) {
          resolve(null);
          return;
        }

        try {
          // Decrypt
          const decryptedBuffer = await this.decryptData(
            keyData.encryptedKey,
            password,
            new Uint8Array(keyData.salt),
            new Uint8Array(keyData.iv)
          );

          const keysJson = new TextDecoder().decode(decryptedBuffer);
          const keysData = JSON.parse(keysJson);

          const keys: StoredKeys = {
            secretKey: new Uint8Array(keysData.secretKey),
            publicKey: new Uint8Array(keysData.publicKey),
            relinKey: keysData.relinKey
              ? new Uint8Array(keysData.relinKey)
              : undefined,
            galoisKeys: keysData.galoisKeys
              ? new Uint8Array(keysData.galoisKeys)
              : undefined
          };

          console.log('✅ Keys loaded from IndexedDB (decrypted)');
          resolve(keys);
        } catch (error) {
          reject(error);
        }
      };

      request.onerror = () => {
        reject(new Error('Failed to load keys from IndexedDB'));
      };
    });
  }

  /**
   * Check if keys exist in IndexedDB
   */
  async hasKeys(): Promise<boolean> {
    if (!this.db) {
      await this.init();
    }

    return new Promise((resolve, reject) => {
      const transaction = this.db!.transaction([STORE_NAME], 'readonly');
      const store = transaction.objectStore(STORE_NAME);
      const request = store.get('fhe_keys');

      request.onsuccess = () => {
        resolve(!!request.result);
      };

      request.onerror = () => {
        reject(new Error('Failed to check keys in IndexedDB'));
      };
    });
  }

  /**
   * Delete keys from IndexedDB
   */
  async deleteKeys(): Promise<void> {
    if (!this.db) {
      await this.init();
    }

    return new Promise((resolve, reject) => {
      const transaction = this.db!.transaction([STORE_NAME], 'readwrite');
      const store = transaction.objectStore(STORE_NAME);
      const request = store.delete('fhe_keys');

      request.onsuccess = () => {
        console.log('🗑️ Keys deleted from IndexedDB');
        resolve();
      };

      request.onerror = () => {
        reject(new Error('Failed to delete keys from IndexedDB'));
      };
    });
  }

  /**
   * Close database connection
   */
  close(): void {
    if (this.db) {
      this.db.close();
      this.db = null;
    }
  }
}

export const indexedDBService = new IndexedDBService();
