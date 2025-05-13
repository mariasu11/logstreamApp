package worker

import (
	"context"
	"sync"
	"time"

	"github.com/mariasu11/logstreamApp/internal/metrics"
)

// Pool represents a worker pool that processes jobs concurrently
type Pool struct {
	workers  int
	jobs     chan Job
	wg       sync.WaitGroup
	metrics  *metrics.Metrics
	stopOnce sync.Once
}

// Job is a function that should be executed by a worker
type Job func()

// NewPool creates a new worker pool with the specified number of workers
func NewPool(workers int) *Pool {
	// Ensure at least one worker
	if workers < 1 {
		workers = 1
	}
	
	return &Pool{
		workers: workers,
		jobs:    make(chan Job, workers*100), // Buffer size is 100x workers
		metrics: metrics.GetMetrics(),
	}
}

// Start starts the worker pool
func (p *Pool) Start(ctx context.Context) {
	p.metrics.WorkersActive.Set(float64(p.workers))
	
	// Start workers
	for i := 0; i < p.workers; i++ {
		p.wg.Add(1)
		go p.worker(ctx, i)
	}
}

// worker is the main worker goroutine
func (p *Pool) worker(ctx context.Context, id int) {
	defer p.wg.Done()
	
	for {
		select {
		case <-ctx.Done():
			// Context is cancelled, exit
			return
		case job, ok := <-p.jobs:
			if !ok {
				// Channel closed, exit
				return
			}
			
			// Process the job
			p.processJob(job)
		}
	}
}

// processJob executes a job and records metrics
func (p *Pool) processJob(job Job) {
	start := time.Now()
	
	// Execute the job
	func() {
		defer func() {
			if r := recover(); r != nil {
				// Log the panic
				p.metrics.WorkItemsErrored.Inc()
			}
		}()
		job()
	}()
	
	// Record metrics
	p.metrics.WorkItemsProcessed.Inc()
	p.metrics.WorkerProcessingTime.Observe(time.Since(start).Seconds())
}

// Submit adds a job to the worker pool
func (p *Pool) Submit(job Job) {
	select {
	case p.jobs <- job:
		// Job submitted successfully
		p.metrics.WorkQueueSize.Inc()
	default:
		// Queue is full, log error
		p.metrics.WorkItemsErrored.Inc()
	}
}

// Stop gracefully shuts down the worker pool
func (p *Pool) Stop(ctx context.Context) {
	p.stopOnce.Do(func() {
		// Close the jobs channel to signal workers to exit
		close(p.jobs)
		
		// Wait for all workers to finish with a timeout
		done := make(chan struct{})
		go func() {
			p.wg.Wait()
			close(done)
		}()
		
		select {
		case <-done:
			// All workers exited cleanly
		case <-ctx.Done():
			// Timeout reached, some workers may still be running
		}
		
		p.metrics.WorkersActive.Set(0)
	})
}

// Metrics returns statistics about the worker pool
func (p *Pool) Metrics() map[string]interface{} {
	return map[string]interface{}{
		"workers":       p.workers,
		"queue_size":    len(p.jobs),
		"queue_capacity": cap(p.jobs),
	}
}

// BatchProcessor handles batch processing using a worker pool
type BatchProcessor struct {
	pool      *Pool
	batchSize int
	ticker    *time.Ticker
	items     []interface{}
	mutex     sync.Mutex
	processor func([]interface{})
}

// NewBatchProcessor creates a new batch processor
func NewBatchProcessor(pool *Pool, batchSize int, interval time.Duration, processor func([]interface{})) *BatchProcessor {
	return &BatchProcessor{
		pool:      pool,
		batchSize: batchSize,
		ticker:    time.NewTicker(interval),
		items:     make([]interface{}, 0, batchSize),
		processor: processor,
	}
}

// Start starts the batch processor
func (b *BatchProcessor) Start(ctx context.Context) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				b.ticker.Stop()
				b.flush()
				return
			case <-b.ticker.C:
				b.flush()
			}
		}
	}()
}

// Add adds an item to the batch
func (b *BatchProcessor) Add(item interface{}) {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	
	b.items = append(b.items, item)
	
	if len(b.items) >= b.batchSize {
		// Batch is full, process it
		b.processCurrentBatch()
	}
}

// flush processes any pending items
func (b *BatchProcessor) flush() {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	
	if len(b.items) > 0 {
		b.processCurrentBatch()
	}
}

// processCurrentBatch processes the current batch of items
func (b *BatchProcessor) processCurrentBatch() {
	// Copy items to avoid race conditions
	batch := make([]interface{}, len(b.items))
	copy(batch, b.items)
	
	// Clear the current batch
	b.items = b.items[:0]
	
	// Submit batch processing job to worker pool
	b.pool.Submit(func() {
		b.processor(batch)
	})
}
