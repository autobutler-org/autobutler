package v1_migration

import (
	"autobutler/pkg/migration"
	"context"
	"log"
	"sync"
)

// JobWorker processes import jobs in the background
type JobWorker struct {
	service migration.GoogleTakeoutService
	queue   chan string
	wg      sync.WaitGroup
	stop    chan struct{}
}

// NewJobWorker creates a new job worker
func NewJobWorker(service migration.GoogleTakeoutService) *JobWorker {
	return &JobWorker{
		service: service,
		queue:   make(chan string, 100),
		stop:    make(chan struct{}),
	}
}

// Start begins processing jobs
func (w *JobWorker) Start() {
	w.wg.Add(1)
	go w.run()
}

// Stop gracefully stops the worker
func (w *JobWorker) Stop() {
	close(w.stop)
	w.wg.Wait()
}

// QueueJob adds a job to the processing queue
func (w *JobWorker) QueueJob(jobID string) {
	select {
	case w.queue <- jobID:
		log.Printf("Queued import job: %s", jobID)
	default:
		log.Printf("Warning: job queue full, dropping job %s", jobID)
	}
}

func (w *JobWorker) run() {
	defer w.wg.Done()

	for {
		select {
		case <-w.stop:
			log.Println("Job worker stopping...")
			return
		case jobID := <-w.queue:
			w.processJob(jobID)
		}
	}
}

func (w *JobWorker) processJob(jobID string) {
	log.Printf("Processing import job: %s", jobID)

	ctx := context.Background()
	if err := w.service.ProcessImport(ctx, jobID); err != nil {
		log.Printf("Error processing job %s: %v", jobID, err)
		return
	}

	log.Printf("Successfully completed import job: %s", jobID)
}
