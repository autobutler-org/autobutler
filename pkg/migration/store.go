package migration

import (
	"context"
	"fmt"
	"sync"
)

// InMemoryJobStore is an in-memory implementation of ImportJobStore
type InMemoryJobStore struct {
	mu   sync.RWMutex
	jobs map[string]*ImportJob
}

// NewInMemoryJobStore creates a new in-memory job store
func NewInMemoryJobStore() *InMemoryJobStore {
	return &InMemoryJobStore{
		jobs: make(map[string]*ImportJob),
	}
}

// Create creates a new import job
func (s *InMemoryJobStore) Create(ctx context.Context, job *ImportJob) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.jobs[job.ID]; exists {
		return fmt.Errorf("job %s already exists", job.ID)
	}

	s.jobs[job.ID] = job
	return nil
}

// Get retrieves an import job by ID
func (s *InMemoryJobStore) Get(ctx context.Context, jobID string) (*ImportJob, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	job, exists := s.jobs[jobID]
	if !exists {
		return nil, fmt.Errorf("job %s not found", jobID)
	}

	return job, nil
}

// Update updates an existing import job
func (s *InMemoryJobStore) Update(ctx context.Context, job *ImportJob) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.jobs[job.ID]; !exists {
		return fmt.Errorf("job %s not found", job.ID)
	}

	s.jobs[job.ID] = job
	return nil
}

// List returns all import jobs
func (s *InMemoryJobStore) List(ctx context.Context) ([]*ImportJob, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	jobs := make([]*ImportJob, 0, len(s.jobs))
	for _, job := range s.jobs {
		jobs = append(jobs, job)
	}

	return jobs, nil
}
