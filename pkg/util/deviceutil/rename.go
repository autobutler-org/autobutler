package deviceutil

import (
	"context"
	"strings"
	"unicode"

	"github.com/autobutler-org/quark/internal/db"
)

// MaxDeviceNameLength caps a display name in runes, not bytes, so a name of
// non-ASCII characters is measured the way the person typing it sees it.
const MaxDeviceNameLength = 64

// RenameParams sets the display name a device is shown under.
type RenameParams struct {
	// Ctx bounds the write.
	Ctx context.Context
	// Queries holds the device_names rows.
	Queries *db.Queries
	// Serial identifies the device, empty for the internal one.
	Serial string
	// Name is the requested display name, before trimming.
	Name string
}

// RenameResult reports the name that was stored, which is the requested one
// with surrounding whitespace removed.
type RenameResult struct {
	Name string
}

// Rename stores a display name for a device after checking it is something a
// UI can show. A name that fails any rule comes back as [ErrInvalidDeviceName].
func Rename(params RenameParams) (RenameResult, error) {
	name := strings.TrimSpace(params.Name)
	if name == "" {
		return RenameResult{}, ErrInvalidDeviceName
	}
	if len([]rune(name)) > MaxDeviceNameLength {
		return RenameResult{}, ErrInvalidDeviceName
	}
	for _, r := range name {
		if unicode.IsControl(r) {
			return RenameResult{}, ErrInvalidDeviceName
		}
	}

	if err := params.Queries.UpsertDeviceName(params.Ctx, db.UpsertDeviceNameParams{
		DeviceSerial: params.Serial,
		DisplayName:  name,
	}); err != nil {
		return RenameResult{}, err
	}

	return RenameResult{Name: name}, nil
}
