/**
 * Worker Performance Demo
 * 
 * Web Worker와 SharedArrayBuffer의 성능 향상을 시연하는 데모 컴포넌트
 */

import React, { useState } from 'react';
import { workerManager } from '../services/workerManager';
import wasmLoader from '../services/wasmLoader';
import './WorkerDemo.css';

const WorkerDemo = () => {
  const [status, setStatus] = useState('idle');
  const [results, setResults] = useState(null);
  const [logs, setLogs] = useState([]);
  const [useWorker, setUseWorker] = useState(true);
  const [useZeroCopy, setUseZeroCopy] = useState(true);
  const [sharedMemorySupported, setSharedMemorySupported] = useState(false);

  const addLog = (message) => {
    setLogs(prev => [...prev, `[${new Date().toLocaleTimeString()}] ${message}`]);
  };

  // SharedArrayBuffer 지원 확인
  React.useEffect(() => {
    const supported = typeof SharedArrayBuffer !== 'undefined';
    setSharedMemorySupported(supported);
    addLog(`SharedArrayBuffer: ${supported ? '✅ Supported' : '❌ Not Supported'}`);
  }, []);

  const runBenchmark = async () => {
    try {
      setStatus('running');
      setLogs([]);
      addLog('🧪 Starting performance benchmark...');

      // WASM 초기화
      await wasmLoader.load();
      addLog('✅ WASM loaded');

      // 키 생성
      addLog('🔑 Generating keys...');
      const keyGenStart = performance.now();
      const keyPair = await wasmLoader.callFunction('fheKeygen');
      const keyGenTime = performance.now() - keyGenStart;
      addLog(`✅ Keys generated in ${keyGenTime.toFixed(2)}ms`);

      // 테스트 데이터
      const features = [0.5, 0.75, 0.3, 0.85, 0.2];
      addLog(`📊 Test features: ${features.length}`);

      const benchmarkResults = {};

      // 1. Main Thread 벤치마크
      if (!useWorker) {
        addLog('🔄 Running Main Thread benchmark...');
        const mainThreadStart = performance.now();
        
        const encryptedFeatures = [];
        for (let i = 0; i < features.length; i++) {
          const encrypted = await wasmLoader.callFunction('fheEncrypt', keyPair.publicKey, features[i]);
          encryptedFeatures.push(encrypted);
          addLog(`  Feature ${i+1}/${features.length} encrypted`);
        }
        
        const mainThreadTime = performance.now() - mainThreadStart;
        benchmarkResults.mainThread = {
          totalTime: mainThreadTime,
          avgPerFeature: mainThreadTime / features.length,
          method: 'Main Thread'
        };
        addLog(`✅ Main Thread: ${mainThreadTime.toFixed(2)}ms (${(mainThreadTime / features.length).toFixed(2)}ms/feature)`);
      }

      // 2. Web Worker 벤치마크
      if (useWorker) {
        addLog('🔄 Initializing Web Worker...');
        await workerManager.initialize();
        addLog('✅ Worker ready');

        if (useZeroCopy && sharedMemorySupported) {
          addLog('🔄 Initializing SharedArrayBuffer...');
          await workerManager.initializeSharedMemory();
          addLog('✅ SharedArrayBuffer ready');
        }

        // 진행 상태 콜백
        workerManager.setProgressCallback((data) => {
          addLog(`  Progress: ${data.current}/${data.total} (${data.percentage}%)`);
        });

        addLog(`🔄 Running Worker benchmark (Zero-Copy: ${useZeroCopy && sharedMemorySupported ? 'ON' : 'OFF'})...`);
        const workerStart = performance.now();
        
        const result = await workerManager.encryptFeatures(
          features,
          keyPair.publicKey,
          useZeroCopy && sharedMemorySupported
        );
        
        const workerTime = performance.now() - workerStart;
        benchmarkResults.worker = {
          totalTime: workerTime,
          avgPerFeature: result.avgTimePerFeature,
          method: `Worker ${result.zeroCopy ? '(Zero-Copy)' : '(Copy)'}`,
          zeroCopy: result.zeroCopy
        };
        addLog(`✅ Worker: ${workerTime.toFixed(2)}ms (${result.avgTimePerFeature.toFixed(2)}ms/feature, Zero-Copy: ${result.zeroCopy})`);

        // Cleanup
        workerManager.cleanup();
      }

      // 결과 분석
      if (benchmarkResults.mainThread && benchmarkResults.worker) {
        const speedup = benchmarkResults.mainThread.totalTime / benchmarkResults.worker.totalTime;
        benchmarkResults.speedup = speedup;
        addLog(`📈 Speedup: ${speedup.toFixed(2)}x`);
      }

      setResults(benchmarkResults);
      setStatus('completed');
      addLog('🎉 Benchmark completed!');

    } catch (error) {
      console.error('Benchmark error:', error);
      addLog(`❌ Error: ${error.message}`);
      setStatus('error');
    }
  };

  return (
    <div className="worker-demo">
      <h2>🚀 Web Worker Performance Demo</h2>
      
      <div className="demo-info">
        <p>
          이 데모는 Web Worker와 SharedArrayBuffer(Zero-Copy)의 성능 향상을 시연합니다.
        </p>
        <ul>
          <li><strong>Main Thread</strong>: 기존 방식 (UI 블로킹)</li>
          <li><strong>Web Worker</strong>: 백그라운드 스레드 (UI 응답성 유지)</li>
          <li><strong>Zero-Copy</strong>: SharedArrayBuffer로 메모리 복사 없이 데이터 공유</li>
        </ul>
      </div>

      <div className="demo-status">
        <div className="status-badge" data-status={status}>
          {status === 'idle' && '⏸️ Ready'}
          {status === 'running' && '⏳ Running...'}
          {status === 'completed' && '✅ Completed'}
          {status === 'error' && '❌ Error'}
        </div>
        
        <div className="support-info">
          <span className={sharedMemorySupported ? 'supported' : 'not-supported'}>
            SharedArrayBuffer: {sharedMemorySupported ? '✅ Supported' : '❌ Not Supported'}
          </span>
          {!sharedMemorySupported && (
            <small>Zero-Copy 기능을 사용하려면 HTTPS 또는 적절한 보안 헤더가 필요합니다.</small>
          )}
        </div>
      </div>

      <div className="demo-controls">
        <label>
          <input
            type="checkbox"
            checked={useWorker}
            onChange={(e) => setUseWorker(e.target.checked)}
            disabled={status === 'running'}
          />
          Use Web Worker
        </label>
        
        <label>
          <input
            type="checkbox"
            checked={useZeroCopy}
            onChange={(e) => setUseZeroCopy(e.target.checked)}
            disabled={status === 'running' || !useWorker || !sharedMemorySupported}
          />
          Use Zero-Copy (SharedArrayBuffer)
          {!sharedMemorySupported && ' - Not Available'}
        </label>

        <button
          onClick={runBenchmark}
          disabled={status === 'running'}
          className="run-button"
        >
          {status === 'running' ? '⏳ Running...' : '▶️ Run Benchmark'}
        </button>
      </div>

      {results && (
        <div className="demo-results">
          <h3>📊 Results</h3>
          
          <table>
            <thead>
              <tr>
                <th>Method</th>
                <th>Total Time</th>
                <th>Avg per Feature</th>
                <th>Speedup</th>
              </tr>
            </thead>
            <tbody>
              {results.mainThread && (
                <tr>
                  <td>{results.mainThread.method}</td>
                  <td>{results.mainThread.totalTime.toFixed(2)} ms</td>
                  <td>{results.mainThread.avgPerFeature.toFixed(2)} ms</td>
                  <td>1.00x (baseline)</td>
                </tr>
              )}
              {results.worker && (
                <tr className="highlight">
                  <td>{results.worker.method}</td>
                  <td>{results.worker.totalTime.toFixed(2)} ms</td>
                  <td>{results.worker.avgPerFeature.toFixed(2)} ms</td>
                  <td className="speedup">
                    {results.speedup ? `${results.speedup.toFixed(2)}x` : 'N/A'}
                  </td>
                </tr>
              )}
            </tbody>
          </table>

          {results.speedup && (
            <div className="result-summary">
              <h4>
                {results.speedup > 1 
                  ? `🎉 ${results.speedup.toFixed(2)}x faster with ${results.worker.method}!`
                  : results.speedup < 1
                  ? `⚠️ ${(1/results.speedup).toFixed(2)}x slower (overhead)`
                  : '🤔 Similar performance'
                }
              </h4>
            </div>
          )}
        </div>
      )}

      <div className="demo-logs">
        <h3>📝 Logs</h3>
        <div className="log-container">
          {logs.map((log, index) => (
            <div key={index} className="log-entry">{log}</div>
          ))}
        </div>
      </div>

      <div className="demo-footer">
        <p>
          <strong>Note:</strong> Worker 오버헤드로 인해 작은 데이터셋에서는 성능 향상이 미미할 수 있습니다.
          큰 데이터셋이나 복잡한 연산에서 Worker의 진가를 발휘합니다.
        </p>
      </div>
    </div>
  );
};

export default WorkerDemo;
