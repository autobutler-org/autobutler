package fileutil

import (
	"context"
	"errors"
	"fmt"
	"io"
	"path"
	"path/filepath"
	"strings"

	"github.com/autobutler-org/quark/pkg/util/docreadutil"
	"github.com/autobutler-org/quark/pkg/util/eventbus"
	"github.com/autobutler-org/quark/pkg/util/uploadutil"
	"github.com/autobutler-org/quark/pkg/vfs"
)

// ConvertDocumentParams converts one word-processing document into the .qdoc
// the document editor opens, written beside the document it came from.
type ConvertDocumentParams struct {
	// Ctx bounds the read and the write.
	Ctx context.Context
	// Registry holds the namespace of the device Serial names.
	Registry vfs.Registry
	// EventBus is told about the file that appeared.
	EventBus *eventbus.Bus
	// FilePath is the files-relative path of the document.
	FilePath string
	// Serial identifies the device, empty for the internal one.
	Serial string
	// Overwrite lets the conversion replace a .qdoc that is already there.
	// Without it an existing file is reported as [vfs.ErrConflict] and left
	// alone.
	Overwrite bool
	// BeforeOverwrite, when set, is called with the .qdoc's path just before
	// the conversion replaces it, while the old content is still there to
	// snapshot.
	BeforeOverwrite func(path string)
}

// ConvertDocumentResult reports the .qdoc that was written and what the
// conversion left out.
type ConvertDocumentResult struct {
	// Path is where the new document landed, files-relative.
	Path string
	// Replaced is true when an existing .qdoc was overwritten, so nothing new
	// was created.
	Replaced bool
	// Truncated reports that the document was longer than a conversion
	// reads, so the .qdoc holds only its beginning.
	Truncated bool
	// Words counts the words that were converted.
	Words int
	// Warnings names what the .qdoc does not hold, in sentences a user can
	// read.
	Warnings []string
}

// IsDocumentPath reports whether a path names a word-processing document
// the document readers understand. The list grows with docreadutil's.
func IsDocumentPath(filePath string) bool {
	_, ok := docreadutil.FormatForName(filePath)
	return ok
}

// ConvertDocumentToQdoc reads the document at params.FilePath and writes it
// as a sibling .qdoc ("Open as Quark doc"). The document itself is left
// untouched.
//
// As [ConvertXlsxToQsheet] does, the conversion streams: the document is read
// in place, the Delta is piped into a temporary file beside it, and that file
// is moved into place only once the whole document has converted, so a
// document that fails halfway leaves nothing behind.
func ConvertDocumentToQdoc(params ConvertDocumentParams) (ConvertDocumentResult, error) {
	format, ok := docreadutil.FormatForName(params.FilePath)
	if !ok {
		return ConvertDocumentResult{}, &UnsupportedError{
			Err: fmt.Errorf("not a document Quark can convert: %s", filepath.Base(params.FilePath)),
		}
	}

	dir := path.Dir(cleanRelPath(params.FilePath))
	if dir == "." || dir == "/" {
		dir = ""
	}
	base := path.Base(cleanRelPath(params.FilePath))
	stem := strings.TrimSuffix(base, filepath.Ext(base))
	target := path.Join(dir, stem+".qdoc")

	fsys, err := FilesVFS(params.Registry, params.Serial)
	if err != nil {
		return ConvertDocumentResult{}, err
	}
	existed, err := exists(params.Ctx, fsys, target)
	if err != nil {
		return ConvertDocumentResult{}, err
	}
	if existed && !params.Overwrite {
		return ConvertDocumentResult{}, fmt.Errorf("%w: %s already exists", vfs.ErrConflict, target)
	}

	// openXlsx opens any file for random access; a document is a zip too.
	source, size, closer, err := openXlsx(params.Ctx, fsys, params.FilePath)
	if err != nil {
		return ConvertDocumentResult{}, err
	}
	defer closer.Close()

	dest := uploadutil.Destination{
		Registry: params.Registry,
		// The move below publishes the path that matters.
		EventBus: nil,
	}
	tempName := "." + stem + ".qdoc.converting"

	pr, pw := io.Pipe()
	type outcome struct {
		result docreadutil.ConvertResult
		err    error
	}
	converted := make(chan outcome, 1)
	go func() {
		result, err := docreadutil.Convert(docreadutil.ConvertParams{
			Ctx:    params.Ctx,
			Source: source,
			Size:   size,
			Format: format,
			Output: docreadutil.OutputDelta,
			Out:    pw,
		})
		pw.CloseWithError(err)
		converted <- outcome{result: result, err: err}
	}()

	_, writeErr := dest.WriteFile(uploadutil.WriteFileParams{
		Ctx:       params.Ctx,
		Reader:    pr,
		RootDir:   dir,
		FileName:  tempName,
		Serial:    params.Serial,
		Overwrite: true,
	})
	pr.CloseWithError(writeErr)
	done := <-converted

	tempPath := path.Join(dir, tempName)
	if done.err != nil {
		discardTemp(params.Ctx, fsys, tempPath)
		return ConvertDocumentResult{}, documentError(done.err)
	}
	if writeErr != nil {
		discardTemp(params.Ctx, fsys, tempPath)
		return ConvertDocumentResult{}, writeErr
	}

	if _, err := MoveFile(MoveFileParams{
		Ctx:             params.Ctx,
		Registry:        params.Registry,
		EventBus:        params.EventBus,
		OldFilePath:     tempPath,
		NewFilePath:     target,
		OldDeviceSerial: params.Serial,
		NewDeviceSerial: params.Serial,
		BeforeOverwrite: params.BeforeOverwrite,
	}); err != nil {
		discardTemp(params.Ctx, fsys, tempPath)
		return ConvertDocumentResult{}, err
	}

	warnings := done.result.Warnings
	if done.result.Truncated {
		warnings = append(warnings, "The document is too long to convert whole; only its beginning was converted")
	}
	return ConvertDocumentResult{
		Path:      target,
		Replaced:  existed,
		Truncated: done.result.Truncated,
		Words:     done.result.Words,
		Warnings:  warnings,
	}, nil
}

// documentError maps what docreadutil reports onto the errors handlers
// derive status codes from. A document that is malformed, past a limit, or
// not what its name says is the caller's file, not a server fault.
func documentError(err error) error {
	if errors.Is(err, docreadutil.ErrMalformed) || errors.Is(err, docreadutil.ErrTooLarge) ||
		errors.Is(err, docreadutil.ErrUnsupportedFormat) {
		return &UnsupportedError{Err: err}
	}
	return err
}
