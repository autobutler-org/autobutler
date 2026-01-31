package v1_migration

import (
	v1_auth "autobutler/internal/server/api/v1/auth"
	"autobutler/pkg/migration"
	"autobutler/pkg/util/serverutil"
	"autobutler/pkg/util/storageutil"
	"fmt"
	"log"

	"github.com/gin-gonic/gin"
)

// Package-level service instance
var (
	apiService *migration.GoogleAPIService
	jobWorker  *GoogleAPIJobWorker
)

func init() {
	// Initialize service with real Google API implementations
	store := migration.NewInMemoryJobStore()

	// Use system cirrus path for now (can be device-specific later)
	cirrusPath := storageutil.GetCirrusDir()
	oauthConfig := v1_auth.GetGoogleOAuthConfig()

	apiService = migration.NewGoogleAPIService(store, cirrusPath, oauthConfig)

	// Start background worker
	jobWorker = NewGoogleAPIJobWorker(apiService)
	jobWorker.Start()
}

type StartImportRequest struct {
	Email    string `json:"email"` // User's Google email
	Services struct {
		Photos   bool `json:"photos"`
		Drive    bool `json:"drive"`
		Contacts bool `json:"contacts"`
		Calendar bool `json:"calendar"`
	} `json:"services"`
	DeviceSerial string `json:"deviceSerial"` // Optional: empty string for internal drive
}

var googleStartRoute = serverutil.ApiRoute(
	"POST", "/migration/google/start", func(c *gin.Context) *serverutil.Response {
		var req StartImportRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			return serverutil.BadRequest(fmt.Errorf("invalid request: %w", err))
		}

		if req.Email == "" {
			return serverutil.BadRequest(fmt.Errorf("email is required"))
		}

		// Get OAuth token for this email
		token, exists := v1_auth.GetGoogleToken(req.Email)
		if !exists {
			return serverutil.Unauthorized(fmt.Errorf("not authenticated with Google"))
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

		log.Printf("Starting import for %s with services: %v", req.Email, services)

		// Start import job
		job, err := apiService.StartImport(c.Request.Context(), services, req.Email, token)
		if err != nil {
			return serverutil.InternalServerError(fmt.Errorf("failed to start import: %w", err))
		}

		// Queue job for background processing
		jobWorker.QueueJob(job.ID, req.Email, token)

		return serverutil.Ok().WithData(gin.H{
			"jobId":        job.ID,
			"exportId":     job.ExportID,
			"message":      "Import started. Downloading your Google data...",
			"status":       job.Status,
			"deviceSerial": req.DeviceSerial,
		})
	},
)

var googleStatusRoute = serverutil.ApiRoute(
	"GET", "/migration/google/status/:jobId", func(c *gin.Context) *serverutil.Response {
		jobID := c.Param("jobId")

		job, err := apiService.GetImportStatus(c.Request.Context(), jobID)
		if err != nil {
			return serverutil.BadRequest(fmt.Errorf("job not found: %w", err))
		}

		return serverutil.Ok().WithData(job)
	},
)
