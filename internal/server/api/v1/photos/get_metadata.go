package v1_photos

import (
	"context"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"math"
	"os"
	"path/filepath"
	"time"

	"github.com/autobutler-org/autobutler/internal/db"
	"github.com/autobutler-org/autobutler/pkg/util/ctxutil"
	"github.com/autobutler-org/autobutler/pkg/util/deputil"
	"github.com/autobutler-org/autobutler/pkg/util/serverutil"
	"github.com/autobutler-org/autobutler/pkg/util/storageutil"
	"github.com/gin-gonic/gin"
	"github.com/rwcarlsen/goexif/exif"
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
	FileName string         `json:"fileName"`
	FileSize int64          `json:"fileSize"`
	MTime    int64          `json:"mtime"`
	Width    int            `json:"width"`
	Height   int            `json:"height"`
	Exif     *ExifJSON      `json:"exif,omitempty"`
	Albums   []AlbumRefJSON `json:"albums"`
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

	// Resolve the base directory, same logic as the thumbnails endpoint.
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

	fullPath := filepath.Join(filesDir, relPath)
	stat, err := os.Stat(fullPath)
	if os.IsNotExist(err) {
		return serverutil.NotFound(fmt.Errorf("photo not found: %s", relPath))
	}
	if err != nil {
		return serverutil.InternalServerError(err)
	}

	// --- Dimensions ---
	width, height := 0, 0
	if f, err := os.Open(fullPath); err == nil {
		if cfg, _, err := image.DecodeConfig(f); err == nil {
			width = cfg.Width
			height = cfg.Height
		}
		f.Close()
	}

	// --- EXIF ---
	var exifData *ExifJSON
	if f, err := os.Open(fullPath); err == nil {
		if x, err := exif.Decode(f); err == nil {
			exifData = extractExif(x)
		}
		f.Close()
	}

	// --- Album membership ---
	albums, err := deps.Database().Queries.ListAlbumsContainingPhoto(
		context.Background(),
		db.ListAlbumsContainingPhotoParams{
			DeviceSerial: serial,
			RelPath:      relPath,
		},
	)
	if err != nil {
		albums = nil // non-fatal; return empty list
	}
	albumRefs := make([]AlbumRefJSON, 0, len(albums))
	for _, a := range albums {
		albumRefs = append(albumRefs, AlbumRefJSON{ID: a.ID, Name: a.Name})
	}

	return serverutil.Ok().WithContentType(serverutil.ContentTypeJSON).WithData(PhotoMetadataJSON{
		FileName: filepath.Base(relPath),
		FileSize: stat.Size(),
		MTime:    stat.ModTime().Unix(),
		Width:    width,
		Height:   height,
		Exif:     exifData,
		Albums:   albumRefs,
	})
}

// extractExif pulls the fields we care about out of a decoded EXIF block.
func extractExif(x *exif.Exif) *ExifJSON {
	e := &ExifJSON{}
	empty := true

	if t, err := x.DateTime(); err == nil {
		s := t.UTC().Format(time.RFC3339)
		e.DateTaken = &s
		empty = false
	}
	if s, err := stringTag(x, exif.Make); err == nil {
		e.Make = &s
		empty = false
	}
	if s, err := stringTag(x, exif.Model); err == nil {
		e.Model = &s
		empty = false
	}
	if s, err := stringTag(x, exif.LensModel); err == nil {
		e.Lens = &s
		empty = false
	}
	if rat, err := x.Get(exif.FNumber); err == nil {
		if n, d, err := rat.Rat2(0); err == nil && d != 0 {
			v := roundTo(float64(n)/float64(d), 2)
			e.Aperture = &v
			empty = false
		}
	}
	if rat, err := x.Get(exif.ExposureTime); err == nil {
		if n, d, err := rat.Rat2(0); err == nil {
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
	}
	if tag, err := x.Get(exif.ISOSpeedRatings); err == nil {
		if v, err := tag.Int(0); err == nil {
			e.ISO = &v
			empty = false
		}
	}
	if rat, err := x.Get(exif.FocalLength); err == nil {
		if n, d, err := rat.Rat2(0); err == nil && d != 0 {
			v := roundTo(float64(n)/float64(d), 2)
			e.FocalLength = &v
			empty = false
		}
	}
	if lat, lon, err := x.LatLong(); err == nil {
		rlat := roundTo(lat, 6)
		rlon := roundTo(lon, 6)
		e.Latitude = &rlat
		e.Longitude = &rlon
		empty = false
	}

	if empty {
		return nil
	}
	return e
}

func stringTag(x *exif.Exif, field exif.FieldName) (string, error) {
	tag, err := x.Get(field)
	if err != nil {
		return "", err
	}
	s, err := tag.StringVal()
	if err != nil {
		return "", err
	}
	return s, nil
}

func roundTo(v float64, decimals int) float64 {
	factor := math.Pow(10, float64(decimals))
	return math.Round(v*factor) / factor
}

var getPhotoMetadataRoute = serverutil.ApiRoute(
	"GET", "/photos/metadata", getPhotoMetadata,
)
