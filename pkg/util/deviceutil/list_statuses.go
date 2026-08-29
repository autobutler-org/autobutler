package deviceutil

import (
	"context"

	"github.com/autobutler-org/quark/internal/db"
	"github.com/autobutler-org/quark/pkg/util/storageutil"
)

// ListStatusesParams reports every detected device with the display name and
// role the database holds for it.
type ListStatusesParams struct {
	// Ctx bounds the database reads.
	Ctx context.Context
	// Storage detects the devices.
	Storage *storageutil.StorageService
	// Database holds the names and roles to overlay. Nil leaves the detector's
	// own names in place and reports no roles at all.
	Database *db.DatabaseSqlc
}

// ListStatusesResult carries the assembled statuses.
type ListStatusesResult struct {
	Statuses []*storageutil.DeviceStatus
}

// ListStatuses assembles the device list the storage UI renders. A database
// that cannot be read is not an error: the statuses still describe the
// hardware, which is the half a client cannot get anywhere else.
func ListStatuses(params ListStatusesParams) (ListStatusesResult, error) {
	statuses, err := params.Storage.GetDeviceStatuses()
	if err != nil {
		return ListStatusesResult{}, err
	}

	// Overlay display names and roles from DB, keyed by device serial.
	if params.Database != nil {
		ctx := params.Ctx

		nameMap := make(map[string]string)
		if names, err := params.Database.Queries.GetAllDeviceNames(ctx); err == nil {
			for _, n := range names {
				nameMap[n.DeviceSerial] = n.DisplayName
			}
		}

		roleMap := make(map[string]string)
		if roles, err := params.Database.Queries.GetAllDeviceRoles(ctx); err == nil {
			for _, r := range roles {
				roleMap[r.DeviceSerial] = r.Role
			}
		}

		for i := range statuses {
			serial := ""
			if statuses[i].UsbInfo != nil {
				serial = statuses[i].UsbInfo.GetSerial()
			}
			if name, ok := nameMap[serial]; ok {
				statuses[i].Name = name
			}
			if role, ok := roleMap[serial]; ok {
				statuses[i].Role = role
			} else {
				statuses[i].Role = RoleUnassigned
			}
		}
	}

	return ListStatusesResult{Statuses: statuses}, nil
}
