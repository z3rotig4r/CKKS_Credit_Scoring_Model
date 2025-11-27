/**
 * Worker Pool Manager for Parallel CKKS Encryption
 * 
 * Manages a pool of Web Workers for parallel encryption of credit scoring features.
 * Provides automatic load balancing and task queuing.
 */

export class WorkerPool {
  constructor(workerPath, poolSize = 5) {
    this.workerPath = workerPath;
    this.poolSize = poolSize;
    this.workers = [];
    this.busyWorkers = new Set();
    this.taskQueue = [];
    this.taskId = 0;
    
    this.initializePool();
  }

  /**
   * Initialize worker pool
   */
  initializePool() {
    for (let i = 0; i < this.poolSize; i++) {
      const worker = new Worker(this.workerPath);
      worker.workerId = i;
      worker.onmessage = (e) => this.handleMessage(i, e);
      worker.onerror = (e) => this.handleError(i, e);
      
      this.workers.push(worker);
    }
    
    console.log(`[WorkerPool] Initialized ${this.poolSize} workers`);
  }

  /**
   * Execute a task in the worker pool
   * @param {string} cmd - Command to execute ('encrypt', 'decrypt', etc.)
   * @param {Object} data - Data for the task
   * @returns {Promise} Resolves with the result
   */
  async execute(cmd, data) {
    return new Promise((resolve, reject) => {
      const id = this.taskId++;
      const task = { id, cmd, data, resolve, reject };
      
      // Try to find a free worker
      const freeWorkerIndex = this.workers.findIndex((_, i) => !this.busyWorkers.has(i));
      
      if (freeWorkerIndex !== -1) {
        // Execute immediately if worker available
        this.executeTask(freeWorkerIndex, task);
      } else {
        // Queue task if all workers busy
        this.taskQueue.push(task);
      }
    });
  }

  /**
   * Execute a task on a specific worker
   */
  executeTask(workerIndex, task) {
    this.busyWorkers.add(workerIndex);
    this.workers[workerIndex].currentTask = task;
    this.workers[workerIndex].postMessage({
      id: task.id,
      cmd: task.cmd,
      data: task.data
    });
  }

  /**
   * Handle message from worker
   */
  handleMessage(workerIndex, event) {
    const { id, cmd, result, success, error } = event.data;
    const worker = this.workers[workerIndex];
    const task = worker.currentTask;
    
    if (!task || task.id !== id) {
      console.warn(`[WorkerPool] Received message for unknown task ${id}`);
      return;
    }
    
    // Mark worker as free
    this.busyWorkers.delete(workerIndex);
    worker.currentTask = null;
    
    // Resolve or reject the task
    if (success) {
      task.resolve(result);
    } else {
      task.reject(new Error(error || 'Worker task failed'));
    }
    
    // Process next queued task if any
    this.processQueue(workerIndex);
  }

  /**
   * Handle worker error
   */
  handleError(workerIndex, error) {
    console.error(`[WorkerPool] Worker ${workerIndex} error:`, error);
    
    const worker = this.workers[workerIndex];
    const task = worker.currentTask;
    
    if (task) {
      task.reject(new Error(`Worker error: ${error.message}`));
      this.busyWorkers.delete(workerIndex);
      worker.currentTask = null;
    }
    
    // Restart worker
    this.restartWorker(workerIndex);
  }

  /**
   * Process queued tasks
   */
  processQueue(workerIndex) {
    if (this.taskQueue.length > 0) {
      const nextTask = this.taskQueue.shift();
      this.executeTask(workerIndex, nextTask);
    }
  }

  /**
   * Restart a crashed worker
   */
  restartWorker(workerIndex) {
    console.log(`[WorkerPool] Restarting worker ${workerIndex}`);
    
    // Terminate old worker
    this.workers[workerIndex].terminate();
    
    // Create new worker
    const worker = new Worker(this.workerPath);
    worker.workerId = workerIndex;
    worker.onmessage = (e) => this.handleMessage(workerIndex, e);
    worker.onerror = (e) => this.handleError(workerIndex, e);
    
    this.workers[workerIndex] = worker;
  }

  /**
   * Terminate all workers
   */
  terminate() {
    console.log('[WorkerPool] Terminating all workers');
    
    this.workers.forEach(worker => worker.terminate());
    this.workers = [];
    this.busyWorkers.clear();
    this.taskQueue = [];
  }

  /**
   * Get pool status
   */
  getStatus() {
    return {
      poolSize: this.poolSize,
      busyWorkers: this.busyWorkers.size,
      freeWorkers: this.poolSize - this.busyWorkers.size,
      queuedTasks: this.taskQueue.length
    };
  }
}

/**
 * Singleton instance for the application
 */
let workerPoolInstance = null;

export function getWorkerPool() {
  if (!workerPoolInstance) {
    const workerPath = new URL('../workers/ckks.worker.js', import.meta.url);
    workerPoolInstance = new WorkerPool(workerPath, 5);
  }
  return workerPoolInstance;
}

export function terminateWorkerPool() {
  if (workerPoolInstance) {
    workerPoolInstance.terminate();
    workerPoolInstance = null;
  }
}
