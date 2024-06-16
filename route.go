package churro

import (
	"net/http"
	"strings"
)

type Route struct {

	// Method is the HTTP method of the route
	Method HttpMethod

	// Path is the actual path that is defined when creating the Route value
	Path string

	// FullPath is the computed Path that can include prefixes inherited from Route Group(s)
	FullPath string

	// middlewares the route level middlewares that will be applied to the route
	middlewares []Middleware

	// handler the route http.Handler that will be run on this route
	handler http.Handler

	// router is the Router holding the Route. It is filled when it is added the a router.
	router *Router
}

func NewRoute(method HttpMethod, path string, handler http.Handler) *Route {
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return &Route{Method: method, Path: path, handler: handler, FullPath: path}
}

func (r *Route) Middlewares(middlewares ...Middleware) {
	r.middlewares = append(r.middlewares, middlewares...)
}
