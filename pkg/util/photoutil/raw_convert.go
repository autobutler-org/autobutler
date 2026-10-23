package photoutil

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/jpeg"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/autobutler-org/quark/pkg/util/storageutil"
)

// rawToolTimeout bounds each converter. A conversion runs while its request
// holds an IO-semaphore slot, so a tool that hangs on a malformed file would
// otherwise keep that slot until the server restarts (#2752).
const rawToolTimeout = 30 * time.Second

// RawToJPEG converts a camera RAW file to a JPEG image by extracting
// the embedded preview. Tries dcraw first, then exiftool, each for at most
// rawToolTimeout.
func RawToJPEG(filePath string) (image.Image, error) {
	for _, convert := range []func(context.Context, string) (image.Image, error){rawViaDcraw, rawViaExiftool} {
		ctx, cancel := context.WithTimeout(context.Background(), rawToolTimeout)
		img, err := convert(ctx, filePath)
		cancel()
		if err == nil {
			return img, nil
		}
	}

	return nil, fmt.Errorf("no RAW converter available (install dcraw or exiftool)")
}

// WriteRawAsJPEG converts a camera RAW file and encodes it straight onto w.
// It used to return the JPEG as a []byte, which put a whole converted image on
// the heap for every concurrent request before a single byte was written
// (#1723).
func WriteRawAsJPEG(w io.Writer, filePath string, quality int) error {
	img, err := RawToJPEG(filePath)
	if err != nil {
		return err
	}
	if err := jpeg.Encode(w, img, &jpeg.Options{Quality: quality}); err != nil {
		return fmt.Errorf("encode JPEG: %w", err)
	}
	return nil
}

// IsRawFile checks whether a file path has a camera RAW extension.
func IsRawFile(filePath string) bool {
	return storageutil.IsRawImageExtension(filepath.Ext(filePath))
}

// IsRawConverterAvailable returns true if at least one RAW conversion
// tool is on the PATH.
func IsRawConverterAvailable() bool {
	for _, cmd := range []string{"dcraw", "exiftool"} {
		if _, err := exec.LookPath(cmd); err == nil {
			return true
		}
	}
	return false
}

// rawViaDcraw extracts the embedded JPEG preview using dcraw -c -e.
func rawViaDcraw(ctx context.Context, filePath string) (image.Image, error) {
	dcraw, err := exec.LookPath("dcraw")
	if err != nil {
		return nil, fmt.Errorf("dcraw not found")
	}

	cmd := rawTool(ctx, dcraw, "-c", "-e", filePath)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("dcraw: %w", err)
	}

	img, _, err := DecodeImage(bytes.NewReader(out))
	if err != nil {
		return nil, fmt.Errorf("decode dcraw output: %w", err)
	}
	return img, nil
}

// rawViaExiftool extracts the embedded preview using exiftool.
func rawViaExiftool(ctx context.Context, filePath string) (image.Image, error) {
	exiftool, err := exec.LookPath("exiftool")
	if err != nil {
		return nil, fmt.Errorf("exiftool not found")
	}

	tmpDir, err := os.MkdirTemp("", "quark-raw-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tmpDir)

	previewPath := filepath.Join(tmpDir, "preview.jpg")
	cmd := rawTool(ctx, exiftool, "-b", "-PreviewImage", "-w!", previewPath, filePath)
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("exiftool: %w", err)
	}

	// exiftool writes to <basename>.jpg in tmpDir
	matches, _ := filepath.Glob(filepath.Join(tmpDir, "*.jpg"))
	if len(matches) == 0 {
		return nil, fmt.Errorf("exiftool produced no preview")
	}

	f, err := os.Open(matches[0])
	if err != nil {
		return nil, err
	}
	defer f.Close()

	img, _, err := DecodeImage(f)
	if err != nil {
		return nil, fmt.Errorf("decode exiftool output: %w", err)
	}
	return img, nil
}

// rawTool is exec.CommandContext with a WaitDelay, so a tool whose child keeps
// the output pipe open cannot hold Output open after the tool is killed.
func rawTool(ctx context.Context, name string, args ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.WaitDelay = time.Second
	return cmd
}
