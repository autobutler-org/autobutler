package v0_migration

import (
	"fmt"
	"log"

	v1_auth "github.com/autobutler-org/autobutler/internal/server/api/v0/auth"
	"github.com/autobutler-org/autobutler/pkg/migration"
	"github.com/autobutler-org/autobutler/pkg/util/serverutil"
	"github.com/autobutler-org/autobutler/pkg/util/storageutil"

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
	cirrusPath, err := storageutil.GetCirrusDir()
	if err != nil {
		panic(fmt.Sprintf("failed to initialize cirrus directory: %v", err))
	}
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
		Calendar bool `json:"calendar"` // UNUSED: calendar migration is not yet implemented (see #856)
	} `json:"services"`
	DeviceSerial string `json:"deviceSerial"` // Optional: empty string for internal drive
}

// startGoogleMigration godoc
// @Summary Start Google import/migration
// @Description Starts an import job for selected Google services for the given email
// @Tags migration
// @Accept json
// @Produce json
// @Param body body StartImportRequest true "Start import request"
// @Success 200 {object} object
// @Failure 400 {object} serverutil.Response "Bad Request"
// @Failure 401 {object} serverutil.Response "Unauthorized"
// @Failure 500 {object} serverutil.Response "Internal Server Error"
// @Router /migration/google/start [post]
func startGoogleMigration(c *gin.Context) *serverutil.Response {
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
	// NOTE: Calendar migration is not yet implemented. This flag is accepted
	// but the backend does not currently process calendar data. See #856.
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
}

var startGoogleMigrationRoute = serverutil.ApiRoute(
	"POST", "/migration/google/start", startGoogleMigration,
)

// getGoogleMigrationStatus godoc
// @Summary Get Google import job status
// @Description Returns the status of a previously started import job
// @Tags migration
// @Produce json
// @Param jobId path string true "Job ID"
// @Success 200 {object} object
// @Failure 400 {object} serverutil.Response "Bad Request"
// @Failure 404 {object} serverutil.Response "Not Found"
// @Router /migration/google/status/{jobId} [get]
func getGoogleMigrationStatus(c *gin.Context) *serverutil.Response {
	jobID := c.Param("jobId")

	job, err := apiService.GetImportStatus(c.Request.Context(), jobID)
	if err != nil {
		return serverutil.BadRequest(fmt.Errorf("job not found: %w", err))
	}

	return serverutil.Ok().WithData(job)
}

var getGoogleMigrationStatusRoute = serverutil.ApiRoute(
	"GET", "/migration/google/status/:jobId", getGoogleMigrationStatus,
)
