package v0_storage

import (
	"time"

	"github.com/autobutler-org/quark/pkg/util/ctxutil"
	"github.com/autobutler-org/quark/pkg/util/deputil"
	"github.com/autobutler-org/quark/pkg/util/serverutil"

	"github.com/gin-gonic/gin"
)

// ProtectionStatus is the body of GET /storage/protection.
type ProtectionStatus struct {
	// Level is unprotected, basic, or protected.
	Level string `json:"level" enums:"unprotected,basic,protected"`
	// Sentence is one plain sentence describing that level.
	Sentence string `json:"sentence"`
	// NextStep is assign-backup-role, start-backup, or keep-one-elsewhere.
	NextStep string `json:"nextStep" enums:"assign-backup-role,start-backup,keep-one-elsewhere"`
	// Drives lists every snapshot-backup drive.
	Drives []ProtectionDrive `json:"drives"`
}

// ProtectionDrive is one snapshot-backup drive in a protection status.
type ProtectionDrive struct {
	Serial string `json:"serial"`
	Name   string `json:"name,omitempty"`
	// Counts is whether this drive adds a backup copy. Partitions of one disk share a single count.
	Counts bool `json:"counts"`
	// Reason is why Counts is false. Empty when Counts is true.
	Reason string `json:"reason"`
	// LastBackedUp is when this drive's newest completed snapshot backup finished.
	// Nil when it has no completed backup.
	LastBackedUp *time.Time `json:"lastBackedUp"`
}

// getProtectionStatus godoc
// @Summary Get data protection status
// @Description Reports how protected the files are. unprotected means no snapshot-backup drive has a completed backup within 30 days, basic means one such drive after partitions of one disk are grouped, and protected means two or more. The live copy on the default-storage drive and the internal drive never count. A failed, unfinished, or stale backup does not count. A drive whose health verdict is replace-soon does not count, even inside 30 days. Watch and unknown still follow the 30-day rule. sentence is one plain sentence. nextStep is assign-backup-role, start-backup, or keep-one-elsewhere. drives lists each snapshot-backup drive, whether it counts, why not, and when it was last backed up. Refetch when a backup completes or fails (backup_completed, backup_failed).
// @Tags storage
// @Produce json
// @Success 200 {object} ProtectionStatus
// @Failure 500 {object} serverutil.Response "Internal Server Error"
// @Security BearerAuth
// @Router /storage/protection [get]
func getProtectionStatus(c *gin.Context) *serverutil.Response {
	deps, ok := ctxutil.Get[deputil.Dependencies](c, "deps")
	if !ok || deps.Database() == nil || deps.StorageService() == nil {
		return serverutil.InternalServerError(nil)
	}

	result, err := protectionStatus(c.Request.Context(), deps)
	if err != nil {
		return serverutil.InternalServerError(err)
	}
	return serverutil.Ok().WithData(protectionResponse(result))
}

var getProtectionStatusRoute = serverutil.ApiRoute(
	"GET", "/storage/protection", getProtectionStatus,
)
