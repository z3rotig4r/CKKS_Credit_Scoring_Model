/**
 * Performance Benchmark Utility
 * Measures encryption performance with and without Web Workers
 */

export class PerformanceBenchmark {
  constructor() {
    this.results = [];
  }

  /**
   * Run a single benchmark test
   */
  async runTest(name, encryptFn, features) {
    const startTime = performance.now();
    await encryptFn(features);
    const endTime = performance.now();
    
    const duration = endTime - startTime;
    const perFeature = duration / features.length;
    
    const result = {
      name,
      totalTime: duration,
      perFeatureTime: perFeature,
      featureCount: features.length,
      timestamp: new Date().toISOString()
    };
    
    this.results.push(result);
    return result;
  }

  /**
   * Compare sequential vs parallel encryption
   */
  async comparePerformance(sequentialFn, parallelFn, features, iterations = 3) {
    console.log(`🔬 Starting performance benchmark (${iterations} iterations)...`);
    
    const sequentialResults = [];
    const parallelResults = [];
    
    // Run sequential tests
    for (let i = 0; i < iterations; i++) {
      console.log(`⏳ Sequential test ${i + 1}/${iterations}...`);
      const result = await this.runTest(`Sequential-${i + 1}`, sequentialFn, features);
      sequentialResults.push(result);
      
      // Small delay between tests
      await new Promise(resolve => setTimeout(resolve, 500));
    }
    
    // Run parallel tests
    for (let i = 0; i < iterations; i++) {
      console.log(`⚡ Parallel test ${i + 1}/${iterations}...`);
      const result = await this.runTest(`Parallel-${i + 1}`, parallelFn, features);
      parallelResults.push(result);
      
      // Small delay between tests
      await new Promise(resolve => setTimeout(resolve, 500));
    }
    
    // Calculate statistics
    const seqAvg = this.calculateAverage(sequentialResults.map(r => r.totalTime));
    const parAvg = this.calculateAverage(parallelResults.map(r => r.totalTime));
    const speedup = seqAvg / parAvg;
    
    const comparison = {
      sequential: {
        avgTotalTime: seqAvg,
        avgPerFeature: seqAvg / features.length,
        results: sequentialResults
      },
      parallel: {
        avgTotalTime: parAvg,
        avgPerFeature: parAvg / features.length,
        results: parallelResults
      },
      speedup: speedup,
      improvement: ((speedup - 1) * 100).toFixed(1) + '%',
      featureCount: features.length
    };
    
    console.log('✅ Benchmark complete!');
    this.logComparison(comparison);
    
    return comparison;
  }

  calculateAverage(values) {
    return values.reduce((sum, val) => sum + val, 0) / values.length;
  }

  logComparison(comparison) {
    console.log('\n📊 Performance Benchmark Results');
    console.log('═══════════════════════════════════════');
    console.log(`Features: ${comparison.featureCount}`);
    console.log('');
    console.log('Sequential Encryption:');
    console.log(`  Total Time:   ${comparison.sequential.avgTotalTime.toFixed(2)}ms`);
    console.log(`  Per Feature:  ${comparison.sequential.avgPerFeature.toFixed(2)}ms`);
    console.log('');
    console.log('Parallel Encryption (Web Workers):');
    console.log(`  Total Time:   ${comparison.parallel.avgTotalTime.toFixed(2)}ms`);
    console.log(`  Per Feature:  ${comparison.parallel.avgPerFeature.toFixed(2)}ms`);
    console.log('');
    console.log(`⚡ Speedup:      ${comparison.speedup.toFixed(2)}x`);
    console.log(`📈 Improvement:  ${comparison.improvement}`);
    console.log('═══════════════════════════════════════\n');
  }

  /**
   * Generate markdown report
   */
  generateReport(comparison) {
    const report = `
# Performance Benchmark Report

**Generated:** ${new Date().toISOString()}

## Test Configuration
- **Feature Count:** ${comparison.featureCount}
- **Iterations:** ${comparison.sequential.results.length}
- **Browser:** ${navigator.userAgent}

## Results

### Sequential Encryption
- **Average Total Time:** ${comparison.sequential.avgTotalTime.toFixed(2)}ms
- **Average Per Feature:** ${comparison.sequential.avgPerFeature.toFixed(2)}ms

### Parallel Encryption (Web Workers)
- **Average Total Time:** ${comparison.parallel.avgTotalTime.toFixed(2)}ms
- **Average Per Feature:** ${comparison.parallel.avgPerFeature.toFixed(2)}ms

### Performance Improvement
- **Speedup:** ${comparison.speedup.toFixed(2)}x
- **Improvement:** ${comparison.improvement}

## Detailed Results

### Sequential Tests
${comparison.sequential.results.map((r, i) => 
  `${i + 1}. ${r.totalTime.toFixed(2)}ms (${r.perFeatureTime.toFixed(2)}ms per feature)`
).join('\n')}

### Parallel Tests
${comparison.parallel.results.map((r, i) => 
  `${i + 1}. ${r.totalTime.toFixed(2)}ms (${r.perFeatureTime.toFixed(2)}ms per feature)`
).join('\n')}

## System Information
- **CPU Cores:** ${navigator.hardwareConcurrency}
- **User Agent:** ${navigator.userAgent}
`;
    
    return report.trim();
  }

  /**
   * Export results as JSON
   */
  exportJSON(comparison) {
    return JSON.stringify({
      ...comparison,
      metadata: {
        timestamp: new Date().toISOString(),
        userAgent: navigator.userAgent,
        cpuCores: navigator.hardwareConcurrency
      }
    }, null, 2);
  }
}

export default PerformanceBenchmark;
