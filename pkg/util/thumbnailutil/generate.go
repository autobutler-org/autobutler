package thumbnailutil

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"time"

	"github.com/autobutler-org/quark/internal/db"
	"github.com/autobutler-org/quark/pkg/util/photoutil"
	"github.com/autobutler-org/quark/pkg/util/videoutil"
)

// Generate renders the thumbnail for a file on disk and commits it to the
// cache. Video sources get a representative frame extracted with ffmpeg first,
// which then goes through the same image pipeline as a photo.
func Generate(params GenerateParams) (GenerateResult, error) {
	thumbSrcPath := params.SourcePath
	if params.IsVideo {
		framePath, cleanup, err := extractVideoFrame(params.Ctx, params.SourcePath)
		if err != nil {
			return GenerateResult{}, err
		}
		defer cleanup()
		thumbSrcPath = framePath
	}

	result, err := photoutil.GenerateThumbnail(photoutil.GenerateThumbnailParams{
		FilePath: thumbSrcPath,
		Width:    params.Width,
		Height:   params.Height,
	})
	if err != nil {
		return GenerateResult{}, err
	}

	// Apply server-side rotation so the cached thumbnail matches the
	// orientation the user has set.
	if params.RotationQuarters != 0 {
		result.Thumbnail = photoutil.ApplyRotation(result.Thumbnail, params.RotationQuarters)
	}

	// Compute and store perceptual dHash async — the image is already
	// decoded here so hashing is nearly free. Non-blocking; a failure to
	// store the hash does not affect thumbnail delivery.
	if !params.IsVideo {
		thumb := result.Thumbnail
		go func() {
			hashHex := photoutil.DHashHex(thumb)
			_ = params.Queries.UpsertPhotoHash(
				context.Background(),
				db.UpsertPhotoHashParams{
					DeviceSerial: params.Serial,
					RelPath:      params.RelPath,
					Dhash:        sql.NullString{String: hashHex, Valid: true},
				},
			)
		}()
	}

	modTime, err := writeCache(params.CachedPath, result.Thumbnail, !params.IsVideo && params.Ext == ".png")
	if err != nil {
		return GenerateResult{}, err
	}
	return GenerateResult{CachedModTime: modTime}, nil
}

// GenerateFromReader renders the thumbnail for an already-open source stream
// and commits it to the cache. A source it cannot decode comes back as
// [ErrUnsupportedSource] so a caller with another way to reach the file can
// fall through to it.
func GenerateFromReader(params GenerateFromReaderParams) (GenerateResult, error) {
	result, err := photoutil.GenerateThumbnailFromReader(params.Reader, params.Ext, params.Width, params.Height)
	if err != nil {
		return GenerateResult{}, fmt.Errorf("%w: %w", ErrUnsupportedSource, err)
	}

	if params.RotationQuarters != 0 {
		result.Thumbnail = photoutil.ApplyRotation(result.Thumbnail, params.RotationQuarters)
	}

	modTime, err := writeCache(params.CachedPath, result.Thumbnail, params.Ext == ".png")
	if err != nil {
		return GenerateResult{}, err
	}
	return GenerateResult{CachedModTime: modTime}, nil
}

// extractVideoFrame writes a representative frame of a video to a temporary
// JPEG and returns its path along with the cleanup that removes it. The
// timestamp is 2s in, or a tenth of the way through a video too short for
// that to land inside it.
func extractVideoFrame(ctx context.Context, videoPath string) (string, func(), error) {
	if !videoutil.Available() {
		return "", nil, ErrFFmpegUnavailable
	}

	probeCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	seekTs := 2 * time.Second
	if info, probeErr := videoutil.Probe(probeCtx, videoPath); probeErr == nil {
		if tenth := info.Duration / 10; info.Duration < 20*time.Second && tenth < seekTs {
			seekTs = tenth
		}
	}

	tmpFile, tmpErr := os.CreateTemp("", "vthumb-*.jpg")
	if tmpErr != nil {
		return "", nil, fmt.Errorf("video thumb temp file: %w", tmpErr)
	}
	tmpFile.Close()
	framePath := tmpFile.Name()
	cleanup := func() { os.Remove(framePath) }

	extractCtx, extractCancel := context.WithTimeout(ctx, 30*time.Second)
	defer extractCancel()
	if extractErr := videoutil.ExtractFrame(extractCtx, videoPath, seekTs, framePath); extractErr != nil {
		cleanup()
		return "", nil, fmt.Errorf("extract video frame: %w", extractErr)
	}
	return framePath, cleanup, nil
}
