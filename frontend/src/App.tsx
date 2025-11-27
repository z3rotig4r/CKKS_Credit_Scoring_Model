import React, { useState } from 'react';
import { FHEProvider } from './contexts/FHEContext';
import CreditInputForm from './components/CreditInputForm';
import { BenchmarkPanel } from './components/BenchmarkPanel';

function App() {
  const [activeTab, setActiveTab] = useState<'credit' | 'benchmark'>('credit');

  const appStyle = {
    minHeight: '100vh',
    background: 'linear-gradient(135deg, #667eea 0%, #764ba2 100%)',
    padding: '48px 16px',
  };

  const containerStyle = {
    maxWidth: '960px',
    margin: '0 auto',
  };

  const headerStyle = {
    textAlign: 'center' as const,
    marginBottom: '48px',
  };

  const titleStyle = {
    fontSize: '48px',
    fontWeight: 'bold',
    color: 'white',
    marginBottom: '16px',
  };

  const subtitleStyle = {
    fontSize: '20px',
    color: 'rgba(255, 255, 255, 0.9)',
  };

  const footerStyle = {
    marginTop: '48px',
    textAlign: 'center' as const,
    color: 'rgba(255, 255, 255, 0.8)',
    fontSize: '14px',
  };

  return (
    <FHEProvider>
      <div style={appStyle}>
        <div style={containerStyle}>
          <div style={headerStyle}>
            <h1 style={titleStyle}>
              🔒 Private Credit Scoring
            </h1>
            <p style={subtitleStyle}>
              Homomorphic Encryption Powered by CKKS
            </p>
          </div>

          {/* Tab Navigation */}
          <div style={{ display: 'flex', justifyContent: 'center', marginBottom: '32px', gap: '16px' }}>
            <button
              onClick={() => setActiveTab('credit')}
              style={{
                padding: '12px 24px',
                borderRadius: '8px',
                fontWeight: 'bold',
                border: 'none',
                cursor: 'pointer',
                background: activeTab === 'credit' ? 'white' : 'rgba(255, 255, 255, 0.2)',
                color: activeTab === 'credit' ? '#667eea' : 'white',
                transition: 'all 0.3s',
              }}
            >
              💳 Credit Scoring
            </button>
            <button
              onClick={() => setActiveTab('benchmark')}
              style={{
                padding: '12px 24px',
                borderRadius: '8px',
                fontWeight: 'bold',
                border: 'none',
                cursor: 'pointer',
                background: activeTab === 'benchmark' ? 'white' : 'rgba(255, 255, 255, 0.2)',
                color: activeTab === 'benchmark' ? '#667eea' : 'white',
                transition: 'all 0.3s',
              }}
            >
              📊 Benchmark
            </button>
          </div>

          {activeTab === 'credit' ? <CreditInputForm /> : <BenchmarkPanel />}

          <div style={footerStyle}>
            <p>Built with Lattigo CKKS • WebAssembly • React + TypeScript</p>
          </div>
        </div>
      </div>
    </FHEProvider>
  );
}

export default App;
