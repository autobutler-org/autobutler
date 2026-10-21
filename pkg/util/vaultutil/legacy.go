package vaultutil

import (
	"log/slog"
	"os"
	"path/filepath"
	"slices"

	"github.com/autobutler-org/quark/internal/db"
	"github.com/autobutler-org/quark/pkg/util/storageutil"
)

// The kinds of household vault an earlier Quark can leave behind (#3616).
const (
	// LegacyVaultKept is the copy the upgrade made of the household vault in
	// quark.db before migration 038 dropped its tables.
	LegacyVaultKept = "kept"
	// LegacyVaultOnDrive is the vault.db an older Quark kept on an external
	// drive once the household vault was moved there.
	LegacyVaultOnDrive = "drive"
)

// LegacyVault is a household vault from before per-account vaults. Nothing
// carries it into an account's vault: its entries are encrypted under the old
// shared master password, and the new vaults under keys the Quark never sees.
// It is listed so nobody loses it without knowing, and never deleted.
type LegacyVault struct {
	// Kind is LegacyVaultKept or LegacyVaultOnDrive.
	Kind string `json:"kind"`
	// Path is the file on this Quark.
	Path string `json:"path"`
	// Readable is false for a file that is not a database Quark can count;
	// it is listed all the same, with no counts.
	Readable bool `json:"readable"`
	// Entries and Folders count what it holds.
	Entries int64 `json:"entries"`
	Folders int64 `json:"folders"`
	// DeviceName names the drive a LegacyVaultOnDrive file is on.
	DeviceName string `json:"deviceName,omitempty"`
	// DeviceSerial is that drive's serial or, for a kept copy, the drive the
	// household vault had been moved to.
	DeviceSerial string `json:"deviceSerial,omitempty"`
}

// ManagedDevices lists the drives Quark manages; *storageutil.StorageService
// is one.
type ManagedDevices interface {
	GetManagedDevices() ([]storageutil.ManagedDevice, error)
}

// ListLegacyVaultsParams says where to look.
type ListLegacyVaultsParams struct {
	// DataDir is this Quark's data directory, where quark.db and the kept
	// copies are.
	DataDir string
	// Storage lists the drives whose data directories may hold a vault.db.
	// Nil looks at DataDir only.
	Storage ManagedDevices
}

// ListLegacyVaultsResult is every household vault found.
type ListLegacyVaultsResult struct {
	// Vaults lists the kept copies first, oldest first, then each drive's
	// vault.db. It is empty, never nil, when there are none.
	Vaults []LegacyVault `json:"vaults"`
}

// ListLegacyVaults finds the household vaults an earlier Quark left on this
// device, and changes none of them. A drive that cannot be listed is logged
// and skipped, so the copies on this Quark are still reported.
func ListLegacyVaults(params ListLegacyVaultsParams) (ListLegacyVaultsResult, error) {
	vaults := []LegacyVault{}
	kept, err := filepath.Glob(filepath.Join(params.DataDir, db.HouseholdVaultFilePattern))
	if err != nil {
		return ListLegacyVaultsResult{}, err
	}
	slices.Sort(kept)
	for _, path := range kept {
		vaults = append(vaults, inspectLegacyVault(LegacyVault{Kind: LegacyVaultKept, Path: path}))
	}

	var devices []storageutil.ManagedDevice
	if params.Storage != nil {
		devices, err = params.Storage.GetManagedDevices()
		if err != nil {
			slog.Warn("could not list the drives to look for a household vault on", "error", err)
		}
	}
	for _, device := range devices {
		path := filepath.Join(device.DataDir, db.LegacyDriveVaultFileName)
		if _, err := os.Stat(path); err != nil {
			continue
		}
		vault := LegacyVault{Kind: LegacyVaultOnDrive, Path: path, DeviceName: device.Name}
		if device.UsbInfo != nil {
			vault.DeviceSerial = device.UsbInfo.GetSerial()
		}
		vaults = append(vaults, inspectLegacyVault(vault))
	}
	return ListLegacyVaultsResult{Vaults: vaults}, nil
}

// inspectLegacyVault fills in what the file at vault.Path holds. A file that
// cannot be read is still returned, marked unreadable.
func inspectLegacyVault(vault LegacyVault) LegacyVault {
	summary, err := db.InspectHouseholdVault(vault.Path)
	if err != nil {
		slog.Warn("found a household vault file that could not be read", "path", vault.Path, "error", err)
		return vault
	}
	vault.Readable = true
	vault.Entries = summary.Entries
	vault.Folders = summary.Folders
	if vault.DeviceSerial == "" {
		vault.DeviceSerial = summary.DeviceSerial
	}
	return vault
}
