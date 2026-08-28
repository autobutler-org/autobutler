package v0_files

import (
	"sort"

	"github.com/autobutler-org/quark/pkg/util/ctxutil"
	"github.com/autobutler-org/quark/pkg/util/deputil"
	"github.com/autobutler-org/quark/pkg/util/serverutil"
	"github.com/autobutler-org/quark/pkg/util/storageutil"
	"github.com/autobutler-org/quark/pkg/vfs"

	"github.com/gin-gonic/gin"
)

// listFilesByType godoc
// @Summary List all files of a given type
// @Description Recursively walks all managed devices and returns files whose fileType matches the given value, sorted newest-first.
// @Tags files
// @Produce json
// @Param fileType query string true "File type to filter by (e.g. qdoc, qsheet)"
// @Param serial query []string false "Filter by device serial(s)"
// @Success 200 {array} FileNodeWithTimeJSON
// @Failure 400 {object} serverutil.Response "Bad Request"
// @Failure 500 {object} serverutil.Response "Internal Server Error"
// @Router /files/by-type [get]
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

	// VFS path: recursive list + type filter.
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
				if storageutil.DetermineFileTypeFromPath(fi.Path) != targetType {
					continue
				}
				allFiles = append(allFiles, FileNodeWithTimeJSON{
					FileNodeJSON: FileNodeJSON{
						Name:     fi.Name,
						Size:     fi.Size,
						IsDir:    false,
						DirPath:  fi.Path,
						FullPath: fi.Path,
						FileType: fileTypeParam,
					},
					ModifiedAt: fi.ModTime,
				})
			}
			sort.Slice(allFiles, func(i, j int) bool {
				return allFiles[i].ModifiedAt.After(allFiles[j].ModifiedAt)
			})
			return serverutil.Ok().WithData(allFiles)
		}
	}

	for _, device := range selectedDevices {
		filesDir := device.FilesDir
		deviceSerial := DefaultDeviceSerial
		if device.UsbInfo != nil {
			deviceSerial = device.UsbInfo.GetSerial()
		}

		// Walk the whole subtree. This used to call StatFilesInDir, a
		// single-level read, so the endpoint only ever returned files sitting
		// at the storage root despite promising a recursive walk (#1605).
		walkErr := storageutil.WalkFilesInDir(
			c.Request.Context(), filesDir, device.Name, device.DataDir, deviceSerial,
			func(f storageutil.WalkedFile) error {
				info := f.Info
				if info.IsDir() {
					return nil
				}
				if storageutil.DetermineFileTypeFromPath(info.FullPath) != targetType {
					return nil
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
						FileType:     fileTypeParam,
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

	sort.Slice(allFiles, func(i, j int) bool {
		return allFiles[i].ModifiedAt.After(allFiles[j].ModifiedAt)
	})

	return serverutil.Ok().WithData(allFiles)
}

var listFilesByTypeRoute = serverutil.ApiRoute(
	"GET", "/files/by-type", func(c *gin.Context) *serverutil.Response {
		return listFilesByType(c)
	},
)
