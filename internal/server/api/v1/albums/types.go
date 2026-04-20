package v1_albums

// AlbumJSON is the JSON representation of a photo album.
type AlbumJSON struct {
	ID        int64       `json:"id"`
	Name      string      `json:"name"`
	ParentID  *int64      `json:"parentId"`
	SmartType *string     `json:"smartType,omitempty"`
	CreatedAt string      `json:"createdAt"`
	UpdatedAt string      `json:"updatedAt"`
	ItemCount int64       `json:"itemCount"`
	Children  []AlbumJSON `json:"children,omitempty"`
}

// AlbumItemJSON is the JSON representation of a photo inside an album.
type AlbumItemJSON struct {
	ID           int64  `json:"id"`
	AlbumID      int64  `json:"albumId"`
	DeviceSerial string `json:"deviceSerial"`
	RelPath      string `json:"relPath"`
	AddedAt      string `json:"addedAt"`
}
