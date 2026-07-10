package v1_vfs

import "time"

// NamespaceJSON is the JSON representation of a registered VFS namespace.
type NamespaceJSON struct {
	ID          string `json:"id"`
	Description string `json:"description"`
}

// FileInfoJSON is the JSON representation of a VFS FileInfo.
type FileInfoJSON struct {
	Name        string    `json:"name"`
	Path        string    `json:"path"`
	Size        int64     `json:"size"`
	IsDir       bool      `json:"is_dir"`
	MimeType    string    `json:"mime_type"`
	ModTime     time.Time `json:"mod_time"`
	ContentHash string    `json:"content_hash,omitempty"`
	Namespace   string    `json:"namespace"`
}
