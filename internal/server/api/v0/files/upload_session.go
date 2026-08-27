package v0_files

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/autobutler-org/quark/pkg/util/ctxutil"
	"github.com/autobutler-org/quark/pkg/util/deputil"
	"github.com/autobutler-org/quark/pkg/util/serverutil"
	"github.com/autobutler-org/quark/pkg/util/uploadutil"
	"github.com/autobutler-org/quark/pkg/vfs"

	"github.com/gin-gonic/gin"
)

// Resumable chunked uploads (#1629). A large file no longer rides on one
// request that a dropped connection costs in full: the client opens a session,
// PUTs the bytes a chunk at a time, and after an interruption asks what landed
// and carries on from there. Small files still take the multipart endpoint in
// upload_files.go, where chunking would be pure overhead.
const (
	// uploadOffsetHeader carries the committed offset back with a 409, so a
	// client that guessed wrong resyncs without a second round trip.
	uploadOffsetHeader = "X-Upload-Offset"
	// sessionIDParam is the path parameter naming the session.
	sessionIDParam = "sessionId"
)

type createUploadSessionRequest struct {
	RootDir   string `json:"rootDir"`
	FileName  string `json:"fileName"`
	TotalSize int64  `json:"totalSize"`
	Serial    string `json:"serial"`
	Overwrite bool   `json:"overwrite"`
}

type createUploadSessionResponse struct {
	SessionID string    `json:"sessionId"`
	Offset    int64     `json:"offset"`
	ExpiresAt time.Time `json:"expiresAt"`
}

type uploadChunkResponse struct {
	SessionID string `json:"sessionId"`
	Offset    int64  `json:"offset"`
	Complete  bool   `json:"complete"`
	Path      string `json:"path,omitempty"`
}

type uploadSessionStatusResponse struct {
	SessionID string    `json:"sessionId"`
	Offset    int64     `json:"offset"`
	TotalSize int64     `json:"totalSize"`
	FileName  string    `json:"fileName"`
	RootDir   string    `json:"rootDir"`
	ExpiresAt time.Time `json:"expiresAt"`
}

// createUploadSession godoc
// @Summary Open a resumable upload session
// @Description Reserve a session for one file; the bytes follow as chunks on PUT
// @Tags files
// @Accept json
// @Produce json
// @Param session body createUploadSessionRequest true "File the session will carry"
// @Success 200 {object} createUploadSessionResponse "OK"
// @Failure 400 {object} serverutil.Response "Bad Request"
// @Failure 500 {object} serverutil.Response "Internal Server Error"
// @Router /files/upload-session [post]
func createUploadSession(c *gin.Context) *serverutil.Response {
	deps, ok := ctxutil.Get[deputil.Dependencies](c, "deps")
	if !ok {
		return serverutil.InternalServerError(nil)
	}
	store := deps.UploadSessions()
	if store == nil {
		return serverutil.InternalServerError(errors.New("upload sessions are not configured"))
	}

	var request createUploadSessionRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		return serverutil.BadRequest(err)
	}

	result, err := store.CreateSession(uploadutil.CreateSessionParams{
		Destination: uploadDestination(deps),
		RootDir:     request.RootDir,
		FileName:    request.FileName,
		TotalSize:   request.TotalSize,
		Serial:      request.Serial,
		Overwrite:   request.Overwrite,
	})
	if err != nil {
		return uploadSessionError(c, err)
	}
	return serverutil.Ok().WithData(createUploadSessionResponse{
		SessionID: result.SessionID,
		Offset:    result.Offset,
		ExpiresAt: result.ExpiresAt.UTC(),
	})
}

// uploadSessionChunk godoc
// @Summary Send one chunk of a resumable upload
// @Description Append the chunk named by Content-Range; the last one commits the file
// @Tags files
// @Accept octet-stream
// @Produce json
// @Param sessionId path string true "Upload session id"
// @Param Content-Range header string true "Byte range of this chunk, e.g. bytes 0-8388607/20971520"
// @Success 200 {object} uploadChunkResponse "OK"
// @Failure 400 {object} serverutil.Response "Bad Request"
// @Failure 404 {object} serverutil.Response "Not Found"
// @Failure 409 {object} serverutil.Response "Conflict"
// @Router /files/upload-session/{sessionId} [put]
func uploadSessionChunk(c *gin.Context) *serverutil.Response {
	deps, ok := ctxutil.Get[deputil.Dependencies](c, "deps")
	if !ok {
		return serverutil.InternalServerError(nil)
	}
	store := deps.UploadSessions()
	if store == nil {
		return serverutil.InternalServerError(errors.New("upload sessions are not configured"))
	}

	chunk, err := uploadutil.ParseContentRange(c.GetHeader("Content-Range"))
	if err != nil {
		return uploadSessionError(c, err)
	}

	result, err := store.WriteChunk(uploadutil.WriteChunkParams{
		Ctx:         c.Request.Context(),
		Destination: uploadDestination(deps),
		SessionID:   c.Param(sessionIDParam),
		Range:       chunk,
		Body:        c.Request.Body,
	})
	if err != nil {
		return uploadSessionError(c, err)
	}
	return serverutil.Ok().WithData(uploadChunkResponse{
		SessionID: result.SessionID,
		Offset:    result.Offset,
		Complete:  result.Complete,
		Path:      result.Path,
	})
}

// getUploadSession godoc
// @Summary Ask what a resumable upload has committed
// @Description Returns the committed offset a resuming client continues from
// @Tags files
// @Produce json
// @Param sessionId path string true "Upload session id"
// @Success 200 {object} uploadSessionStatusResponse "OK"
// @Failure 404 {object} serverutil.Response "Not Found"
// @Router /files/upload-session/{sessionId} [get]
func getUploadSession(c *gin.Context) *serverutil.Response {
	deps, ok := ctxutil.Get[deputil.Dependencies](c, "deps")
	if !ok {
		return serverutil.InternalServerError(nil)
	}
	store := deps.UploadSessions()
	if store == nil {
		return serverutil.InternalServerError(errors.New("upload sessions are not configured"))
	}

	result, err := store.DescribeSession(uploadutil.DescribeSessionParams{
		SessionID: c.Param(sessionIDParam),
	})
	if err != nil {
		return uploadSessionError(c, err)
	}
	return serverutil.Ok().WithData(uploadSessionStatusResponse{
		SessionID: result.SessionID,
		Offset:    result.Offset,
		TotalSize: result.TotalSize,
		FileName:  result.FileName,
		RootDir:   result.RootDir,
		ExpiresAt: result.ExpiresAt.UTC(),
	})
}

// deleteUploadSession godoc
// @Summary Abandon a resumable upload session
// @Description Drops the session and the bytes staged for it
// @Tags files
// @Produce json
// @Param sessionId path string true "Upload session id"
// @Success 200 {object} serverutil.Response "OK"
// @Failure 404 {object} serverutil.Response "Not Found"
// @Router /files/upload-session/{sessionId} [delete]
func deleteUploadSession(c *gin.Context) *serverutil.Response {
	deps, ok := ctxutil.Get[deputil.Dependencies](c, "deps")
	if !ok {
		return serverutil.InternalServerError(nil)
	}
	store := deps.UploadSessions()
	if store == nil {
		return serverutil.InternalServerError(errors.New("upload sessions are not configured"))
	}

	if _, err := store.DeleteSession(uploadutil.DeleteSessionParams{
		SessionID: c.Param(sessionIDParam),
	}); err != nil {
		return uploadSessionError(c, err)
	}
	return serverutil.Ok()
}

// uploadSessionError maps what uploadutil reports onto the status codes the
// client contract is written against. The offset mismatch is the only one that
// carries state back in a header: everything the client needs to resync from a
// 409 is in the response it already has.
func uploadSessionError(c *gin.Context, err error) *serverutil.Response {
	var mismatch *uploadutil.OffsetMismatchError
	switch {
	case errors.As(err, &mismatch):
		c.Header(uploadOffsetHeader, strconv.FormatInt(mismatch.Offset, 10))
		return serverutil.Conflict(err)
	case errors.Is(err, uploadutil.ErrSessionNotFound):
		return serverutil.NotFound(err)
	case errors.Is(err, uploadutil.ErrInvalidRange),
		errors.Is(err, uploadutil.ErrInvalidRequest),
		// The file already exists and the caller did not ask to overwrite it.
		// The multipart endpoint answers 400 for that, and the same upload
		// arriving in chunks should not answer something else.
		errors.Is(err, vfs.ErrConflict):
		return serverutil.BadRequest(err)
	default:
		return serverutil.InternalServerError(err)
	}
}

// The session endpoints sit at /files/upload-session, not the more obvious
// /files/upload/session. POST /files//upload/*rootDir looks like it reserves a
// different subtree, but gin runs the group prefix and the route path through
// path.Join, which collapses that deliberate double slash — so the route
// actually registered is /api/v0/files/upload/*rootDir, and any static child
// under /files/upload/ panics at startup with a wildcard conflict. Verified by
// TestUploadSessionRoutesRegisterWithoutConflict.
var createUploadSessionRoute = serverutil.ApiRoute(
	http.MethodPost, "/files/upload-session", createUploadSession,
)

var uploadSessionChunkRoute = serverutil.ApiRoute(
	http.MethodPut, "/files/upload-session/:"+sessionIDParam, uploadSessionChunk,
)

var getUploadSessionRoute = serverutil.ApiRoute(
	http.MethodGet, "/files/upload-session/:"+sessionIDParam, getUploadSession,
)

var deleteUploadSessionRoute = serverutil.ApiRoute(
	http.MethodDelete, "/files/upload-session/:"+sessionIDParam, deleteUploadSession,
)
