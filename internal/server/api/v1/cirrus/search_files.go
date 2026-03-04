package v1_files

import (
	"strings"

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
// @Param search query string false "Search term to find"
// @Param serial query string false "Device serial number to filter by"
// @Router /cirrus/search [get]
func searchFiles(c *gin.Context) *serverutil.Response {
	search := strings.TrimSpace(c.Query("search"))
	serials := c.QueryArray("serial")

	devices, err := storageutil.GetManagedDevices()
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

			if search == "" || strings.Contains(strings.ToLower(entry.Name), strings.ToLower(search)) {
				allFiles = append(allFiles, entry)
			}
		}
	}

	jsonData := allFiles
	return serverutil.Ok().WithData(jsonData)
}

var searchFilesRoute = serverutil.ApiRoute(
	"GET", "/cirrus/search", func(c *gin.Context) *serverutil.Response {
		return searchFiles(c)
	},
)
