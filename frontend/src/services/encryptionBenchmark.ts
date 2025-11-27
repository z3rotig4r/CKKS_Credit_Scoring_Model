/**
 * Comprehensive benchmark comparing sequential vs parallel encryption
 */

import wasmLoader from './wasmLoader';
import { BenchmarkWorkerPool } from './benchmarkWorkerPool';

export interface BenchmarkResult {
  sequential: {
    totalTime: number;
    perFeatureTime: number;
    times: number[];
  };
  parallel: {
    totalTime: number;
    perFeatureTime: number;
    times: number[];
    workerCount: number;
  };
  speedup: number;
  overhead: number;
}

export class EncryptionBenchmark {
  private publicKey: Uint8Array | null = null;

  async init(publicKey: Uint8Array): Promise<void> {
    this.publicKey = publicKey;
    await wasmLoader.load();
  }

  async benchmarkSequential(values: number[]): Promise<{
    totalTime: number;
    times: number[];
  }> {
    if (!this.publicKey) throw new Error('Not initialized');

    const times: number[] = [];
    const startTotal = performance.now();

    for (const value of values) {
      const start = performance.now();
      await wasmLoader.callFunction('fheEncrypt', this.publicKey, value);
      times.push(performance.now() - start);
    }

    const totalTime = performance.now() - startTotal;
    return { totalTime, times };
  }

  async benchmarkParallel(
    values: number[],
    workerCount: number
  ): Promise<{
    totalTime: number;
    times: number[];
  }> {
    if (!this.publicKey) throw new Error('Not initialized');

    const pool = new BenchmarkWorkerPool(workerCount);
    
    try {
      await pool.init(this.publicKey);
      const { totalTime, perFeatureTimes } = await pool.encryptParallel(values);
      return { totalTime, times: perFeatureTimes };
    } finally {
      pool.terminate();
    }
  }

  async runComprehensiveBenchmark(
    featureCount: number = 5,
    iterations: number = 3
  ): Promise<BenchmarkResult> {
    console.log(`🔬 Starting comprehensive benchmark:`);
    console.log(`   Features: ${featureCount}`);
    console.log(`   Iterations: ${iterations}`);
    console.log(`   Workers: ${navigator.hardwareConcurrency || 4}`);

    const testValues = Array(featureCount)
      .fill(0)
      .map(() => Math.random());

    // Warmup
    console.log('🔥 Warming up...');
    await this.benchmarkSequential(testValues);

    // Sequential benchmarks
    console.log('📊 Running sequential benchmarks...');
    const sequentialResults: number[] = [];
    const sequentialTimes: number[][] = [];

    for (let i = 0; i < iterations; i++) {
      const { totalTime, times } = await this.benchmarkSequential(testValues);
      sequentialResults.push(totalTime);
      sequentialTimes.push(times);
      console.log(`   Iteration ${i + 1}: ${totalTime.toFixed(2)}ms`);
    }

    // Parallel benchmarks
    console.log('📊 Running parallel benchmarks...');
    const parallelResults: number[] = [];
    const parallelTimes: number[][] = [];
    const workerCount = navigator.hardwareConcurrency || 4;

    for (let i = 0; i < iterations; i++) {
      const { totalTime, times } = await this.benchmarkParallel(testValues, workerCount);
      parallelResults.push(totalTime);
      parallelTimes.push(times);
      console.log(`   Iteration ${i + 1}: ${totalTime.toFixed(2)}ms`);
    }

    // Calculate statistics
    const avgSequential = sequentialResults.reduce((a, b) => a + b) / iterations;
    const avgParallel = parallelResults.reduce((a, b) => a + b) / iterations;
    const avgSeqPerFeature =
      sequentialTimes.flat().reduce((a, b) => a + b) / (iterations * featureCount);
    const avgParPerFeature =
      parallelTimes.flat().reduce((a, b) => a + b) / (iterations * featureCount);

    const speedup = avgSequential / avgParallel;
    const overhead = avgParallel - avgParPerFeature;

    console.log('\n📈 Results:');
    console.log(`   Sequential: ${avgSequential.toFixed(2)}ms`);
    console.log(`   Parallel: ${avgParallel.toFixed(2)}ms`);
    console.log(`   Speedup: ${speedup.toFixed(2)}x`);
    console.log(`   Overhead: ${overhead.toFixed(2)}ms`);

    return {
      sequential: {
        totalTime: avgSequential,
        perFeatureTime: avgSeqPerFeature,
        times: sequentialResults
      },
      parallel: {
        totalTime: avgParallel,
        perFeatureTime: avgParPerFeature,
        times: parallelResults,
        workerCount
      },
      speedup,
      overhead
    };
  }
}
