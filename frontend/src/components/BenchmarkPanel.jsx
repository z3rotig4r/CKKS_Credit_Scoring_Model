import React, { useState } from 'react';
import { useFHE } from '../contexts/FHEContext';
import PerformanceBenchmark from '../utils/performanceBenchmark';

export default function BenchmarkPanel() {
  const { wasmLoaded, isReady, encrypt, encryptMultiple, supportsParallelEncryption } = useFHE();
  const keysGenerated = isReady;
  const [running, setRunning] = useState(false);
  const [results, setResults] = useState(null);
  const [report, setReport] = useState('');

  const testFeatures = [650.0, 0.35, 25000.0, 5.0, 30.0]; // Typical credit features

  const runBenchmark = async () => {
    if (!wasmLoaded || !keysGenerated) {
      alert('Please generate keys first!');
      return;
    }

    setRunning(true);
    setResults(null);
    setReport('');

    try {
      const benchmark = new PerformanceBenchmark();

      // Sequential encryption function
      const sequentialFn = async (features) => {
        await Promise.all(features.map(f => encrypt(f)));
      };

      // Parallel encryption function
      const parallelFn = async (features) => {
        await encryptMultiple(features);
      };

      // Run comparison (3 iterations each)
      const comparison = await benchmark.comparePerformance(
        sequentialFn,
        parallelFn,
        testFeatures,
        3
      );

      setResults(comparison);
      setReport(benchmark.generateReport(comparison));

    } catch (error) {
      console.error('❌ Benchmark failed:', error);
      alert('Benchmark failed: ' + error.message);
    } finally {
      setRunning(false);
    }
  };

  const copyReport = () => {
    navigator.clipboard.writeText(report);
    alert('Report copied to clipboard!');
  };

  return (
    <div className="space-y-6">
      <div className="bg-white rounded-lg shadow-md p-6">
        <div className="flex items-center gap-3 mb-6">
          <div className="w-12 h-12 bg-gradient-to-br from-purple-500 to-indigo-500 rounded-xl flex items-center justify-center">
            <svg className="w-6 h-6 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z" />
            </svg>
          </div>
          <div>
            <h2 className="text-xl font-bold text-gray-800">Performance Benchmark</h2>
            <p className="text-sm text-gray-600">Compare sequential vs parallel encryption</p>
          </div>
        </div>

        {!supportsParallelEncryption && (
          <div className="bg-yellow-50 border border-yellow-200 rounded-lg p-4 mb-6">
            <div className="flex items-start gap-3">
              <svg className="w-5 h-5 text-yellow-600 flex-shrink-0 mt-0.5" fill="currentColor" viewBox="0 0 20 20">
                <path fillRule="evenodd" d="M8.257 3.099c.765-1.36 2.722-1.36 3.486 0l5.58 9.92c.75 1.334-.213 2.98-1.742 2.98H4.42c-1.53 0-2.493-1.646-1.743-2.98l5.58-9.92zM11 13a1 1 0 11-2 0 1 1 0 012 0zm-1-8a1 1 0 00-1 1v3a1 1 0 002 0V6a1 1 0 00-1-1z" clipRule="evenodd" />
              </svg>
              <div>
                <p className="text-sm font-semibold text-yellow-800 mb-1">Web Workers Not Supported</p>
                <p className="text-xs text-yellow-700">Your browser doesn't support Web Workers. Only sequential encryption will be tested.</p>
              </div>
            </div>
          </div>
        )}

        <div className="mb-6">
          <h3 className="text-sm font-semibold text-gray-700 mb-2">Test Configuration</h3>
          <div className="bg-gray-50 rounded-lg p-4 space-y-2 text-sm">
            <div className="flex justify-between">
              <span className="text-gray-600">Features to Encrypt:</span>
              <span className="font-mono font-semibold text-gray-800">{testFeatures.length}</span>
            </div>
            <div className="flex justify-between">
              <span className="text-gray-600">Iterations per Method:</span>
              <span className="font-mono font-semibold text-gray-800">3</span>
            </div>
            <div className="flex justify-between">
              <span className="text-gray-600">CPU Cores:</span>
              <span className="font-mono font-semibold text-gray-800">{navigator.hardwareConcurrency}</span>
            </div>
            <div className="flex justify-between">
              <span className="text-gray-600">WASM Status:</span>
              <span className={`font-semibold ${wasmLoaded ? 'text-green-600' : 'text-red-600'}`}>
                {wasmLoaded ? '✓ Loaded' : '✗ Not Loaded'}
              </span>
            </div>
            <div className="flex justify-between">
              <span className="text-gray-600">Keys Status:</span>
              <span className={`font-semibold ${keysGenerated ? 'text-green-600' : 'text-red-600'}`}>
                {keysGenerated ? '✓ Generated' : '✗ Not Generated'}
              </span>
            </div>
          </div>
        </div>

        <button
          onClick={runBenchmark}
          disabled={running || !wasmLoaded || !keysGenerated}
          className={`w-full py-3 px-6 rounded-lg font-semibold text-white transition-all ${
            running || !wasmLoaded || !keysGenerated
              ? 'bg-gray-400 cursor-not-allowed'
              : 'bg-gradient-to-r from-purple-500 to-indigo-500 hover:from-purple-600 hover:to-indigo-600 hover:shadow-lg'
          }`}
        >
          {running ? (
            <span className="flex items-center justify-center gap-2">
              <svg className="animate-spin h-5 w-5" viewBox="0 0 24 24">
                <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" fill="none" />
                <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
              </svg>
              Running Benchmark...
            </span>
          ) : (
            '🚀 Run Benchmark'
          )}
        </button>
      </div>

      {results && (
        <div className="bg-white rounded-lg shadow-md p-6 space-y-6">
          <div className="flex items-center justify-between mb-4">
            <h3 className="text-lg font-bold text-gray-800">Benchmark Results</h3>
            <button
              onClick={copyReport}
              className="px-4 py-2 bg-gray-100 hover:bg-gray-200 rounded-lg text-sm font-medium text-gray-700 transition-colors"
            >
              📋 Copy Report
            </button>
          </div>

          {/* Summary Cards */}
          <div className="grid md:grid-cols-3 gap-4">
            <div className="bg-gradient-to-br from-blue-50 to-blue-100 rounded-lg p-4 border border-blue-200">
              <p className="text-xs font-semibold text-blue-600 mb-1">Sequential</p>
              <p className="text-2xl font-bold text-blue-900">{results.sequential.avgTotalTime.toFixed(0)}ms</p>
              <p className="text-xs text-blue-700 mt-1">{results.sequential.avgPerFeature.toFixed(1)}ms per feature</p>
            </div>

            <div className="bg-gradient-to-br from-purple-50 to-purple-100 rounded-lg p-4 border border-purple-200">
              <p className="text-xs font-semibold text-purple-600 mb-1">Parallel (Workers)</p>
              <p className="text-2xl font-bold text-purple-900">{results.parallel.avgTotalTime.toFixed(0)}ms</p>
              <p className="text-xs text-purple-700 mt-1">{results.parallel.avgPerFeature.toFixed(1)}ms per feature</p>
            </div>

            <div className="bg-gradient-to-br from-green-50 to-green-100 rounded-lg p-4 border border-green-200">
              <p className="text-xs font-semibold text-green-600 mb-1">Performance Gain</p>
              <p className="text-2xl font-bold text-green-900">{results.speedup.toFixed(2)}x</p>
              <p className="text-xs text-green-700 mt-1">{results.improvement} faster</p>
            </div>
          </div>

          {/* Detailed Results */}
          <div className="space-y-4">
            <div>
              <h4 className="text-sm font-semibold text-gray-700 mb-2">Sequential Test Runs</h4>
              <div className="space-y-1">
                {results.sequential.results.map((r, i) => (
                  <div key={i} className="flex justify-between items-center bg-gray-50 rounded px-3 py-2 text-sm">
                    <span className="text-gray-600">Run {i + 1}</span>
                    <span className="font-mono font-semibold text-gray-800">{r.totalTime.toFixed(2)}ms</span>
                  </div>
                ))}
              </div>
            </div>

            <div>
              <h4 className="text-sm font-semibold text-gray-700 mb-2">Parallel Test Runs</h4>
              <div className="space-y-1">
                {results.parallel.results.map((r, i) => (
                  <div key={i} className="flex justify-between items-center bg-purple-50 rounded px-3 py-2 text-sm">
                    <span className="text-gray-600">Run {i + 1}</span>
                    <span className="font-mono font-semibold text-gray-800">{r.totalTime.toFixed(2)}ms</span>
                  </div>
                ))}
              </div>
            </div>
          </div>

          {/* Markdown Report Preview */}
          <div>
            <h4 className="text-sm font-semibold text-gray-700 mb-2">Markdown Report</h4>
            <pre className="bg-gray-900 text-gray-100 rounded-lg p-4 text-xs overflow-x-auto">
              {report}
            </pre>
          </div>
        </div>
      )}
    </div>
  );
}
