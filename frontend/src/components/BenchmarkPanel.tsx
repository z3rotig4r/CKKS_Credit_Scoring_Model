import React, { useState } from 'react';
import { useFHE } from '../contexts/FHEContext';
import { EncryptionBenchmark, BenchmarkResult } from '../services/encryptionBenchmark';
import './BenchmarkPanel.css';

export const BenchmarkPanel: React.FC = () => {
  const { isReady, keys } = useFHE() as any;
  const [running, setRunning] = useState(false);
  const [result, setResult] = useState<BenchmarkResult | null>(null);

  const runBenchmark = async () => {
    if (!keys.publicKey) {
      alert('Please generate keys first');
      return;
    }

    setRunning(true);
    setResult(null);

    try {
      const benchmark = new EncryptionBenchmark();
      await benchmark.init(keys.publicKey);
      const benchmarkResult = await benchmark.runComprehensiveBenchmark(5, 3);
      setResult(benchmarkResult);
    } catch (error: any) {
      console.error('Benchmark failed:', error);
      alert(`Benchmark failed: ${error.message}`);
    } finally {
      setRunning(false);
    }
  };

  return (
    <div className="benchmark-panel">
      <h2>🔬 Encryption Performance Benchmark</h2>
      
      <div className="benchmark-info">
        <p>
          This benchmark compares sequential vs parallel encryption performance
          using Web Workers. It encrypts 5 features 3 times for each method.
        </p>
      </div>

      <button
        onClick={runBenchmark}
        disabled={!isReady || running}
        className="benchmark-button"
      >
        {running ? '⏳ Running Benchmark...' : '🚀 Run Benchmark'}
      </button>

      {result && (
        <div className="benchmark-results">
          <h3>📊 Results</h3>
          
          <div className="result-grid">
            <div className="result-card sequential">
              <h4>Sequential Encryption</h4>
              <div className="metric">
                <span className="label">Total Time:</span>
                <span className="value">{result.sequential.totalTime.toFixed(2)}ms</span>
              </div>
              <div className="metric">
                <span className="label">Per Feature:</span>
                <span className="value">{result.sequential.perFeatureTime.toFixed(2)}ms</span>
              </div>
              <div className="metric">
                <span className="label">CPU Usage:</span>
                <span className="value">Single Core</span>
              </div>
            </div>

            <div className="result-card parallel">
              <h4>Parallel Encryption (Web Workers)</h4>
              <div className="metric">
                <span className="label">Total Time:</span>
                <span className="value">{result.parallel.totalTime.toFixed(2)}ms</span>
              </div>
              <div className="metric">
                <span className="label">Per Feature:</span>
                <span className="value">{result.parallel.perFeatureTime.toFixed(2)}ms</span>
              </div>
              <div className="metric">
                <span className="label">Workers:</span>
                <span className="value">{result.parallel.workerCount}</span>
              </div>
              <div className="metric">
                <span className="label">Overhead:</span>
                <span className="value">{result.overhead.toFixed(2)}ms</span>
              </div>
            </div>
          </div>

          <div className="speedup-card">
            <h4>Performance Improvement</h4>
            <div className="speedup-value">
              {result.speedup >= 1 
                ? `${result.speedup.toFixed(2)}x Faster` 
                : `${(1/result.speedup).toFixed(2)}x Slower`}
            </div>
            {result.speedup < 1 && (
              <div className="warning">
                ⚠️ Web Workers are slower than sequential due to:
                <ul>
                  <li>Worker creation/initialization overhead</li>
                  <li>Message passing serialization cost</li>
                  <li>WASM module loading per worker</li>
                </ul>
              </div>
            )}
            {result.speedup >= 1 && result.speedup < 2 && (
              <div className="info">
                ℹ️ Modest speedup. Overhead reduces benefits for WASM operations.
              </div>
            )}
            {result.speedup >= 2 && (
              <div className="success">
                ✅ Significant speedup! Parallel encryption is beneficial.
              </div>
            )}
          </div>

          <div className="analysis">
            <h4>📈 Analysis</h4>
            <p>
              <strong>Sequential:</strong> {result.sequential.totalTime.toFixed(0)}ms total 
              = {result.sequential.perFeatureTime.toFixed(0)}ms × 5 features
            </p>
            <p>
              <strong>Parallel:</strong> {result.parallel.totalTime.toFixed(0)}ms total 
              = max({result.parallel.perFeatureTime.toFixed(0)}ms per feature) + {result.overhead.toFixed(0)}ms overhead
            </p>
            <p>
              <strong>Overhead includes:</strong> Worker pool initialization, 
              WASM module loading in each worker, message serialization
            </p>
          </div>
        </div>
      )}
    </div>
  );
};
