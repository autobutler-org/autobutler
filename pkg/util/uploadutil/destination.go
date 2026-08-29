package uploadutil

import (
	"io"
	"mime/multipart"
	"path"
	"path/filepath"

	"github.com/autobutler-org/quark/pkg/util/eventbus"
	"github.com/autobutler-org/quark/pkg/util/storageutil"
	"github.com/autobutler-org/quark/pkg/vfs"
)

// filesNamespace is the VFS namespace holding the local file store. Registered
// by deputil.DefaultDependencies; absent in older deployments and in tests that
// exercise the StorageService directly.
const filesNamespace = "files"

// FilesVFS returns the VFS backing the local namespace, or nil when the write
// has to go through the StorageService instead: a named device serial routes
// past the VFS, and a deployment without the namespace has nothing to route to.
func (d Destination) FilesVFS(serial string) vfs.VFS {
	if serial != "" || d.Registry == nil {
		return nil
	}
	fsys, ok := d.Registry.Get(filesNamespace)
	if !ok {
		return nil
	}
	return fsys
}

// Writable reports whether an upload for this serial has anywhere to go. A
// session is worth opening only if the bytes it collects can eventually land.
func (d Destination) Writable(serial string) bool {
	return d.FilesVFS(serial) != nil || d.Storage != nil
}

// WriteFile streams one file into the destination and publishes the upload
// event the file index and the clients listen for.
func (d Destination) WriteFile(params WriteFileParams) (WriteFileResult, error) {
	// The client-supplied name never carries structure; rootDir does (#1603).
	fileName := filepath.Base(params.FileName)

	if fsys := d.FilesVFS(params.Serial); fsys != nil {
		if err := d.writeToVFS(fsys, params, fileName); err != nil {
			return WriteFileResult{}, err
		}
	} else if err := d.writeToStorageService(params, fileName); err != nil {
		return WriteFileResult{}, err
	}

	if d.EventBus != nil {
		d.EventBus.Publish(eventbus.Event{
			Kind: eventbus.EventUpload,
			Path: params.RootDir,
		})
	}
	return WriteFileResult{Path: path.Join(params.RootDir, fileName)}, nil
}

func (d Destination) writeToVFS(fsys vfs.VFS, params WriteFileParams, fileName string) error {
	if params.RootDir != "" {
		if err := fsys.MkdirAll(params.Ctx, params.RootDir); err != nil {
			return err
		}
	}
	opts := vfs.WriteOptions{}
	if !params.Overwrite {
		// vfs.ErrConflict comes back when the file already exists, which the
		// HTTP layer reports as a 400 the way the multipart endpoint does.
		opts.IfNoneMatch = "*"
	}
	return fsys.Write(params.Ctx, path.Join(params.RootDir, fileName), params.Reader, opts)
}

// writeToStorageService replays the file through the same multipart-streaming
// path POST /files/upload uses, so device routing and name-conflict handling
// (file_(1).ext) stay in one implementation instead of being copied here and
// drifting. The pipe keeps it streaming: only the copy buffer is ever in
// memory, which matters because this path exists for multi-gigabyte files.
func (d Destination) writeToStorageService(params WriteFileParams, fileName string) error {
	pr, pw := io.Pipe()
	mw := multipart.NewWriter(pw)

	go func() {
		part, err := mw.CreateFormFile("files", fileName)
		if err != nil {
			pw.CloseWithError(err)
			return
		}
		if _, err := io.Copy(part, params.Reader); err != nil {
			pw.CloseWithError(err)
			return
		}
		pw.CloseWithError(mw.Close())
	}()
	defer pr.Close()

	return d.Storage.UploadFilesStreamed(storageutil.UploadFilesStreamedParams{
		Reader:       multipart.NewReader(pr, mw.Boundary()),
		RootDir:      params.RootDir,
		DeviceSerial: params.Serial,
		Overwrite:    params.Overwrite,
	})
}
