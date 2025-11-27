/**
 * Worker Pool for benchmarking parallel encryption
 */

export class BenchmarkWorkerPool {
  private workers: Worker[] = [];
  private busy: boolean[] = [];

  constructor(size: number) {
    for (let i = 0; i < size; i++) {
      const worker = new Worker(
        new URL('../workers/benchmark.worker.ts', import.meta.url)
      );
      this.workers.push(worker);
      this.busy[i] = false;
    }
  }

  async init(publicKey: Uint8Array): Promise<void> {
    // Initialize all workers
    const initPromises = this.workers.map(
      worker =>
        new Promise<void>((resolve, reject) => {
          const timeout = setTimeout(() => reject(new Error('Worker init timeout')), 10000);
          
          worker.onmessage = (e) => {
            if (e.data.cmd === 'init' && e.data.success) {
              clearTimeout(timeout);
              // Set public key
              worker.postMessage({
                cmd: 'setPublicKey',
                data: { publicKey }
              });
            } else if (e.data.cmd === 'setPublicKey' && e.data.success) {
              resolve();
            } else if (e.data.cmd === 'error') {
              clearTimeout(timeout);
              reject(new Error(e.data.error));
            }
          };

          worker.postMessage({ cmd: 'init' });
        })
    );

    await Promise.all(initPromises);
  }

  async encryptParallel(values: number[]): Promise<{
    results: Uint8Array[];
    totalTime: number;
    perFeatureTimes: number[];
  }> {
    const startTotal = performance.now();
    const results: Uint8Array[] = new Array(values.length);
    const perFeatureTimes: number[] = new Array(values.length);

    const promises = values.map((value, index) => {
      return new Promise<void>((resolve, reject) => {
        const workerIndex = index % this.workers.length;
        const worker = this.workers[workerIndex];

        const messageHandler = (e: MessageEvent) => {
          if (e.data.cmd === 'encrypt' && e.data.index === index) {
            results[index] = e.data.result;
            perFeatureTimes[index] = e.data.time;
            worker.removeEventListener('message', messageHandler);
            resolve();
          } else if (e.data.cmd === 'error') {
            worker.removeEventListener('message', messageHandler);
            reject(new Error(e.data.error));
          }
        };

        worker.addEventListener('message', messageHandler);
        worker.postMessage({
          cmd: 'encrypt',
          data: { value, index }
        });
      });
    });

    await Promise.all(promises);
    const totalTime = performance.now() - startTotal;

    return { results, totalTime, perFeatureTimes };
  }

  terminate(): void {
    this.workers.forEach(w => w.terminate());
  }
}
