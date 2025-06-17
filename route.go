package churro

import (
	"net/http"
	"strings"
)

type Route struct {

	// Method is the HTTP method of the route
	Method HttpMethod

	// path is the actual path that is defined when creating the Route value
	path string

	// fullPath is the computed path that can include prefixes inherited from Route Group(s)
	fullPath string

	// handler the route http.Handler that will be run on this route
	handler http.Handler

	// router is the Router holding the Route. It is filled when it is added the a router.
	router *Router

	// Matchers contains the path param Matchers defined with [Route.Matches].
	Matchers map[string]string

	// middlewares stores middlewares specific to this router level
	middlewares []Middleware
}

func NewRoute(method HttpMethod, path string, handler http.Handler) *Route {

	// Making sure path starts always with one "/"
	path = strings.TrimPrefix(path, "/")

	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	return &Route{
		Method:   method,
		path:     path,
		handler:  handler,
		fullPath: path,
	}
}

func (r *Route) Middlewares(middlewares ...Middleware) {
	r.middlewares = append(r.middlewares, middlewares...)
}

// Matches defines a regex for a Route path param.
func (r *Route) Matches(pathParam, regex string) *Route {
	if pathParam == "" || regex == "" {
		return r
	}
	if r.Matchers == nil {
		r.Matchers = make(map[string]string)
	}
	r.Matchers[pathParam] = regex
	return r
}

func (r *Route) RouteHandler() *RouteHandler {

	// Collect middlewares: router chain + route-specific
	routerMiddlewares := r.router.collectMiddlewaresChain()
	allMiddlewares := append(routerMiddlewares, r.middlewares...)

	return &RouteHandler{
		Path:        r.fullPath,
		Method:      r.Method,
		Matcher:     r.Matchers,
		Handler:     r.handler,
		Middlewares: allMiddlewares,
	}
}
