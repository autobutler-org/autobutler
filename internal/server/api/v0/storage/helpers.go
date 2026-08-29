package v0_storage

import (
	"errors"

	"github.com/autobutler-org/quark/pkg/util/deviceutil"
	"github.com/autobutler-org/quark/pkg/util/serverutil"
)

// deviceError maps what deviceutil reports onto the status codes the client
// contract is written against. The message travels unchanged in every case, so
// the client keeps reading the same sentences it always has.
func deviceError(err error) *serverutil.Response {
	var (
		invalid      *deviceutil.InvalidRequestError
		unauthorized *deviceutil.UnauthorizedError
		notFound     *deviceutil.NotFoundError
	)
	switch {
	case errors.As(err, &invalid):
		return serverutil.BadRequest(err)
	case errors.As(err, &unauthorized):
		return serverutil.Unauthorized(err)
	case errors.As(err, &notFound):
		return serverutil.NotFound(err)
	}
	return serverutil.InternalServerError(err)
}
