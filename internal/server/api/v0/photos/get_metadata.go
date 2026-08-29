package v0_photos

import (
	"bytes"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/autobutler-org/quark/internal/db"
	"github.com/autobutler-org/quark/pkg/util/ctxutil"
	"github.com/autobutler-org/quark/pkg/util/deputil"
	"github.com/autobutler-org/quark/pkg/util/photoutil"
	"github.com/autobutler-org/quark/pkg/util/serverutil"
	"github.com/autobutler-org/quark/pkg/util/storageutil"
	"github.com/autobutler-org/quark/pkg/vfs"
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

// getMetadata godoc
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
func getMetadata(c *gin.Context) *serverutil.Response {
	deps, ok := ctxutil.Get[deputil.Dependencies](c, "deps")
	if !ok {
		return serverutil.InternalServerError(nil)
	}

	relPath := c.Query("relPath")
	if relPath == "" {
		return serverutil.BadRequest(fmt.Errorf("relPath is required"))
	}
	serial := c.Query("serial")

	// Stat and EXIF: use VFS when available, fall back to direct disk access.
	var statSize int64
	var statMTime int64
	var exifData *ExifJSON
	width, height := 0, 0

	if reg := deps.VFSRegistry(); reg != nil {
		if fsys, ok := reg.Get("files"); ok {
			fi, statErr := fsys.Stat(c.Request.Context(), relPath)
			if statErr != nil {
				if errors.Is(statErr, vfs.ErrNotFound) {
					return serverutil.NotFound(fmt.Errorf("photo not found: %s", relPath))
				}
				return serverutil.InternalServerError(statErr)
			}
			statSize = fi.Size
			statMTime = fi.ModTime.Unix()

			imgFormat := photoutil.ImageFormatFromPath(relPath)
			if imgFormat != 0 {
				if rc, openErr := fsys.Open(c.Request.Context(), relPath); openErr == nil {
					var rs io.ReadSeeker
					if seekable, ok := rc.(io.ReadSeeker); ok {
						rs = seekable
					} else {
						raw, _ := io.ReadAll(rc)
						rs = bytes.NewReader(raw)
					}
					if data, exifErr := photoutil.DecodeExif(rs, imgFormat); exifErr == nil && data != nil {
						exifData = exifDataToJSON(data)
						width = data.Width
						height = data.Height
					}
					rc.Close()
				}
			}
			goto rotations
		}
	}

	// Fallback: direct disk access.
	{
		filesDir, err := storageutil.GetFilesDir()
		if err != nil {
			return serverutil.InternalServerError(err)
		}
		if deviceDir, ok := deps.StorageService().FindDeviceFilesDirBySerial(serial); ok {
			filesDir = deviceDir
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
		statSize = stat.Size()
		statMTime = stat.ModTime().Unix()

		imgFormat := photoutil.ImageFormatFromPath(fullPath)
		if imgFormat != 0 {
			if f, err := os.Open(fullPath); err == nil {
				if data, exifErr := photoutil.DecodeExif(f, imgFormat); exifErr == nil && data != nil {
					exifData = exifDataToJSON(data)
					width = data.Width
					height = data.Height
				}
				f.Close()
			}
		}
	}

rotations:

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

	// Live-video companion: resolve via disk only (VFS doesn't expose sidecar detection).
	liveVideoPath := ""
	if filesDir, err := storageutil.GetFilesDir(); err == nil {
		searchDir := filesDir
		if deviceDir, ok := deps.StorageService().FindDeviceFilesDirBySerial(serial); ok {
			searchDir = deviceDir
		}
		fullPath := filepath.Join(filepath.Clean(searchDir), relPath)
		liveVideoPath = findLivePhotoVideo(fullPath, relPath)
	}

	return serverutil.Ok().WithContentType(serverutil.ContentTypeJSON).WithData(PhotoMetadataJSON{
		FileName:           filepath.Base(relPath),
		FileSize:           statSize,
		MTime:              statMTime,
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
//
// The sibling is found by reading the directory rather than by stat-ing
// candidate spellings. A stat loop reports the spelling it guessed, not the
// one on disk: on a case-insensitive filesystem (macOS, Windows) stat of
// "photo.MOV" succeeds for a file actually named "photo.mov", so the client
// was handed a path that does not exist as spelled — and would 404 against a
// case-sensitive filesystem holding the same library.
func findLivePhotoVideo(fullPath, relPath string) string {
	ext := strings.ToLower(filepath.Ext(fullPath))
	if ext != ".heic" && ext != ".heif" && ext != ".jpg" && ext != ".jpeg" {
		return ""
	}

	entries, err := os.ReadDir(filepath.Dir(fullPath))
	if err != nil {
		return ""
	}

	imageName := filepath.Base(fullPath)
	stem := strings.TrimSuffix(imageName, filepath.Ext(imageName))

	// .mov before .mp4: a Live Photo's companion is a .mov, and preferring it
	// keeps the result stable for a library holding both. Within an extension
	// os.ReadDir is already sorted, so the pick is deterministic either way.
	for _, want := range []string{".mov", ".mp4"} {
		for _, e := range entries {
			name := e.Name()
			if e.IsDir() || !strings.EqualFold(filepath.Ext(name), want) {
				continue
			}
			if !strings.EqualFold(strings.TrimSuffix(name, filepath.Ext(name)), stem) {
				continue
			}
			// Keep relPath's directory and swap in the real filename, so the
			// returned path is spelled exactly as it is on disk.
			if dir := path.Dir(relPath); dir != "." && dir != "/" {
				return path.Join(dir, name)
			}
			return name
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

var getMetadataRoute = serverutil.ApiRoute(
	"GET", "/photos/metadata", getMetadata,
)
