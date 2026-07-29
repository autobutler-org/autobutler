package vfs

import (
	"context"
	"errors"
	"io"
	"time"
)

// VFS is the host-side interface backing a namespace.
type VFS interface {
	List(ctx context.Context, path string, filter *ListFilter) ([]FileInfo, error)
	Stat(ctx context.Context, path string) (FileInfo, error)
	Open(ctx context.Context, path string) (io.ReadCloser, error)
	Write(ctx context.Context, path string, r io.Reader, opts WriteOptions) error
	Delete(ctx context.Context, path string, opts DeleteOptions) error
	MkdirAll(ctx context.Context, path string) error
	Move(ctx context.Context, src, dst string) error
	Watch(ctx context.Context, path string) (<-chan WatchEvent, error)
}

type FileInfo struct {
	Name        string    `json:"name"`
	Path        string    `json:"path"`
	Size        int64     `json:"size"`
	IsDir       bool      `json:"is_dir"`
	MimeType    string    `json:"mime_type"`
	ModTime     time.Time `json:"mod_time"`
	ContentHash string    `json:"content_hash"`
	Namespace   string    `json:"namespace"`
}

type Namespace struct {
	ID          string `json:"id"`
	PluginID    string `json:"plugin_id"`
	MountPath   string `json:"mount_path"`
	Description string `json:"description"`
}

type ListFilter struct {
	MimePrefix   string
	Recursive    bool
	MaxResults   int
	AfterPath    string
	SerialFilter []string // if non-empty, restrict to devices with these serials
}

type WriteOptions struct {
	ContentType  string
	IfNoneMatch  string
	ExpectedSize int64
}

type DeleteOptions struct {
	Recursive bool
}

type WatchEvent struct {
	Op   WatchOp  `json:"op"`
	Path string   `json:"path"`
	Info FileInfo `json:"info"`
}

type WatchOp string

const (
	WatchOpCreated  WatchOp = "created"
	WatchOpModified WatchOp = "modified"
	WatchOpDeleted  WatchOp = "deleted"
	WatchOpRenamed  WatchOp = "renamed"
)

var (
	ErrNotFound          = errors.New("vfs: not found")
	ErrNotEmpty          = errors.New("vfs: directory not empty")
	ErrPermissionDenied  = errors.New("vfs: permission denied")
	ErrWatchNotSupported = errors.New("vfs: watch not supported by this implementation")
	ErrNamespaceConflict = errors.New("vfs: namespace already registered")
	ErrConflict          = errors.New("vfs: conflict")
)
