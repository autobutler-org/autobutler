package v0_playback

import (
	"errors"

	"github.com/autobutler-org/quark/pkg/util/accessutil"
	"github.com/autobutler-org/quark/pkg/util/ctxutil"
	"github.com/autobutler-org/quark/pkg/util/deputil"
	"github.com/autobutler-org/quark/pkg/util/fileutil"
	"github.com/autobutler-org/quark/pkg/util/playbackutil"
	"github.com/autobutler-org/quark/pkg/util/serverutil"
	"github.com/autobutler-org/quark/pkg/vfs"
	"github.com/gin-gonic/gin"
)

// resolveTarget checks that the caller may read the file a request names and
// that it is a file on that device, so nobody stores state against a path
// that holds nothing. A non-nil response is the refusal to send.
func resolveTarget(c *gin.Context, deviceSerial, relPath string) (target, *serverutil.Response) {
	if relPath == "" {
		return target{}, serverutil.BadRequest(errors.New("relPath is required"))
	}
	deps, ok := ctxutil.Get[deputil.Dependencies](c, "deps")
	if !ok {
		return target{}, serverutil.InternalServerError(nil)
	}
	access, err := accessutil.LoadRequest(c, deps.Database(), deps.StorageService())
	if err != nil {
		return target{}, serverutil.InternalServerError(err)
	}
	if !access.Check(deviceSerial, relPath, accessutil.Read).Readable {
		return target{}, serverutil.NotFound(errNoAccess)
	}
	fsys, err := fileutil.FilesVFS(deps.VFSRegistry(), deviceSerial)
	var noDevice *fileutil.NotFoundError
	if errors.As(err, &noDevice) {
		return target{}, serverutil.NotFound(errNoAccess)
	}
	if err != nil {
		return target{}, serverutil.InternalServerError(err)
	}
	info, err := fsys.Stat(c.Request.Context(), relPath)
	// A path that climbs out of the files directory is refused by the
	// namespace, and names nothing the caller could play either.
	if errors.Is(err, vfs.ErrNotFound) || errors.Is(err, vfs.ErrPermissionDenied) || (err == nil && info.IsDir) {
		return target{}, serverutil.NotFound(errNoAccess)
	}
	if err != nil {
		return target{}, serverutil.InternalServerError(err)
	}
	return target{
		store:        deps.MetadataStore(),
		userID:       access.Principal().UserID,
		deviceSerial: deviceSerial,
		relPath:      relPath,
	}, nil
}

// stateResponse answers with a playback state, or with the failure that kept
// it from being read or stored.
func stateResponse(state playbackutil.State, err error) *serverutil.Response {
	if errors.Is(err, playbackutil.ErrInvalid) {
		return serverutil.BadRequest(err)
	}
	if err != nil {
		return serverutil.InternalServerError(err)
	}
	return serverutil.Ok().WithContentType(serverutil.ContentTypeJSON).WithData(state)
}
