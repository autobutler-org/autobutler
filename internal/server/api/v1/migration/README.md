# Google Takeout Migration - Testing Documentation

This document describes the testing infrastructure for the Google Takeout import functionality.

## Overview

The Google Takeout import system is designed with testability in mind, using interfaces and dependency injection to enable comprehensive unit and integration testing.

## Architecture

### Core Components

1. **GoogleTakeoutClient** - Interface for Google Takeout API interactions
2. **GoogleTakeoutService** - Business logic for import orchestration
3. **ImportJobStore** - Persistence layer for import job tracking
4. **FileUploader** - Handles uploading files to cirrus storage
5. **ArchiveExtractor** - Extracts zip archives from Google Takeout

### Import Flow

```
1. User initiates import via API
2. Service calls GoogleTakeoutClient.RequestExport()
3. Background worker polls for export completion
4. When complete, downloads archive files
5. Extracts archives to temporary directory
6. Uploads extracted files to cirrus storage
7. Updates job status to completed
```

## Test Structure

### Unit Tests

Located in: `pkg/migration/google_takeout_test.go`

Tests cover:
- ✅ **StartImport** - Service initialization and validation
- ✅ **GetImportStatus** - Status retrieval
- ✅ **ProcessImport** - Full import workflow
- ✅ **Error Handling** - All failure scenarios
- ✅ **Progress Tracking** - Progress updates throughout the process

**Run unit tests:**
```bash
go test -v ./pkg/migration/
```

**Run with coverage:**
```bash
go test -v -cover ./pkg/migration/
```

### Integration Tests

Located in: `internal/server/api/v1/migration/google_migration_test.go`

Tests cover:
- ✅ **API Endpoints** - HTTP request/response handling
- ✅ **Request Validation** - Invalid input handling
- ✅ **Full Flow** - End-to-end import simulation
- ✅ **Status Polling** - Frontend polling simulation

**Run integration tests:**
```bash
go test -v ./internal/server/api/v1/migration/
```

**Skip long-running tests:**
```bash
go test -short -v ./internal/server/api/v1/migration/
```

## Mock Objects

Located in: `pkg/migration/mocks.go`

### MockGoogleTakeoutClient

Simulates Google Takeout API with configurable behavior:

```go
mockClient := migration.NewMockGoogleTakeoutClient()

// Configure errors
mockClient.RequestExportError = errors.New("API error")

// Configure timing
mockClient.ExportProcessingDelay = 5 * time.Second

// Manually control export status
mockClient.SetExportCompleted(exportID)
mockClient.SetExportFailed(exportID, "reason")
```

### MockImportJobStore

In-memory storage for testing:

```go
mockStore := migration.NewMockImportJobStore()

// Configure errors
mockStore.CreateError = errors.New("database error")
```

### MockFileUploader

Tracks uploaded files without actual I/O:

```go
mockUploader := migration.NewMockFileUploader()

// After test, verify uploads
uploadCount := mockUploader.GetUploadCount()
```

### MockArchiveExtractor

Simulates archive extraction:

```go
mockExtractor := migration.NewMockArchiveExtractor()

// Configure mock contents
mockExtractor.MockContents = []string{"file1.jpg", "file2.jpg"}

// After test, verify extractions
extractCount := mockExtractor.GetExtractCount()
```

## Test Scenarios

### Successful Import

```go
func TestSuccessfulImport(t *testing.T) {
    // Setup mocks
    mockClient := migration.NewMockGoogleTakeoutClient()
    mockStore := migration.NewMockImportJobStore()
    mockUploader := migration.NewMockFileUploader()
    mockExtractor := migration.NewMockArchiveExtractor()
    
    service := migration.NewGoogleTakeoutService(
        mockClient, mockStore, mockUploader, mockExtractor,
    )
    
    // Start import
    job, _ := service.StartImport(ctx, []string{"photos"})
    
    // Complete export
    mockClient.SetExportCompleted(job.ExportID)
    
    // Process
    service.ProcessImport(ctx, job.ID)
    
    // Verify
    finalJob, _ := service.GetImportStatus(ctx, job.ID)
    assert.Equal(t, ImportStatusCompleted, finalJob.Status)
}
```

### Error Scenarios

The test suite covers:

1. **Google API Failures**
   - Export request fails
   - Export status check fails
   - Download fails

2. **Storage Failures**
   - Database errors
   - Upload errors
   - Extraction errors

3. **Validation Failures**
   - No services selected
   - Invalid job ID
   - Corrupted archives

4. **Timeout Scenarios**
   - Export takes too long
   - Context cancellation

## Implementation Checklist

When implementing the actual functionality, ensure:

- [ ] Implement real GoogleTakeoutClient using Google API
- [ ] Implement persistent ImportJobStore (database)
- [ ] Implement FileUploader using existing cirrus upload logic
- [ ] Implement ArchiveExtractor using archive/zip package
- [ ] Add background worker for ProcessImport
- [ ] Add API endpoints for status polling
- [ ] Add proper error logging and metrics
- [ ] Add retry logic for transient failures
- [ ] Add rate limiting for Google API calls
- [ ] Add cleanup of temporary files
- [ ] Add webhook/notification on completion
- [ ] Add cancellation support

## Performance Considerations

### Benchmarks

Run benchmarks to ensure performance:

```bash
go test -bench=. -benchmem ./pkg/migration/
```

Current benchmarks:
- `BenchmarkGoogleTakeoutService_StartImport` - Measures job creation overhead
- `BenchmarkGoogleTakeoutService_ProcessImport` - Measures full import flow

### Optimization Opportunities

1. **Concurrent Archive Processing** - Process multiple archives in parallel
2. **Streaming Upload** - Stream files directly from extraction to upload
3. **Incremental Progress** - Update progress more granularly
4. **Archive Validation** - Validate before extraction
5. **Deduplication** - Skip already-imported files

## Future Enhancements

### Additional Tests Needed

- [ ] Concurrent job processing
- [ ] Large file handling (>1GB archives)
- [ ] Network interruption recovery
- [ ] Partial import resume
- [ ] Rate limiting behavior
- [ ] Memory usage profiling
- [ ] Database transaction handling

### Monitoring & Observability

- [ ] Add OpenTelemetry tracing
- [ ] Add Prometheus metrics (job duration, success rate, etc.)
- [ ] Add structured logging with context
- [ ] Add alerting for failures

## Running All Tests

```bash
# All tests
make test

# With coverage report
make test-coverage

# Integration tests only
go test -v -run Integration ./...

# Specific test
go test -v -run TestGoogleTakeoutService_StartImport ./pkg/migration/
```

## Contributing

When adding new functionality:

1. Write tests first (TDD)
2. Ensure all existing tests pass
3. Add integration tests for API changes
4. Update this documentation
5. Add benchmarks for performance-critical code

## References

- [Google Takeout API Documentation](https://developers.google.com/takeout)
- [Gin Testing Guide](https://github.com/gin-gonic/gin#testing)
- [Go Testing Best Practices](https://golang.org/doc/tutorial/add-a-test)
