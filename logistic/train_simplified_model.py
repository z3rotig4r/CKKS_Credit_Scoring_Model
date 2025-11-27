"""
Simplified Production Model - Use only strong features
Keep coefficient magnitudes in CKKS-safe range (0.01 ~ 1.0)
"""

import pandas as pd
import numpy as np
from sklearn.linear_model import LogisticRegression
from sklearn.model_selection import train_test_split
from sklearn.metrics import roc_auc_score
import json

def preprocess_features(df):
    """
    Use only 5 strong features - NO external score!
    We are CALCULATING credit score, not using it as input.
    """
    X = pd.DataFrame()
    
    # 1. Age: DAYS_BIRTH → years, normalize to ~0-10 range
    if 'DAYS_BIRTH' in df.columns:
        age_years = -df['DAYS_BIRTH'] / 365.25
        X['age'] = age_years / 10.0  # 0-10 scale
    
    # 2. Loan/Income Ratio: More predictive than separate values
    if 'AMT_CREDIT' in df.columns and 'AMT_INCOME_TOTAL' in df.columns:
        X['loan_to_income'] = (df['AMT_CREDIT'] / df['AMT_INCOME_TOTAL']).clip(0, 10)
    
    # 3. Debt/Income Ratio: Financial burden indicator
    if 'AMT_ANNUITY' in df.columns and 'AMT_INCOME_TOTAL' in df.columns:
        X['debt_to_income'] = (df['AMT_ANNUITY'] / df['AMT_INCOME_TOTAL'] * 100).clip(0, 100)
    
    # 4. Credit Amount (scaled)
    if 'AMT_CREDIT' in df.columns:
        X['credit_amount'] = df['AMT_CREDIT'] / 100000  # 10만원 단위
    
    # 5. Income (scaled)
    if 'AMT_INCOME_TOTAL' in df.columns:
        X['income'] = df['AMT_INCOME_TOTAL'] / 100000  # 10만원 단위
    
    # NOTE: EXT_SOURCE_2 제거! 우리가 신용점수를 계산하는 시스템이므로!
    
    return X

def train_model(csv_path='dataset/application_train.csv'):
    """Train simplified model with strong features"""
    
    print("=" * 70)
    print("Training Simplified Production Model (CKKS-Optimized)")
    print("=" * 70)
    
    # Load data
    print("\n1. Loading data...")
    df = pd.read_csv(csv_path)
    print(f"   Loaded {len(df):,} rows")
    
    # Check available columns
    print("\n2. Available columns:")
    important_cols = ['DAYS_BIRTH', 'AMT_CREDIT', 'AMT_INCOME_TOTAL', 
                     'AMT_ANNUITY', 'TARGET']
    for col in important_cols:
        if col in df.columns:
            print(f"   ✅ {col}")
        else:
            print(f"   ❌ {col} (missing)")
    
    print("\n   ⚠️ EXT_SOURCE_2 intentionally excluded (we calculate score, not use it!)")
    
    # Preprocess
    print("\n3. Preprocessing features...")
    X = preprocess_features(df)
    y = df['TARGET']
    
    print(f"\n   Features created:")
    for col in X.columns:
        print(f"   - {col}")
    
    # Remove NaN
    mask = ~(X.isna().any(axis=1) | y.isna())
    X_clean = X[mask]
    y_clean = y[mask]
    print(f"\n   After removing NaN: {len(X_clean):,} rows ({len(X_clean)/len(df)*100:.1f}%)")
    
    # Feature statistics
    print("\n4. Feature statistics:")
    print(X_clean.describe())
    
    # Train-test split
    print("\n5. Splitting data...")
    X_train, X_test, y_train, y_test = train_test_split(
        X_clean, y_clean, test_size=0.2, random_state=42, stratify=y_clean
    )
    
    # Train
    print("\n6. Training logistic regression...")
    model = LogisticRegression(max_iter=1000, random_state=42)
    model.fit(X_train, y_train)
    
    # Evaluate
    print("\n7. Evaluating...")
    y_proba = model.predict_proba(X_test)[:, 1]
    auc = roc_auc_score(y_test, y_proba)
    print(f"   AUC-ROC: {auc:.4f}")
    
    # Coefficients
    print("\n8. Model Coefficients (CKKS Safety Check):")
    print("-" * 70)
    
    feature_names = X_clean.columns.tolist()
    safe_count = 0
    
    for name, coef in zip(feature_names, model.coef_[0]):
        magnitude = abs(coef)
        
        if 0.01 <= magnitude <= 1.0:
            status = "✅ SAFE"
            safe_count += 1
        elif 0.001 <= magnitude < 0.01:
            status = "⚠️ OK"
        elif magnitude < 0.001:
            status = "❌ TOO SMALL"
        else:
            status = "⚠️ TOO LARGE"
        
        print(f"   {name:20s}  {coef:12.8f}  (|{magnitude:.2e}|)  {status}")
    
    print(f"\n   Intercept:           {model.intercept_[0]:12.8f}")
    print("-" * 70)
    print(f"\n   CKKS Safety: {safe_count}/{len(feature_names)} coefficients in optimal range")
    
    # Save
    coefficients = {
        'weights': model.coef_[0].tolist(),
        'bias': float(model.intercept_[0]),
        'feature_names': feature_names,
        'auc_roc': float(auc),
        'num_features': len(feature_names),
        'safe_count': safe_count
    }
    
    output_file = 'logistic/model_coefficients.json'
    with open(output_file, 'w') as f:
        json.dump(coefficients, f, indent=2)
    print(f"\n✅ Coefficients saved to {output_file}")
    
    # Generate Go code
    print("\n9. Go Backend Code:")
    print("=" * 70)
    print("// Copy this to backend/main.go")
    print()
    print("var model = LogisticRegressionModel{")
    print("    Weights: []float64{")
    for coef in model.coef_[0]:
        print(f"        {coef:.10f},")
    print("    },")
    print(f"    Bias: {model.intercept_[0]:.10f},")
    print("}")
    print("=" * 70)
    
    # Generate frontend feature config
    print("\n10. Frontend Feature Config Update Needed:")
    print("=" * 70)
    print("// Update frontend/src/config/features.ts")
    print()
    print("export const CREDIT_FEATURES: FeatureConfig[] = [")
    
    feature_mappings = {
        'age': {'label': 'Age', 'scale': 10.0, 'unit': 'years'},
        'loan_to_income': {'label': 'Loan-to-Income Ratio', 'scale': 1.0, 'unit': 'ratio'},
        'debt_to_income': {'label': 'Debt-to-Income Ratio', 'scale': 100.0, 'unit': 'percent'},
        'credit_amount': {'label': 'Credit Amount', 'scale': 100000.0, 'unit': 'KRW'},
        'income': {'label': 'Annual Income', 'scale': 100000.0, 'unit': 'KRW'},
    }
    
    for feat in feature_names:
        if feat in feature_mappings:
            info = feature_mappings[feat]
            print(f"  // {feat}: {info['label']} / {info['scale']}")
    
    print("];")
    print("=" * 70)
    
    return model, feature_names, coefficients

if __name__ == "__main__":
    model, features, coeffs = train_model()
    
    print("\n" + "=" * 70)
    print("✅ Training Complete!")
    print("=" * 70)
    print("\nSummary:")
    print(f"  - Features: {len(features)}")
    print(f"  - AUC-ROC: {coeffs['auc_roc']:.4f}")
    print(f"  - CKKS-safe coefficients: {coeffs['safe_count']}/{len(features)}")
    print("\nNext steps:")
    print("  1. Update backend/main.go with new weights")
    print("  2. Update frontend/src/config/features.ts")
    print("  3. Test CKKS inference")
