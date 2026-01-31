package v1_migration

import (
	"autobutler/pkg/migration"
	"autobutler/pkg/util/serverutil"
	"fmt"

	"github.com/gin-gonic/gin"
)

// Package-level service instance
var (
	takeoutService migration.GoogleTakeoutService
	jobWorker      *JobWorker
)

func init() {
	// Initialize service with real implementations
	client := migration.NewUploadBasedClient()
	store := migration.NewInMemoryJobStore()
	uploader := migration.NewCirrusFileUploader("")
	extractor := migration.NewZipExtractor()

	takeoutService = migration.NewService(client, store, uploader, extractor)

	// Start background worker
	jobWorker = NewJobWorker(takeoutService)
	jobWorker.Start()
}

type StartImportRequest struct {
	Services struct {
		Photos   bool `json:"photos"`
		Drive    bool `json:"drive"`
		Contacts bool `json:"contacts"`
		Calendar bool `json:"calendar"`
	} `json:"services"`
}

var googleStartRoute = serverutil.ApiRoute(
	"POST", "/migration/google/start", func(c *gin.Context) *serverutil.Response {
		var req StartImportRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			return serverutil.BadRequest(fmt.Errorf("invalid request: %w", err))
		}

		// Convert services struct to slice
		services := []string{}
		if req.Services.Photos {
			services = append(services, "photos")
		}
		if req.Services.Drive {
			services = append(services, "drive")
		}
		if req.Services.Contacts {
			services = append(services, "contacts")
		}
		if req.Services.Calendar {
			services = append(services, "calendar")
		}

		if len(services) == 0 {
			return serverutil.BadRequest(fmt.Errorf("no services selected"))
		}

		// Start import job
		job, err := takeoutService.StartImport(c.Request.Context(), services)
		if err != nil {
			return serverutil.InternalServerError(fmt.Errorf("failed to start import: %w", err))
		}

		// Queue job for background processing
		jobWorker.QueueJob(job.ID)

		return serverutil.Ok().WithData(gin.H{
			"jobId":    job.ID,
			"exportId": job.ExportID,
			"message":  "Import job created. You can now upload your Google Takeout archives.",
			"status":   job.Status,
		})
	},
)

var googleStatusRoute = serverutil.ApiRoute(
	"GET", "/migration/google/status/:jobId", func(c *gin.Context) *serverutil.Response {
		jobID := c.Param("jobId")

		job, err := takeoutService.GetImportStatus(c.Request.Context(), jobID)
		if err != nil {
			return serverutil.BadRequest(fmt.Errorf("job not found: %w", err))
		}

		return serverutil.Ok().WithData(job)
	},
)
