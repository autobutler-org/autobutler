package v1_files

import (
	"path/filepath"
	"sort"
	"strconv"
	"time"

	"github.com/autobutler-org/autobutler/pkg/util/ctxutil"
	"github.com/autobutler-org/autobutler/pkg/util/deputil"
	"github.com/autobutler-org/autobutler/pkg/util/serverutil"
	"github.com/autobutler-org/autobutler/pkg/util/storageutil"

	"github.com/gin-gonic/gin"
)

const defaultRecentLimit = 20
const maxRecentLimit = 200

// FileNodeWithTimeJSON extends FileNodeJSON with modification time for sorting/display
type FileNodeWithTimeJSON struct {
	FileNodeJSON
	ModifiedAt time.Time `json:"modifiedAt"`
}

// listRecentFiles godoc
// @Summary List recently uploaded files
// @Description Returns files sorted by modification time (newest first) across all managed devices.
// @Tags cirrus
// @Produce json
// @Param limit query int false "Maximum number of files to return (default 20, max 200)"
// @Param serial query []string false "Filter by device serial(s)"
// @Success 200 {array} FileNodeWithTimeJSON
// @Failure 500 {object} serverutil.Response "Internal Server Error"
// @Router /cirrus/recent [get]
func listRecentFiles(c *gin.Context) *serverutil.Response {
	deps, ok := ctxutil.Get[deputil.Dependencies](c, "deps")
	if !ok {
		return serverutil.InternalServerError(nil)
	}

	limitStr := c.DefaultQuery("limit", strconv.Itoa(defaultRecentLimit))
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = defaultRecentLimit
	}
	if limit > maxRecentLimit {
		limit = maxRecentLimit
	}

	devices, err := deps.StorageService().GetManagedDevices()
	if err != nil {
		return serverutil.InternalServerError(err)
	}

	serials := c.QueryArray("serial")
	var selectedDevices []storageutil.ManagedDevice
	if len(serials) == 0 {
		selectedDevices = devices
	} else {
		serialSet := make(map[string]bool)
		for _, s := range serials {
			serialSet[s] = true
		}
		for _, d := range devices {
			if d.UsbInfo != nil {
				if serialSet[d.UsbInfo.GetSerial()] {
					selectedDevices = append(selectedDevices, d)
				}
			} else {
				if serialSet[DefaultDeviceSerial] {
					selectedDevices = append(selectedDevices, d)
				}
			}
		}
	}

	var allFiles []FileNodeWithTimeJSON
	for _, device := range selectedDevices {
		cirrusDir := device.CirrusDir
		deviceSerial := DefaultDeviceSerial
		if device.UsbInfo != nil {
			deviceSerial = device.UsbInfo.GetSerial()
		}
		// Walk all files recursively
		infos, walkErr := storageutil.StatFilesInDir(cirrusDir, device.Name, device.DataDir, deviceSerial)
		if walkErr != nil {
			continue
		}
		for _, info := range infos {
			if info.IsDir() {
				continue // only return files, not directories
			}
			fullPath := filepath.Join(info.DevicePath, info.Name())
			// Compute the API-relative path by stripping the cirrusDir prefix
			// so the client can use it directly in API calls (same as list_files).
			relDir, relErr := filepath.Rel(cirrusDir, info.DevicePath)
			if relErr != nil {
				relDir = info.DevicePath
			}
			dirPath := filepath.Join(relDir, info.Name())
			allFiles = append(allFiles, FileNodeWithTimeJSON{
				FileNodeJSON: FileNodeJSON{
					Name:         info.Name(),
					Size:         info.FileInfo.Size(),
					IsDir:        false,
					DeviceName:   info.DeviceName,
					DevicePath:   info.DevicePath,
					DirPath:      dirPath,
					FullPath:     fullPath,
					DeviceSerial: deviceSerial,
				},
				ModifiedAt: info.ModTime(),
			})
		}
	}

	// Sort newest first
	sort.Slice(allFiles, func(i, j int) bool {
		return allFiles[i].ModifiedAt.After(allFiles[j].ModifiedAt)
	})

	if limit < len(allFiles) {
		allFiles = allFiles[:limit]
	}

	return serverutil.Ok().WithData(allFiles)
}

var listRecentFilesRoute = serverutil.ApiRoute(
	"GET", "/cirrus/recent", func(c *gin.Context) *serverutil.Response {
		return listRecentFiles(c)
	},
)
