package v0_videos

import (
	"database/sql"
	"errors"
	"fmt"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/autobutler-org/quark/internal/db"
	"github.com/autobutler-org/quark/pkg/util/ctxutil"
	"github.com/autobutler-org/quark/pkg/util/deputil"
	"github.com/autobutler-org/quark/pkg/util/serverutil"
	"github.com/autobutler-org/quark/pkg/util/storageutil"
	"github.com/autobutler-org/quark/pkg/util/videoutil"
	"github.com/gin-gonic/gin"
)

// VideoMetadataJSON is the response body for GET /videos/metadata.
type VideoMetadataJSON struct {
	FileName   string         `json:"fileName"`
	FileSize   int64          `json:"fileSize"`
	MTime      int64          `json:"mtime"`
	Duration   float64        `json:"duration"` // seconds
	Width      int            `json:"width"`
	Height     int            `json:"height"`
	VideoCodec string         `json:"videoCodec,omitempty"`
	AudioCodec string         `json:"audioCodec,omitempty"`
	Bitrate    int64          `json:"bitrate"` // bits/s
	Framerate  float64        `json:"framerate"`
	Rotation   int            `json:"rotation"` // degrees
	IsFavorite bool           `json:"isFavorite"`
	Albums     []albumRefJSON `json:"albums"`
}

type albumRefJSON struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

// getVideoMetadata godoc
// @Summary Get metadata for a single video file
// @Description Returns duration, resolution, codec, bitrate, framerate, rotation, and album membership for the specified video.
// @Tags videos
// @Produce json
// @Param serial query string false "Device serial"
// @Param relPath query string true "Relative path to the video file"
// @Success 200 {object} VideoMetadataJSON
// @Failure 400 {object} serverutil.Response "Bad Request"
// @Failure 404 {object} serverutil.Response "Not Found"
// @Failure 501 {object} serverutil.Response "Not Implemented — ffprobe not available"
// @Failure 500 {object} serverutil.Response "Internal Server Error"
// @Router /videos/metadata [get]
func getVideoMetadata(c *gin.Context) *serverutil.Response {
	if !videoutil.Available() {
		return serverutil.NewResponse().
			WithStatusCode(http.StatusNotImplemented).
			WithContentType(serverutil.ContentTypeJSON).
			WithData(gin.H{"error": "ffprobe is not installed on this device"})
	}

	deps, ok := ctxutil.Get[deputil.Dependencies](c, "deps")
	if !ok {
		return serverutil.InternalServerError(nil)
	}

	relPath := c.Query("relPath")
	if relPath == "" {
		return serverutil.BadRequest(fmt.Errorf("relPath is required"))
	}
	serial := c.Query("serial")

	// Resolve the cirrus directory — same pattern as photos.
	filesDir, err := storageutil.GetCirrusDir()
	if err != nil {
		return serverutil.InternalServerError(err)
	}
	if serial != "" {
		if devices, err := deps.StorageService().GetManagedDevices(); err == nil {
			for _, d := range devices {
				if d.UsbInfo != nil && d.UsbInfo.GetSerial() == serial {
					filesDir = d.CirrusDir
					break
				}
			}
		}
	}

	cleanFilesDir := filepath.Clean(filesDir)
	fullPath := filepath.Join(cleanFilesDir, relPath)
	if !strings.HasPrefix(fullPath, cleanFilesDir+string(filepath.Separator)) {
		return serverutil.BadRequest(fmt.Errorf("invalid relPath"))
	}

	stat, err := os.Stat(fullPath)
	if os.IsNotExist(err) {
		return serverutil.NotFound(fmt.Errorf("video not found: %s", relPath))
	}
	if err != nil {
		return serverutil.InternalServerError(err)
	}

	// Probe via ffprobe.
	info, err := videoutil.Probe(c.Request.Context(), fullPath)
	if err != nil {
		return serverutil.InternalServerError(fmt.Errorf("probe video: %w", err))
	}

	ctx := c.Request.Context()

	isFavorite, err := deps.Database().Queries.IsFavorite(
		ctx,
		db.IsFavoriteParams{DeviceSerial: serial, RelPath: relPath},
	)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		_ = c.Error(fmt.Errorf("check favorite for %q: %w", relPath, err))
	}

	albums, err := deps.Database().Queries.ListAlbumsContainingPhoto(
		ctx,
		db.ListAlbumsContainingPhotoParams{
			DeviceSerial: serial,
			RelPath:      relPath,
		},
	)
	if err != nil {
		_ = c.Error(fmt.Errorf("list albums for %q: %w", relPath, err))
		albums = nil
	}
	albumRefs := make([]albumRefJSON, 0, len(albums))
	for _, a := range albums {
		albumRefs = append(albumRefs, albumRefJSON{ID: a.ID, Name: a.Name})
	}

	return serverutil.Ok().WithContentType(serverutil.ContentTypeJSON).WithData(VideoMetadataJSON{
		FileName:   filepath.Base(relPath),
		FileSize:   stat.Size(),
		MTime:      stat.ModTime().Unix(),
		Duration:   roundTo(info.Duration.Seconds(), 3),
		Width:      info.Width,
		Height:     info.Height,
		VideoCodec: info.VideoCodec,
		AudioCodec: info.AudioCodec,
		Bitrate:    info.Bitrate,
		Framerate:  roundTo(info.Framerate, 3),
		Rotation:   info.Rotation,
		IsFavorite: isFavorite,
		Albums:     albumRefs,
	})
}

func roundTo(v float64, decimals int) float64 {
	factor := math.Pow(10, float64(decimals))
	return math.Round(v*factor) / factor
}

var getVideoMetadataRoute = serverutil.ApiRoute(
	"GET", "/videos/metadata", getVideoMetadata,
)
