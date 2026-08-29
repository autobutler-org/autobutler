package v0_books

import "github.com/autobutler-org/quark/pkg/util/serverutil"

// Router for /api/v0/books endpoints
// Registers the /books route

type router struct{}

func NewRouter() serverutil.Router {
	return &router{}
}

func (r *router) Routes() []*serverutil.Route {
	return []*serverutil.Route{
		listBooksRoute,
	}
}
