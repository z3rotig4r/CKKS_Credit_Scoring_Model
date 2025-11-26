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
 * Default credit scoring features
 * Can be easily modified to add/remove features
 */
export const CREDIT_FEATURES: FeatureConfig[] = [
  {
    name: 'age',
    label: 'Age',
    type: 'number',
    min: 18,
    max: 100,
    step: 1,
    defaultValue: 30,
    preprocessing: {
      method: 'normalize',
      params: { scale: 100 }, // age / 100
    },
    description: 'Applicant age in years',
  },
  {
    name: 'income',
    label: 'Annual Income',
    type: 'number',
    min: 0,
    max: 1000000,
    step: 1000,
    defaultValue: 50000,
    preprocessing: {
      method: 'normalize',
      params: { scale: 100000 }, // income / 100000
    },
    description: 'Annual income in dollars',
  },
  {
    name: 'creditHistory',
    label: 'Credit History (months)',
    type: 'number',
    min: 0,
    max: 600,
    step: 1,
    defaultValue: 60,
    preprocessing: {
      method: 'normalize',
      params: { scale: 120 }, // creditHistory / 120 (10 years)
    },
    description: 'Length of credit history in months',
  },
  {
    name: 'loanAmount',
    label: 'Requested Loan Amount',
    type: 'number',
    min: 0,
    max: 500000,
    step: 1000,
    defaultValue: 25000,
    preprocessing: {
      method: 'normalize',
      params: { scale: 50000 }, // loanAmount / 50000
    },
    description: 'Requested loan amount in dollars',
  },
  {
    name: 'employmentYears',
    label: 'Years of Employment',
    type: 'number',
    min: 0,
    max: 50,
    step: 1,
    defaultValue: 5,
    preprocessing: {
      method: 'normalize',
      params: { scale: 40 }, // employmentYears / 40
    },
    description: 'Years at current employment',
  },
  {
    name: 'debtToIncome',
    label: 'Debt-to-Income Ratio',
    type: 'number',
    min: 0,
    max: 1,
    step: 0.01,
    defaultValue: 0.3,
    preprocessing: {
      method: 'none', // Already in [0, 1] range
    },
    description: 'Ratio of monthly debt to monthly income',
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
 */
export function preprocessFeatures(
  rawData: Record<string, number>
): number[] {
  return CREDIT_FEATURES.map((feature) => {
    const rawValue = rawData[feature.name] || feature.defaultValue;
    const value = typeof rawValue === 'number' ? rawValue : parseFloat(String(rawValue));
    return preprocessFeature(value, feature.preprocessing);
  });
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
