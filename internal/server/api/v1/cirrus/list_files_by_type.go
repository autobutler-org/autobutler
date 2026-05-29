package v1_files

import (
	"path/filepath"
	"sort"

	"github.com/autobutler-org/autobutler/pkg/util/ctxutil"
	"github.com/autobutler-org/autobutler/pkg/util/deputil"
	"github.com/autobutler-org/autobutler/pkg/util/serverutil"
	"github.com/autobutler-org/autobutler/pkg/util/storageutil"

	"github.com/gin-gonic/gin"
)

// listFilesByType godoc
// @Summary List all files of a given type
// @Description Recursively walks all managed devices and returns files whose fileType matches the given value, sorted newest-first.
// @Tags cirrus
// @Produce json
// @Param fileType query string true "File type to filter by (e.g. abdoc, absheet)"
// @Param serial query []string false "Filter by device serial(s)"
// @Success 200 {array} FileNodeWithTimeJSON
// @Failure 400 {object} serverutil.Response "Bad Request"
// @Failure 500 {object} serverutil.Response "Internal Server Error"
// @Router /cirrus/by-type [get]
func listFilesByType(c *gin.Context) *serverutil.Response {
	deps, ok := ctxutil.Get[deputil.Dependencies](c, "deps")
	if !ok {
		return serverutil.InternalServerError(nil)
	}

	fileTypeParam := c.Query("fileType")
	if fileTypeParam == "" {
		return serverutil.BadRequest(nil)
	}
	targetType := storageutil.FileType(fileTypeParam)

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

	// Use make() instead of var to ensure JSON serialization produces []
	// instead of null when there are no files (nil slice encodes as null).
	allFiles := make([]FileNodeWithTimeJSON, 0)
	for _, device := range selectedDevices {
		cirrusDir := device.CirrusDir
		deviceSerial := DefaultDeviceSerial
		if device.UsbInfo != nil {
			deviceSerial = device.UsbInfo.GetSerial()
		}

		infos, walkErr := storageutil.StatFilesInDir(cirrusDir, device.Name, device.DataDir, deviceSerial)
		if walkErr != nil {
			continue
		}
		for _, info := range infos {
			if info.IsDir() {
				continue
			}
			if storageutil.DetermineFileTypeFromPath(info.FullPath) != targetType {
				continue
			}
			relPath, relErr := filepath.Rel(cirrusDir, info.FullPath)
			if relErr != nil {
				relPath = info.Name()
			}
			allFiles = append(allFiles, FileNodeWithTimeJSON{
				FileNodeJSON: FileNodeJSON{
					Name:         info.Name(),
					Size:         info.FileInfo.Size(),
					IsDir:        false,
					DeviceName:   info.DeviceName,
					DevicePath:   info.DevicePath,
					DirPath:      relPath,
					FullPath:     info.FullPath,
					DeviceSerial: deviceSerial,
					FileType:     fileTypeParam,
				},
				ModifiedAt: info.ModTime(),
			})
		}
	}

	sort.Slice(allFiles, func(i, j int) bool {
		return allFiles[i].ModifiedAt.After(allFiles[j].ModifiedAt)
	})

	return serverutil.Ok().WithData(allFiles)
}

var listFilesByTypeRoute = serverutil.ApiRoute(
	"GET", "/cirrus/by-type", func(c *gin.Context) *serverutil.Response {
		return listFilesByType(c)
	},
)
