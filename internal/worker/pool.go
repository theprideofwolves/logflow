package worker

import (
	"context"
	"sync"
)

// Job represents a single unit of work for our log workers.
type Job struct {
	// Raw JSON log payload to be processed by the worker
	Payload []byte
}

// ProcessFunc is the callback signature workers use to handle each job.
// This allows external logic (like the ThreatDetector) to be injected
// without creating circular package dependencies.
type ProcessFunc func(payload []byte)

// Worker represents the execution unit that processes jobs.
type Worker struct {
	ID        int
	JobQueue  <-chan Job
	WaitGroup *sync.WaitGroup
	Ctx       context.Context
	Process   ProcessFunc
}

// Start initiates the worker's processing loop.
// It listens on the JobQueue for new jobs and executes them.
func (w *Worker) Start() {
	go func() {
		// Signal completion when the goroutine exits
		defer w.WaitGroup.Done()
		
		for {
			select {
			case job, ok := <-w.JobQueue:
				if !ok {
					// The channel has been closed, time to shut down the worker
					return
				}
				// Hand off to the injected processing logic
				if w.Process != nil {
					w.Process(job.Payload)
				}
			case <-w.Ctx.Done():
				// Context got cancelled, stop the worker
				return
			}
		}
	}()
}



// Dispatcher manages the worker pool and distributes incoming jobs.
type Dispatcher struct {
	MaxWorkers  int
	JobQueue    chan Job
	ProcessFunc ProcessFunc
	workers     []*Worker
	wg          sync.WaitGroup
	ctx         context.Context
	cancel      context.CancelFunc
}

// NewDispatcher initializes a dispatcher with a specified worker count, job queue size,
// and a processing function that each worker will call for every job.
func NewDispatcher(maxWorkers, queueSize int, fn ProcessFunc) *Dispatcher {
	ctx, cancel := context.WithCancel(context.Background())
	return &Dispatcher{
		MaxWorkers:  maxWorkers,
		JobQueue:    make(chan Job, queueSize),
		ProcessFunc: fn,
		workers:     make([]*Worker, maxWorkers),
		ctx:         ctx,
		cancel:      cancel,
	}
}

// Start creates and launches all the workers defined in the dispatcher.
func (d *Dispatcher) Start() {
	for i := 0; i < d.MaxWorkers; i++ {
		worker := &Worker{
			ID:        i + 1,
			JobQueue:  d.JobQueue,
			WaitGroup: &d.wg,
			Ctx:       d.ctx,
			Process:   d.ProcessFunc,
		}
		d.workers[i] = worker
		
		d.wg.Add(1)
		worker.Start()
	}
}

// Submit allows handing off a new Job to the worker pool via the JobQueue.
func (d *Dispatcher) Submit(job Job) {
	d.JobQueue <- job
}

// Stop initiates a graceful shutdown of all workers, waiting for them to complete.
func (d *Dispatcher) Stop() {
	// Closing the channel signals workers to exit once they finish their current job
	close(d.JobQueue)
	// Wait for all workers to finish
	d.wg.Wait()
	// Clean up dispatcher context
	d.cancel()
}
