package v1_books

import "autobutler/pkg/util/serverutil"

// Router for /api/v1/books endpoints
// Registers the /books route

type router struct{}

func NewRouter() serverutil.Router {
	return &router{}
}

func (r *router) Routes() []*serverutil.Route {
	return []*serverutil.Route{
		getBooksRoute,
	}
}
