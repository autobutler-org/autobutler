package v0_photos

import (
	"database/sql"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/autobutler-org/autobutler/internal/db"
	"github.com/autobutler-org/autobutler/pkg/util/ctxutil"
	"github.com/autobutler-org/autobutler/pkg/util/deputil"
	"github.com/autobutler-org/autobutler/pkg/util/photoutil"
	"github.com/autobutler-org/autobutler/pkg/util/serverutil"
	"github.com/autobutler-org/autobutler/pkg/util/storageutil"
	"github.com/gin-gonic/gin"
)

// ExifJSON holds the EXIF fields we expose. All fields are pointers so that
// absent values are omitted from the JSON output.
type ExifJSON struct {
	DateTaken    *string  `json:"dateTaken,omitempty"`
	Make         *string  `json:"make,omitempty"`
	Model        *string  `json:"model,omitempty"`
	Lens         *string  `json:"lens,omitempty"`
	Aperture     *float64 `json:"aperture,omitempty"`
	ShutterSpeed *string  `json:"shutterSpeed,omitempty"`
	ISO          *int     `json:"iso,omitempty"`
	FocalLength  *float64 `json:"focalLength,omitempty"`
	Latitude     *float64 `json:"latitude,omitempty"`
	Longitude    *float64 `json:"longitude,omitempty"`
}

// AlbumRefJSON is a minimal album reference for embedding in metadata.
type AlbumRefJSON struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

// PhotoMetadataJSON is the full metadata response for a single photo.
type PhotoMetadataJSON struct {
	FileName           string         `json:"fileName"`
	FileSize           int64          `json:"fileSize"`
	MTime              int64          `json:"mtime"`
	Width              int            `json:"width"`
	Height             int            `json:"height"`
	RotationQuarters   int64          `json:"rotationQuarters"`
	IsFavorite         bool           `json:"isFavorite"`
	Exif               *ExifJSON      `json:"exif,omitempty"`
	Albums             []AlbumRefJSON `json:"albums"`
	LivePhotoVideoPath string         `json:"livePhotoVideoPath,omitempty"`
}

// getPhotoMetadata godoc
// @Summary Get metadata for a single photo
// @Description Returns EXIF, file info, and album membership for the specified photo.
// @Tags photos
// @Produce json
// @Param serial query string false "Device serial"
// @Param relPath query string true "Relative path to the photo file"
// @Success 200 {object} PhotoMetadataJSON
// @Failure 400 {object} serverutil.Response "Bad Request"
// @Failure 404 {object} serverutil.Response "Not Found"
// @Failure 500 {object} serverutil.Response "Internal Server Error"
// @Router /photos/metadata [get]
func getPhotoMetadata(c *gin.Context) *serverutil.Response {
	deps, ok := ctxutil.Get[deputil.Dependencies](c, "deps")
	if !ok {
		return serverutil.InternalServerError(nil)
	}

	relPath := c.Query("relPath")
	if relPath == "" {
		return serverutil.BadRequest(fmt.Errorf("relPath is required"))
	}
	serial := c.Query("serial")

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
		return serverutil.NotFound(fmt.Errorf("photo not found: %s", relPath))
	}
	if err != nil {
		return serverutil.InternalServerError(err)
	}

	// --- EXIF + dimensions (works for JPEG, HEIC, PNG, WebP, TIFF, RAW) ---
	var exifData *ExifJSON
	width, height := 0, 0

	imgFormat := photoutil.ImageFormatFromPath(fullPath)
	if imgFormat != 0 {
		if f, err := os.Open(fullPath); err == nil {
			if data, err := photoutil.DecodeExif(f, imgFormat); err == nil && data != nil {
				exifData = exifDataToJSON(data)
				width = data.Width
				height = data.Height
			}
			f.Close()
		}
	}

	ctx := c.Request.Context()

	var rotationQuarters int64
	if rq, err := deps.Database().Queries.GetPhotoRotation(
		ctx,
		db.GetPhotoRotationParams{DeviceSerial: serial, RelPath: relPath},
	); err == nil {
		rotationQuarters = rq
	} else if !errors.Is(err, sql.ErrNoRows) {
		return serverutil.InternalServerError(fmt.Errorf("get photo rotation: %w", err))
	}

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
		_ = c.Error(fmt.Errorf("list albums containing photo %q for device %q: %w", relPath, serial, err))
		albums = nil
	}
	albumRefs := make([]AlbumRefJSON, 0, len(albums))
	for _, a := range albums {
		albumRefs = append(albumRefs, AlbumRefJSON{ID: a.ID, Name: a.Name})
	}

	liveVideoPath := findLivePhotoVideo(fullPath, relPath)

	return serverutil.Ok().WithContentType(serverutil.ContentTypeJSON).WithData(PhotoMetadataJSON{
		FileName:           filepath.Base(relPath),
		FileSize:           stat.Size(),
		MTime:              stat.ModTime().Unix(),
		Width:              width,
		Height:             height,
		RotationQuarters:   rotationQuarters,
		IsFavorite:         isFavorite,
		Exif:               exifData,
		Albums:             albumRefs,
		LivePhotoVideoPath: liveVideoPath,
	})
}

// findLivePhotoVideo checks if a companion .MOV file exists for an image,
// which indicates an iPhone Live Photo. Returns the relative path to the
// video, or "" if none found.
func findLivePhotoVideo(fullPath, relPath string) string {
	ext := strings.ToLower(filepath.Ext(fullPath))
	if ext != ".heic" && ext != ".heif" && ext != ".jpg" && ext != ".jpeg" {
		return ""
	}

	base := strings.TrimSuffix(fullPath, filepath.Ext(fullPath))
	for _, vidExt := range []string{".MOV", ".mov", ".Mp4", ".mp4"} {
		candidate := base + vidExt
		if _, err := os.Stat(candidate); err == nil {
			relBase := strings.TrimSuffix(relPath, filepath.Ext(relPath))
			return relBase + vidExt
		}
	}
	return ""
}

func exifDataToJSON(data *photoutil.ExifData) *ExifJSON {
	e := &ExifJSON{}
	empty := true

	if data.DateTaken != nil {
		s := data.DateTaken.Format(time.RFC3339)
		e.DateTaken = &s
		empty = false
	}
	if data.Make != "" {
		e.Make = &data.Make
		empty = false
	}
	if data.Model != "" {
		e.Model = &data.Model
		empty = false
	}
	if data.LensModel != "" {
		e.Lens = &data.LensModel
		empty = false
	}
	if data.Aperture != 0 {
		v := roundTo(data.Aperture, 2)
		e.Aperture = &v
		empty = false
	}
	if data.ShutterSpeed[1] != 0 {
		n, d := data.ShutterSpeed[0], data.ShutterSpeed[1]
		var s string
		if n == 0 {
			s = "0"
		} else if d%n == 0 {
			s = fmt.Sprintf("1/%d", d/n)
		} else {
			s = fmt.Sprintf("%d/%d", n, d)
		}
		e.ShutterSpeed = &s
		empty = false
	}
	if data.ISO != 0 {
		e.ISO = &data.ISO
		empty = false
	}
	if data.FocalLength != 0 {
		v := roundTo(data.FocalLength, 2)
		e.FocalLength = &v
		empty = false
	}
	if data.HasGPS {
		lat := roundTo(data.Latitude, 6)
		lon := roundTo(data.Longitude, 6)
		e.Latitude = &lat
		e.Longitude = &lon
		empty = false
	}

	if empty {
		return nil
	}
	return e
}

func roundTo(v float64, decimals int) float64 {
	factor := math.Pow(10, float64(decimals))
	return math.Round(v*factor) / factor
}

var getPhotoMetadataRoute = serverutil.ApiRoute(
	"GET", "/photos/metadata", getPhotoMetadata,
)
