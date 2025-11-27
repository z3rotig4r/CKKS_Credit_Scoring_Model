/**
 * Feature Configuration for Credit Scoring
 * Centralized configuration for input features with preprocessing specifications
 */

export interface FeatureConfig {
  name: string;
  label: string;
  type: 'number' | 'select' | 'checkbox';
  min?: number;
  max?: number;
  step?: number;
  defaultValue: number | string;
  options?: { value: string; label: string }[];
  preprocessing: PreprocessingConfig;
  description?: string;
  unit?: string;
}

export interface PreprocessingConfig {
  method: 'normalize' | 'standardize' | 'minmax' | 'log' | 'none';
  params?: {
    min?: number;
    max?: number;
    mean?: number;
    std?: number;
    scale?: number;
  };
}

/**
 * Production credit scoring features (4 user inputs → 5 model features)
 * AUC-ROC: 0.5886, All coefficients CKKS-safe
 * NOTE: EXT_SOURCE_2 제거! 우리가 신용점수를 계산하는 시스템이므로!
 * 
 * User provides 4 inputs:
 * 1. age: Age in years
 * 2. loanAmount: Requested loan amount (KRW)
 * 3. income: Annual income (KRW)
 * 4. monthlyPayment: Expected monthly payment (KRW)
 * 
 * Backend model expects 5 features:
 * [age/10, loan_to_income, debt_to_income, credit_amount, income/100000]
 */
export const CREDIT_FEATURES: FeatureConfig[] = [
  {
    name: 'age',
    label: 'Age (years)',
    type: 'number',
    min: 18,
    max: 70,
    step: 1,
    defaultValue: 40,
    unit: 'years',
    preprocessing: {
      method: 'normalize',
      params: { scale: 10 }, // age / 10 → 결과: 1.8 ~ 7.0
    },
    description: 'Applicant age in years',
  },
  {
    name: 'loanAmount',
    label: 'Requested Loan Amount (KRW)',
    type: 'number',
    min: 1000000,
    max: 50000000,
    step: 1000000,
    defaultValue: 10000000,
    unit: 'KRW',
    preprocessing: {
      method: 'none', // Will be used in ratio calculation
    },
    description: 'Loan amount requested (used for ratio calculation)',
  },
  {
    name: 'income',
    label: 'Annual Income (KRW)',
    type: 'number',
    min: 10000000,
    max: 200000000,
    step: 1000000,
    defaultValue: 50000000,
    unit: 'KRW',
    preprocessing: {
      method: 'normalize',
      params: { scale: 100000 }, // income / 100000 → 결과: 100 ~ 2000
    },
    description: 'Annual income in KRW',
  },
  {
    name: 'monthlyPayment',
    label: 'Expected Monthly Payment (KRW)',
    type: 'number',
    min: 100000,
    max: 10000000,
    step: 100000,
    defaultValue: 1000000,
    unit: 'KRW',
    preprocessing: {
      method: 'none', // Will be used in debt_to_income calculation
    },
    description: 'Expected monthly loan payment',
  },
];

/**
 * Preprocess a single feature value
 */
export function preprocessFeature(
  value: number,
  config: PreprocessingConfig
): number {
  switch (config.method) {
    case 'normalize':
      // x / scale
      return value / (config.params?.scale || 1);

    case 'standardize':
      // (x - mean) / std
      const mean = config.params?.mean || 0;
      const std = config.params?.std || 1;
      return (value - mean) / std;

    case 'minmax':
      // (x - min) / (max - min)
      const min = config.params?.min || 0;
      const max = config.params?.max || 1;
      return (value - min) / (max - min);

    case 'log':
      // log(1 + x)
      return Math.log1p(value);

    case 'none':
    default:
      return value;
  }
}

/**
 * Preprocess all features based on their configurations
 * Returns 5 features matching backend model:
 * [age, loan_to_income, debt_to_income, credit_amount, income]
 * NOTE: EXT_SOURCE_2 제거! 우리가 신용점수를 계산하는 시스템이므로!
 */
export function preprocessFeatures(
  rawData: Record<string, number>
): number[] {
  // Extract raw values (only 4 inputs from user)
  const age = rawData['age'] || 40;
  const loanAmount = rawData['loanAmount'] || 10000000;
  const income = rawData['income'] || 50000000;
  const monthlyPayment = rawData['monthlyPayment'] || 1000000;
  
  // Feature 1: age / 10
  const f1_age = age / 10;
  
  // Feature 2: loan_to_income ratio (clip to 0-10)
  const f2_loan_to_income = Math.min(10, Math.max(0, loanAmount / income));
  
  // Feature 3: debt_to_income ratio * 100 (clip to 0-100)
  // Monthly payment / Monthly income * 100
  const monthlyIncome = income / 12;
  const f3_debt_to_income = Math.min(100, Math.max(0, (monthlyPayment / monthlyIncome) * 100));
  
  // Feature 4: credit_amount / 100000
  const f4_credit_amount = loanAmount / 100000;
  
  // Feature 5: income / 100000
  const f5_income = income / 100000;
  
  return [f1_age, f2_loan_to_income, f3_debt_to_income, f4_credit_amount, f5_income];
}

/**
 * Get feature names in order
 */
export function getFeatureNames(): string[] {
  return CREDIT_FEATURES.map((f) => f.name);
}

/**
 * Get feature labels for display
 */
export function getFeatureLabels(): string[] {
  return CREDIT_FEATURES.map((f) => f.label);
}

/**
 * Validate feature value against config
 */
export function validateFeature(
  featureName: string,
  value: number
): { valid: boolean; error?: string } {
  const config = CREDIT_FEATURES.find((f) => f.name === featureName);
  if (!config) {
    return { valid: false, error: 'Unknown feature' };
  }

  if (config.min !== undefined && value < config.min) {
    return { valid: false, error: `Value must be >= ${config.min}` };
  }

  if (config.max !== undefined && value > config.max) {
    return { valid: false, error: `Value must be <= ${config.max}` };
  }

  return { valid: true };
}
