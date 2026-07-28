package v0_photos

import (
	"sort"
	"strconv"

	"github.com/autobutler-org/autobutler/pkg/util/ctxutil"
	"github.com/autobutler-org/autobutler/pkg/util/deputil"
	"github.com/autobutler-org/autobutler/pkg/util/photoutil"
	"github.com/autobutler-org/autobutler/pkg/util/serverutil"
	"github.com/autobutler-org/autobutler/pkg/util/storageutil"
	"github.com/autobutler-org/autobutler/pkg/vfs"

	"github.com/gin-gonic/gin"
)

const defaultLimit = 50
const maxLimit = 200

// PhotoJSON is a JSON-serializable representation of a photo file.
type PhotoJSON struct {
	RelPath      string `json:"relPath"`
	FileName     string `json:"fileName"`
	Size         int64  `json:"size"`
	MTime        int64  `json:"mtime"`
	Serial       string `json:"serial"`
	HasLiveVideo bool   `json:"hasLiveVideo,omitempty"`
}

// PaginatedPhotosResponse wraps a page of photos with pagination metadata.
type PaginatedPhotosResponse struct {
	Photos []PhotoJSON `json:"photos"`
	Total  int         `json:"total"`
	Offset int         `json:"offset"`
	Limit  int         `json:"limit"`
}

// listPhotos godoc
// @Summary List photos
// @Description Finds all photos across all managed devices with pagination support.
// @Tags photos
// @Produce json
// @Param offset query int false "Pagination offset (default 0)"
// @Param limit query int false "Page size (default 50, max 200)"
// @Param serial query string false "Device serial to filter by"
// @Success 200 {object} PaginatedPhotosResponse
// @Failure 500 {object} serverutil.Response "Internal Server Error"
// @Router /photos [get]
func listPhotos(c *gin.Context) *serverutil.Response {
	// Parse pagination params
	offset := 0
	if v := c.Query("offset"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed >= 0 {
			offset = parsed
		}
	}
	limit := defaultLimit
	if v := c.Query("limit"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 {
			limit = parsed
		}
	}
	if limit > maxLimit {
		limit = maxLimit
	}

	serial := c.Query("serial")

	// Collect photos from all managed devices (or just the requested serial).
	// TODO: For very large collections, an index/cache would be needed instead
	// of walking the entire directory tree on every request. The walk itself is
	// fast — it's downstream operations like thumbnail generation that are slow.
	deps, ok := ctxutil.Get[deputil.Dependencies](c, "deps")
	if !ok {
		return serverutil.InternalServerError(nil)
	}

	var allPhotos []PhotoJSON

	// VFS path: use recursive image listing when registry is available.
	if reg := deps.VFSRegistry(); reg != nil {
		if fsys, ok := reg.Get("files"); ok {
			serialFilter := []string{}
			if serial != "" {
				serialFilter = []string{serial}
			}
			infos, listErr := fsys.List(c.Request.Context(), "", &vfs.ListFilter{
				Recursive:    true,
				MimePrefix:   "image/",
				SerialFilter: serialFilter,
			})
			if listErr != nil {
				return serverutil.InternalServerError(listErr)
			}
			for _, fi := range infos {
				if fi.IsDir {
					continue
				}
				allPhotos = append(allPhotos, PhotoJSON{
					RelPath:  fi.Path,
					FileName: fi.Name,
					Size:     fi.Size,
					MTime:    fi.ModTime.Unix(),
				})
			}
			goto paginate
		}
	}

	// Fallback: walk devices via photoutil.
	{
		devices, err := deps.StorageService().GetManagedDevices()
		if err != nil {
			return serverutil.InternalServerError(err)
		}
		if serial != "" {
			filtered := make([]storageutil.ManagedDevice, 0, 1)
			for _, d := range devices {
				deviceSerial := ""
				if d.UsbInfo != nil {
					deviceSerial = d.UsbInfo.GetSerial()
				}
				if deviceSerial == serial {
					filtered = append(filtered, d)
				}
			}
			devices = filtered
		}
		for _, device := range devices {
			deviceSerial := ""
			if device.UsbInfo != nil {
				deviceSerial = device.UsbInfo.GetSerial()
			}
			photos, err := photoutil.FindAllPhotosRecursively(device.CirrusDir)
			if err != nil {
				continue
			}
			for _, photo := range photos {
				info := photo.FileInfo
				allPhotos = append(allPhotos, PhotoJSON{
					RelPath:      photo.RelPath,
					FileName:     info.Name(),
					Size:         info.Size(),
					MTime:        info.ModTime().Unix(),
					Serial:       deviceSerial,
					HasLiveVideo: photo.HasLiveVideo,
				})
			}
		}
	}

paginate:

	// Sort by modification time descending (newest first)
	sort.Slice(allPhotos, func(i, j int) bool {
		return allPhotos[i].MTime > allPhotos[j].MTime
	})

	total := len(allPhotos)

	// Apply pagination
	if offset >= total {
		return serverutil.Ok().WithContentType(serverutil.ContentTypeJSON).WithData(PaginatedPhotosResponse{
			Photos: []PhotoJSON{},
			Total:  total,
			Offset: offset,
			Limit:  limit,
		})
	}

	end := offset + limit
	if end > total {
		end = total
	}

	return serverutil.Ok().WithContentType(serverutil.ContentTypeJSON).WithData(PaginatedPhotosResponse{
		Photos: allPhotos[offset:end],
		Total:  total,
		Offset: offset,
		Limit:  limit,
	})
}

var listPhotosRoute = serverutil.ApiRoute(
	"GET", "/photos", listPhotos,
)
