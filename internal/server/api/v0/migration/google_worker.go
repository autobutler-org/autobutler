package v0_migration

import (
	"context"
	"log"
	"sync"

	"github.com/autobutler-org/autobutler/pkg/migration"

	"golang.org/x/oauth2"
)

// GoogleAPIJobWorker processes import jobs using Google APIs
type GoogleAPIJobWorker struct {
	service *migration.GoogleAPIService
	queue   chan *JobRequest
	wg      sync.WaitGroup
	stop    chan struct{}
}

// JobRequest contains the job ID and authentication info
type JobRequest struct {
	JobID string
	Email string
	Token *oauth2.Token
}

// NewGoogleAPIJobWorker creates a new job worker for Google API downloads
func NewGoogleAPIJobWorker(service *migration.GoogleAPIService) *GoogleAPIJobWorker {
	return &GoogleAPIJobWorker{
		service: service,
		queue:   make(chan *JobRequest, 100),
		stop:    make(chan struct{}),
	}
}

// Start begins processing jobs
func (w *GoogleAPIJobWorker) Start() {
	w.wg.Add(1)
	go w.run()
}

// Stop gracefully stops the worker
func (w *GoogleAPIJobWorker) Stop() {
	close(w.stop)
	w.wg.Wait()
}

// QueueJob adds a job to the processing queue
func (w *GoogleAPIJobWorker) QueueJob(jobID string, email string, token *oauth2.Token) {
	req := &JobRequest{
		JobID: jobID,
		Email: email,
		Token: token,
	}

	select {
	case w.queue <- req:
		log.Printf("Queued import job: %s for %s", jobID, email)
	default:
		log.Printf("Warning: job queue full, dropping job %s", jobID)
	}
}

func (w *GoogleAPIJobWorker) run() {
	defer w.wg.Done()

	for {
		select {
		case <-w.stop:
			log.Println("Job worker stopping...")
			return
		case req := <-w.queue:
			w.processJob(req)
		}
	}
}

func (w *GoogleAPIJobWorker) processJob(req *JobRequest) {
	log.Printf("Processing import job: %s for %s", req.JobID, req.Email)

	ctx := context.Background()
	if err := w.service.ProcessImport(ctx, req.JobID, req.Email, req.Token); err != nil {
		log.Printf("Error processing job %s: %v", req.JobID, err)
		return
	}

	log.Printf("Successfully completed import job: %s", req.JobID)
}
