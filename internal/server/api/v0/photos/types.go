package v0_photos

import (
	"github.com/autobutler-org/quark/pkg/util/serverutil"
)

type router struct{}

func (r *router) Routes() []*serverutil.Route {
	return []*serverutil.Route{
		listPhotosRoute,
		getMetadataRoute,
		rotatePhotoRoute,
		copyPhotoRoute,
		listDuplicatesRoute,
	}
}

type copyPhotoRequest struct {
	RelPath string `json:"relPath" binding:"required"`
	Serial  string `json:"serial"`
}

type copyPhotoResponse struct {
	RelPath string `json:"relPath"`
}

type rotatePhotoRequest struct {
	RelPath          string `json:"relPath"          binding:"required"`
	Serial           string `json:"serial"`
	RotationQuarters int64  `json:"rotationQuarters"` // 0–3; 0 deletes the record
}
