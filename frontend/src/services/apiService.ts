// Backend URL - 프로덕션에서는 백엔드 서버 직접 연결 (Netlify 프록시는 body 크기 제한으로 사용 불가)
const API_BASE_URL = process.env.REACT_APP_API_URL || 'http://141.164.42.210:8080';

export interface InferenceRequest {
  encryptedFeatures: string[]; // Base64 encoded
  relinearizationKey: string; // Base64 encoded
}

export interface InferenceResponse {
  encryptedScore: string; // Base64 encoded
  timestamp: number;
  serverInferenceTime: number; // Server-side inference time in ms
}

export interface ComputeScoreResult {
  encryptedScore: Uint8Array;
  serverInferenceTime: number;
}

export const creditAPI = {
  async computeScore(encryptedFeatures: Uint8Array[], relinearizationKey: Uint8Array): Promise<ComputeScoreResult> {
    // Convert Uint8Array to base64
    const base64Features = encryptedFeatures.map(arr => {
      // Handle large arrays by chunking to avoid call stack overflow
      let binary = '';
      const chunkSize = 8192;
      for (let i = 0; i < arr.length; i += chunkSize) {
        const chunk = arr.subarray(i, Math.min(i + chunkSize, arr.length));
        binary += String.fromCharCode.apply(null, Array.from(chunk));
      }
      return btoa(binary);
    });

    // Convert RLK to base64
    let rlkBinary = '';
    const chunkSize = 8192;
    for (let i = 0; i < relinearizationKey.length; i += chunkSize) {
      const chunk = relinearizationKey.subarray(i, Math.min(i + chunkSize, relinearizationKey.length));
      rlkBinary += String.fromCharCode.apply(null, Array.from(chunk));
    }
    const base64RLK = btoa(rlkBinary);

    const response = await fetch(`${API_BASE_URL}/api/inference`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({ 
        encryptedFeatures: base64Features,
        relinearizationKey: base64RLK,
      }),
    });

    if (!response.ok) {
      let errorMsg = `Backend error (HTTP ${response.status})`;
      try {
        const errorData = await response.text();
        if (errorData) {
          errorMsg += `: ${errorData}`;
        }
      } catch {
        // Ignore parse errors
      }
      throw new Error(errorMsg);
    }

    const data: InferenceResponse = await response.json();
    
    // Convert base64 back to Uint8Array
    const binaryString = atob(data.encryptedScore);
    const bytes = new Uint8Array(binaryString.length);
    for (let i = 0; i < binaryString.length; i++) {
      bytes[i] = binaryString.charCodeAt(i);
    }
    
    return { encryptedScore: bytes, serverInferenceTime: data.serverInferenceTime };
  },

  async healthCheck(): Promise<boolean> {
    try {
      const response = await fetch(`${API_BASE_URL}/health`);
      return response.ok;
    } catch {
      return false;
    }
  }
};
