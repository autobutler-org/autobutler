package v1_files

import (
	"strings"

	"github.com/autobutler-org/autobutler/pkg/util/ctxutil"
	"github.com/autobutler-org/autobutler/pkg/util/deputil"
	"github.com/autobutler-org/autobutler/pkg/util/serverutil"
	"github.com/autobutler-org/autobutler/pkg/util/storageutil"

	"github.com/gin-gonic/gin"
)

// searchFiles godoc
// @Summary Searches for files
// @Schemes http https
// @Description searches for a file across all managed devices for the given search term. If deviceSerial is empty, search across all devices. Otherwise, only for the specified device
// @Tags cirrus
// @Produce json
// @Success 200 {array} FileNodeJSON
// @Failure 500 {object} serverutil.Response "Internal Server Error"
// @Param query query string false "Search term to find"
// @Param serial query string false "Device serial number to filter by"
// @Router /cirrus/search [get]
func searchFiles(c *gin.Context) *serverutil.Response {
	deps, ok := ctxutil.Get[deputil.Dependencies](c, "deps")
	if !ok {
		return serverutil.InternalServerError(nil)
	}
	query := strings.TrimSpace(c.Query("query"))
	serials := c.QueryArray("serial")

	idx := deps.FileIndex()
	if idx == nil {
		return searchFilesDiskWalk(c, deps, query, serials)
	}

	serialSet := make(map[string]bool, len(serials))
	for _, s := range serials {
		serialSet[s] = true
	}

	matches := idx.Search(query, serialSet)
	allFiles := make([]FileNodeJSON, 0, len(matches))
	for _, f := range matches {
		// DirPath must be the full relative path (e.g. "docs/notes.txt"), not
		// just the parent dir. The Flutter CirrusFileNode.apiPath getter uses
		// DirPath as the full API path, consistent with how list_files.go
		// populates it (filepath.Join(rootDir, file.Name())).
		allFiles = append(allFiles, FileNodeJSON{
			Name:         f.Name,
			DirPath:      f.RelPath,
			IsDir:        false,
			DeviceSerial: f.DeviceSerial,
		})
	}
	return serverutil.Ok().WithData(allFiles)
}

// searchFilesDiskWalk is the original BFS implementation used as fallback when the
// in-memory index is unavailable.
func searchFilesDiskWalk(c *gin.Context, deps deputil.Dependencies, query string, serials []string) *serverutil.Response {
	devices, err := deps.StorageService().GetManagedDevices()
	if err != nil {
		return serverutil.InternalServerError(err)
	}
	var selectedDevices []storageutil.ManagedDevice
	if len(serials) == 0 {
		selectedDevices = devices
	} else {
		// Build a set for quick lookup
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
				// Internal device (no UsbInfo) represented by empty serial string
				if serialSet[DefaultDeviceSerial] {
					selectedDevices = append(selectedDevices, d)
				}
			}
		}
		if len(selectedDevices) == 0 {
			return serverutil.Ok().WithContentType(serverutil.ContentTypeJSON).WithData([]FileNodeJSON{})
		}
	}
	allFiles := make([]FileNodeJSON, 0)
	dirsToScan := []string{""}
	seenDirs := map[string]bool{"": true}

	for len(dirsToScan) > 0 {
		currentDir := dirsToScan[0]
		dirsToScan = dirsToScan[1:]

		entries, err := listFilesImpl(currentDir, selectedDevices)
		if err != nil {
			continue
		}

		for _, entry := range entries {
			if entry.IsDir {
				if !seenDirs[entry.DirPath] {
					seenDirs[entry.DirPath] = true
					dirsToScan = append(dirsToScan, entry.DirPath)
				}
				continue
			}

			if query == "" || strings.Contains(strings.ToLower(entry.Name), strings.ToLower(query)) {
				allFiles = append(allFiles, entry)
			}
		}
	}

	return serverutil.Ok().WithData(allFiles)
}

var searchFilesRoute = serverutil.ApiRoute(
	"GET", "/cirrus/search", func(c *gin.Context) *serverutil.Response {
		return searchFiles(c)
	},
)
