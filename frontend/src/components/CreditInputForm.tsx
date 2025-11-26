import React, { useState } from 'react';
import { useFHE } from '../contexts/FHEContext';
import {
  CREDIT_FEATURES,
  preprocessFeatures,
  validateFeature,
  FeatureConfig,
} from '../config/features';

interface FormData {
  [key: string]: number;
}

const CreditInputForm: React.FC = () => {
  const { encrypt, decrypt, sendToBackend, isReady, timings, generateKeys, loadStoredKeys, hasStoredKeys } = useFHE();

  const initialFormData: FormData = {};
  CREDIT_FEATURES.forEach((feature) => {
    initialFormData[feature.name] =
      typeof feature.defaultValue === 'number'
        ? feature.defaultValue
        : parseFloat(String(feature.defaultValue));
  });

  const [formData, setFormData] = useState<FormData>(initialFormData);
  const [result, setResult] = useState<number | null>(null);
  const [loading, setLoading] = useState(false);
  const [password, setPassword] = useState('');
  const [showPasswordPrompt, setShowPasswordPrompt] = useState(false);
  const [passwordAction, setPasswordAction] = useState<'generate' | 'load'>('generate');

  const handleInputChange = (name: string, value: string) => {
    const numValue = parseFloat(value) || 0;
    const validation = validateFeature(name, numValue);
    
    if (!validation.valid) {
      console.warn(name + ': ' + validation.error);
    }
    
    setFormData((prev) => ({ ...prev, [name]: numValue }));
  };

  const handlePasswordSubmit = async () => {
    if (!password) {
      alert('Please enter a password');
      return;
    }

    try {
      if (passwordAction === 'generate') {
        await generateKeys(password);
        alert('Keys generated successfully! 🎉');
      } else {
        await loadStoredKeys(password);
        alert('Keys loaded successfully! ✅');
      }
      setShowPasswordPrompt(false);
      setPassword('');
    } catch (error: any) {
      alert('Error: ' + error.message);
    }
  };

  const handleGenerateKeys = () => {
    setPasswordAction('generate');
    setShowPasswordPrompt(true);
  };

  const handleLoadKeys = () => {
    setPasswordAction('load');
    setShowPasswordPrompt(true);
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();

    if (!isReady) {
      alert('FHE system not ready. Please generate or load keys first.');
      return;
    }

    setLoading(true);
    setResult(null);

    try {
      const preprocessedFeatures = preprocessFeatures(formData);
      const encryptedFeatures = await Promise.all(
        preprocessedFeatures.map((f) => encrypt(f))
      );
      const encryptedScore = await sendToBackend(encryptedFeatures);
      const score = await decrypt(encryptedScore);
      setResult(score);
    } catch (error: any) {
      let userMessage = 'Failed to compute credit score';
      if (error.message?.includes('HTTP 400')) {
        userMessage = 'Invalid input data. Please check your entries.';
      } else if (error.message?.includes('HTTP 500')) {
        userMessage = 'Server error during computation. Please try again.';
      } else if (error.message?.includes('Failed to fetch')) {
        userMessage = 'Cannot connect to server. Please check if backend is running.';
      } else if (error.message) {
        userMessage += ': ' + error.message;
      }
      alert(userMessage);
    } finally {
      setLoading(false);
    }
  };

  const getCreditRating = (score: number) => {
    if (score >= 0.8) return { 
      label: 'Excellent', 
      color: 'bg-gradient-to-r from-green-500 to-emerald-500',
      textColor: 'text-green-700',
      bgColor: 'bg-green-50',
      icon: '★★★★★'
    };
    if (score >= 0.6) return { 
      label: 'Good', 
      color: 'bg-gradient-to-r from-primary to-accent',
      textColor: 'text-primary',
      bgColor: 'bg-blue-50',
      icon: '★★★★☆'
    };
    if (score >= 0.4) return { 
      label: 'Fair', 
      color: 'bg-gradient-to-r from-yellow-500 to-orange-500',
      textColor: 'text-yellow-700',
      bgColor: 'bg-yellow-50',
      icon: '★★★☆☆'
    };
    return { 
      label: 'Needs Improvement', 
      color: 'bg-gradient-to-r from-secondary to-red-500',
      textColor: 'text-red-700',
      bgColor: 'bg-red-50',
      icon: '★★☆☆☆'
    };
  };

  const renderFeatureInput = (feature: FeatureConfig) => {
    const value = formData[feature.name];
    const validation = validateFeature(feature.name, value);

    return (
      <div key={feature.name} className="group">
        <label className="block text-sm font-semibold text-cau-gray mb-2">
          {feature.label}
          {feature.unit && <span className="text-cau-light-gray font-normal ml-1.5">({feature.unit})</span>}
        </label>
        {feature.description && (
          <p className="text-xs text-cau-light-gray mb-2">{feature.description}</p>
        )}
        <input
          type="number"
          value={value}
          onChange={(e) => handleInputChange(feature.name, e.target.value)}
          step={feature.step || 1}
          min={feature.min}
          max={feature.max}
          className={`w-full px-4 py-3 bg-white border-2 rounded-xl text-dark font-medium
            focus:outline-none focus:ring-4 focus:ring-primary/20 focus:border-primary
            transition-all duration-200 hover:border-cau-light-gray
            ${!validation.valid ? 'border-secondary bg-red-50' : 'border-cau-light-gray'}`}
        />
        {!validation.valid && (
          <p className="text-xs text-secondary mt-1.5 flex items-center gap-1">
            <svg className="w-3.5 h-3.5" fill="currentColor" viewBox="0 0 20 20">
              <path fillRule="evenodd" d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7 4a1 1 0 11-2 0 1 1 0 012 0zm-1-9a1 1 0 00-1 1v4a1 1 0 102 0V6a1 1 0 00-1-1z" clipRule="evenodd" />
            </svg>
            {validation.error}
          </p>
        )}
        {feature.min !== undefined && feature.max !== undefined && (
          <p className="text-xs text-cau-light-gray mt-1">
            Range: {feature.min.toLocaleString()} - {feature.max.toLocaleString()}
          </p>
        )}
      </div>
    );
  };

  return (
    <div className="min-h-screen bg-gradient-to-br from-light via-white to-blue-50/30">
      {/* Modern Header */}
      <header className="bg-white/80 backdrop-blur-md border-b border-cau-light-gray/50 sticky top-0 z-40">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-4">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-3">
              <div className="w-10 h-10 bg-gradient-to-br from-primary to-accent rounded-xl flex items-center justify-center shadow-soft">
                <svg className="w-6 h-6 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z" />
                </svg>
              </div>
              <div>
                <h1 className="text-xl font-bold text-dark">Private Credit Scoring</h1>
                <p className="text-xs text-cau-gray">Homomorphic Encryption Powered by CKKS</p>
              </div>
            </div>
            
            <div className="flex items-center gap-3">
              <div className={`flex items-center gap-2 px-3 py-1.5 rounded-full text-xs font-semibold ${
                isReady ? 'bg-green-50 text-green-700' : 'bg-cau-light-gray/30 text-cau-gray'
              }`}>
                <div className={`w-2 h-2 rounded-full ${isReady ? 'bg-green-500 animate-pulse' : 'bg-cau-gray'}`} />
                {isReady ? 'System Ready' : 'Initializing...'}
              </div>
            </div>
          </div>
        </div>
      </header>

      <main className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        {/* Key Management Card */}
        <div className="bg-white rounded-2xl shadow-soft border border-cau-light-gray/50 p-6 mb-6">
          <div className="flex items-center justify-between mb-5">
            <div className="flex items-center gap-3">
              <div className="w-10 h-10 bg-gradient-to-br from-cau-gold to-yellow-600 rounded-xl flex items-center justify-center">
                <svg className="w-5 h-5 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 7a2 2 0 012 2m4 0a6 6 0 01-7.743 5.743L11 17H9v2H7v2H4a1 1 0 01-1-1v-2.586a1 1 0 01.293-.707l5.964-5.964A6 6 0 1121 9z" />
                </svg>
              </div>
              <div>
                <h2 className="text-lg font-bold text-dark">Key Management</h2>
                <p className="text-xs text-cau-gray">Secure your encryption keys with a password</p>
              </div>
            </div>
          </div>
          
          <div className="space-y-3">
            <div className="grid md:grid-cols-2 gap-3">
              <button
                onClick={handleGenerateKeys}
                disabled={loading}
                className="group relative px-6 py-4 bg-gradient-to-r from-primary to-accent text-white font-semibold rounded-xl
                  hover:shadow-medium transition-all duration-200 disabled:opacity-50 disabled:cursor-not-allowed
                  flex items-center justify-center gap-2"
              >
                <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
                </svg>
                Generate New Keys
              </button>
              
              <button
                onClick={handleLoadKeys}
                disabled={loading || !hasStoredKeys}
                className="px-6 py-4 bg-white border-2 border-primary text-primary font-semibold rounded-xl
                  hover:bg-primary hover:text-white transition-all duration-200 disabled:opacity-50 disabled:cursor-not-allowed
                  flex items-center justify-center gap-2"
              >
                <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
                </svg>
                Load Saved Keys
              </button>
            </div>
            
            {/* Clear Keys Warning */}
            {hasStoredKeys && (
              <div className="bg-yellow-50 border border-yellow-200 rounded-xl p-4">
                <div className="flex items-start gap-3">
                  <svg className="w-5 h-5 text-yellow-600 flex-shrink-0 mt-0.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
                  </svg>
                  <div className="flex-1">
                    <p className="text-sm font-semibold text-yellow-800 mb-1">Parameter Mismatch Error?</p>
                    <p className="text-xs text-yellow-700 mb-3">If you see "ring degree does not match" error, your saved keys were created with different parameters. Generate new keys to fix this.</p>
                    <button
                      onClick={async () => {
                        if (window.confirm('This will delete all saved keys. You will need to generate new keys. Continue?')) {
                          const { indexedDBService } = await import('../services/indexedDBService');
                          await indexedDBService.deleteKeys();
                          alert('Keys deleted! Please generate new keys.');
                          window.location.reload();
                        }
                      }}
                      className="text-xs px-3 py-1.5 bg-yellow-600 text-white font-semibold rounded-lg hover:bg-yellow-700 transition-all"
                    >
                      Clear Saved Keys
                    </button>
                  </div>
                </div>
              </div>
            )}
          </div>
        </div>

        {/* Password Modal */}
        {showPasswordPrompt && (
          <div className="fixed inset-0 bg-dark/60 backdrop-blur-sm flex items-center justify-center z-50 p-4">
            <div className="bg-white rounded-3xl shadow-hard max-w-md w-full p-8 transform transition-all">
              <div className="w-16 h-16 bg-gradient-to-br from-primary to-accent rounded-2xl flex items-center justify-center mx-auto mb-6">
                <svg className="w-8 h-8 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z" />
                </svg>
              </div>
              
              <h3 className="text-2xl font-bold text-dark text-center mb-2">
                {passwordAction === 'generate' ? 'Secure Your Keys' : 'Unlock Your Keys'}
              </h3>
              <p className="text-cau-gray text-center mb-6 text-sm">
                {passwordAction === 'generate'
                  ? 'Create a strong password to protect your encryption keys'
                  : 'Enter your password to access saved keys'}
              </p>
              
              <input
                type="password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                onKeyPress={(e) => e.key === 'Enter' && handlePasswordSubmit()}
                placeholder="Enter password"
                autoFocus
                className="w-full px-4 py-3 bg-light border-2 border-cau-light-gray rounded-xl text-dark font-medium
                  focus:outline-none focus:ring-4 focus:ring-primary/20 focus:border-primary transition-all mb-6"
              />
              
              <div className="flex gap-3">
                <button
                  onClick={() => {
                    setShowPasswordPrompt(false);
                    setPassword('');
                  }}
                  className="flex-1 px-4 py-3 bg-light text-cau-gray font-semibold rounded-xl hover:bg-cau-light-gray/30 transition-all"
                >
                  Cancel
                </button>
                <button
                  onClick={handlePasswordSubmit}
                  className="flex-1 px-4 py-3 bg-gradient-to-r from-primary to-accent text-white font-semibold rounded-xl hover:shadow-medium transition-all"
                >
                  {passwordAction === 'generate' ? 'Generate' : 'Unlock'}
                </button>
              </div>
            </div>
          </div>
        )}

        {/* Main Form */}
        <form onSubmit={handleSubmit} className="bg-white rounded-2xl shadow-soft border border-cau-light-gray/50 p-8 mb-6">
          <div className="flex items-center gap-3 mb-8">
            <div className="w-10 h-10 bg-gradient-to-br from-primary to-purple-500 rounded-xl flex items-center justify-center">
              <svg className="w-5 h-5 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
              </svg>
            </div>
            <div>
              <h2 className="text-lg font-bold text-dark">Credit Application</h2>
              <p className="text-xs text-cau-gray">Fill in your information for encrypted evaluation</p>
            </div>
          </div>

          <div className="grid md:grid-cols-2 gap-6 mb-8">
            {CREDIT_FEATURES.map((feature) => renderFeatureInput(feature))}
          </div>

          <button
            type="submit"
            disabled={loading || !isReady}
            className="w-full py-4 bg-gradient-to-r from-primary via-accent to-primary bg-size-200 bg-pos-0 hover:bg-pos-100
              text-white text-lg font-bold rounded-xl shadow-medium hover:shadow-hard
              transition-all duration-500 disabled:opacity-50 disabled:cursor-not-allowed
              flex items-center justify-center gap-3 group"
          >
            {loading ? (
              <>
                <svg className="animate-spin h-6 w-6" viewBox="0 0 24 24">
                  <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" fill="none" />
                  <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
                </svg>
                <span>Computing Encrypted Score...</span>
              </>
            ) : (
              <>
                <svg className="w-6 h-6 group-hover:scale-110 transition-transform" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 10V3L4 14h7v7l9-11h-7z" />
                </svg>
                <span>Calculate Credit Score</span>
              </>
            )}
          </button>
        </form>

        {/* Results Section */}
        {result !== null && (
          <div className="bg-white rounded-2xl shadow-soft border border-cau-light-gray/50 p-8">
            <div className="flex items-center gap-3 mb-8">
              <div className="w-10 h-10 bg-gradient-to-br from-green-500 to-emerald-500 rounded-xl flex items-center justify-center">
                <svg className="w-5 h-5 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z" />
                </svg>
              </div>
              <div>
                <h2 className="text-lg font-bold text-dark">Evaluation Results</h2>
                <p className="text-xs text-cau-gray">Your encrypted credit score assessment</p>
              </div>
            </div>

            <div className="grid md:grid-cols-2 gap-6 mb-8">
              {/* Score Display */}
              <div className="relative overflow-hidden bg-gradient-to-br from-light to-white rounded-2xl p-8 border border-cau-light-gray/50">
                <div className="absolute top-0 right-0 w-32 h-32 bg-primary/5 rounded-full -mr-16 -mt-16"></div>
                <p className="text-sm font-semibold text-cau-gray mb-3">Credit Score</p>
                <div className="flex items-baseline gap-2 mb-2">
                  <span className="text-6xl font-bold bg-gradient-to-r from-primary to-accent bg-clip-text text-transparent">
                    {(result * 100).toFixed(1)}
                  </span>
                  <span className="text-3xl font-bold text-cau-gray/40">/ 100</span>
                </div>
                <div className="w-full bg-cau-light-gray/30 rounded-full h-2 mt-4">
                  <div 
                    className={`h-2 rounded-full ${getCreditRating(result).color} transition-all duration-1000`}
                    style={{ width: `${result * 100}%` }}
                  />
                </div>
              </div>

              {/* Rating Badge */}
              <div className="bg-gradient-to-br from-light to-white rounded-2xl p-8 border border-cau-light-gray/50">
                <p className="text-sm font-semibold text-cau-gray mb-4">Credit Rating</p>
                <div className={`${getCreditRating(result).bgColor} rounded-xl p-6 text-center`}>
                  <div className="text-4xl mb-3">{getCreditRating(result).icon}</div>
                  <div className={`text-2xl font-bold ${getCreditRating(result).textColor}`}>
                    {getCreditRating(result).label}
                  </div>
                </div>
              </div>
            </div>

            {/* Performance Metrics */}
            {timings && (
              <div className="bg-light rounded-xl p-6 border border-cau-light-gray/50">
                <h3 className="text-sm font-bold text-dark mb-4 flex items-center gap-2">
                  <svg className="w-4 h-4 text-primary" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
                  </svg>
                  Performance Metrics
                </h3>
                
                <div className="grid grid-cols-2 md:grid-cols-5 gap-4">
                  {timings.keygenTime && (
                    <div className="bg-white rounded-lg p-4 text-center border border-cau-light-gray/30">
                      <p className="text-xs text-cau-gray mb-1">Key Generation</p>
                      <p className="text-lg font-bold text-dark">{timings.keygenTime.toFixed(0)}<span className="text-xs text-cau-gray ml-1">ms</span></p>
                    </div>
                  )}
                  {timings.encryptionTime && (
                    <div className="bg-white rounded-lg p-4 text-center border border-cau-light-gray/30">
                      <p className="text-xs text-cau-gray mb-1">Encryption</p>
                      <p className="text-lg font-bold text-primary">{timings.encryptionTime.toFixed(0)}<span className="text-xs text-cau-gray ml-1">ms</span></p>
                    </div>
                  )}
                  {timings.inferenceTime && (
                    <div className="bg-white rounded-lg p-4 text-center border border-cau-light-gray/30">
                      <p className="text-xs text-cau-gray mb-1">Inference</p>
                      <p className="text-lg font-bold text-accent">{timings.inferenceTime.toFixed(0)}<span className="text-xs text-cau-gray ml-1">ms</span></p>
                    </div>
                  )}
                  {timings.decryptionTime && (
                    <div className="bg-white rounded-lg p-4 text-center border border-cau-light-gray/30">
                      <p className="text-xs text-cau-gray mb-1">Decryption</p>
                      <p className="text-lg font-bold text-green-600">{timings.decryptionTime.toFixed(0)}<span className="text-xs text-cau-gray ml-1">ms</span></p>
                    </div>
                  )}
                  {(timings.encryptionTime && timings.inferenceTime && timings.decryptionTime) && (
                    <div className="bg-gradient-to-br from-primary to-accent rounded-lg p-4 text-center text-white">
                      <p className="text-xs mb-1 opacity-90">Total Time</p>
                      <p className="text-lg font-bold">{(timings.encryptionTime + timings.inferenceTime + timings.decryptionTime).toFixed(0)}<span className="text-xs ml-1 opacity-75">ms</span></p>
                    </div>
                  )}
                </div>
              </div>
            )}

            {/* Security Notice */}
            <div className="mt-6 bg-blue-50 border border-primary/20 rounded-xl p-4">
              <div className="flex items-start gap-3">
                <svg className="w-5 h-5 text-primary flex-shrink-0 mt-0.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.042-.133-2.052-.382-3.016z" />
                </svg>
                <div>
                  <p className="text-sm font-semibold text-primary mb-1">Privacy Protected</p>
                  <p className="text-xs text-cau-gray">All computations performed on encrypted data using homomorphic encryption. Your sensitive information never leaves your device in plaintext.</p>
                </div>
              </div>
            </div>
          </div>
        )}
      </main>

      {/* Footer */}
      <footer className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8 text-center">
        <p className="text-sm text-cau-gray">
          Powered by <span className="font-semibold">Lattigo CKKS</span> • Privacy-Preserving Credit Evaluation
        </p>
      </footer>
    </div>
  );
};

export default CreditInputForm;
