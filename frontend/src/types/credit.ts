export interface CreditInputData {
  age: number;
  income: number;
  creditHistory: number; // months
  loanAmount: number;
  employmentYears: number;
  debtToIncome: number;
}

export interface EncryptedCreditData {
  encryptedFeatures: Uint8Array[];
  timestamp: number;
}

export interface CreditScoreResponse {
  encryptedScore: Uint8Array;
  timestamp: number;
}
