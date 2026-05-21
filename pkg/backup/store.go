package backup

import (
	"context"
	"fmt"
	"sync"
)

type InMemoryBackupJobStore struct {
	mu   sync.RWMutex
	jobs map[string]*BackupJob
}

func NewInMemoryBackupJobStore() *InMemoryBackupJobStore {
	return &InMemoryBackupJobStore{jobs: make(map[string]*BackupJob)}
}

func (s *InMemoryBackupJobStore) Create(_ context.Context, job *BackupJob) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.jobs[job.ID]; exists {
		return fmt.Errorf("backup job %s already exists", job.ID)
	}
	s.jobs[job.ID] = job
	return nil
}

func (s *InMemoryBackupJobStore) Get(_ context.Context, jobID string) (*BackupJob, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	job, exists := s.jobs[jobID]
	if !exists {
		return nil, fmt.Errorf("backup job %s not found", jobID)
	}
	return job, nil
}

func (s *InMemoryBackupJobStore) Update(_ context.Context, job *BackupJob) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.jobs[job.ID]; !exists {
		return fmt.Errorf("backup job %s not found", job.ID)
	}
	s.jobs[job.ID] = job
	return nil
}

func (s *InMemoryBackupJobStore) List(_ context.Context) ([]*BackupJob, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	jobs := make([]*BackupJob, 0, len(s.jobs))
	for _, job := range s.jobs {
		jobs = append(jobs, job)
	}
	return jobs, nil
}
