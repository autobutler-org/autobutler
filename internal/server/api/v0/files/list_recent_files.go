package v0_files

import (
	"sort"
	"strconv"
	"time"

	"github.com/autobutler-org/quark/pkg/util/ctxutil"
	"github.com/autobutler-org/quark/pkg/util/deputil"
	"github.com/autobutler-org/quark/pkg/util/serverutil"
	"github.com/autobutler-org/quark/pkg/util/storageutil"
	"github.com/autobutler-org/quark/pkg/vfs"

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
// @Tags files
// @Produce json
// @Param limit query int false "Maximum number of files to return (default 20, max 200)"
// @Param serial query []string false "Filter by device serial(s)"
// @Success 200 {array} FileNodeWithTimeJSON
// @Failure 500 {object} serverutil.Response "Internal Server Error"
// @Router /files/recent [get]
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

	serials := c.QueryArray("serial")

	var allFiles []FileNodeWithTimeJSON

	// VFS path: use recursive list when registry is available.
	if reg := deps.VFSRegistry(); reg != nil {
		if fsys, ok := reg.Get("files"); ok {
			infos, err := fsys.List(c.Request.Context(), "", &vfs.ListFilter{Recursive: true, SerialFilter: serials})
			if err != nil {
				return serverutil.InternalServerError(err)
			}
			for _, fi := range infos {
				if fi.IsDir {
					continue
				}
				allFiles = append(allFiles, FileNodeWithTimeJSON{
					FileNodeJSON: FileNodeJSON{
						Name:     fi.Name,
						Size:     fi.Size,
						IsDir:    false,
						DirPath:  fi.Path,
						FullPath: fi.Path,
						FileType: string(storageutil.DetermineFileTypeFromPath(fi.Path)),
					},
					ModifiedAt: fi.ModTime,
				})
			}
			sort.Slice(allFiles, func(i, j int) bool {
				return allFiles[i].ModifiedAt.After(allFiles[j].ModifiedAt)
			})
			if limit < len(allFiles) {
				allFiles = allFiles[:limit]
			}
			return serverutil.Ok().WithData(allFiles)
		}
	}

	// Fallback: walk devices via StorageService.
	devices, err := deps.StorageService().GetManagedDevices()
	if err != nil {
		return serverutil.InternalServerError(err)
	}

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

	for _, device := range selectedDevices {
		filesDir := device.FilesDir
		deviceSerial := DefaultDeviceSerial
		if device.UsbInfo != nil {
			deviceSerial = device.UsbInfo.GetSerial()
		}
		// Walk all files recursively. This used to call StatFilesInDir, a
		// single-level read, so "recent files" could never surface anything
		// outside the storage root (#1605). WalkedFile.RelPath is the
		// API-relative path the client uses directly — the same shape
		// list_files produces.
		walkErr := storageutil.WalkFilesInDir(
			c.Request.Context(), filesDir, device.Name, device.DataDir, deviceSerial,
			func(f storageutil.WalkedFile) error {
				info := f.Info
				if info.IsDir() {
					return nil // only return files, not directories
				}
				allFiles = append(allFiles, FileNodeWithTimeJSON{
					FileNodeJSON: FileNodeJSON{
						Name:         info.Name(),
						Size:         info.FileInfo.Size(),
						IsDir:        false,
						DeviceName:   info.DeviceName,
						DevicePath:   info.DevicePath,
						DirPath:      f.RelPath,
						FullPath:     info.FullPath,
						DeviceSerial: deviceSerial,
					},
					ModifiedAt: info.ModTime(),
				})
				return nil
			},
		)
		if walkErr != nil {
			continue
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
	"GET", "/files/recent", func(c *gin.Context) *serverutil.Response {
		return listRecentFiles(c)
	},
)
