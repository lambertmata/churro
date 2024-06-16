package churro

import (
	"net/http"
	"path"
)

type HttpMethod string

const (
	Get     HttpMethod = http.MethodGet
	Post    HttpMethod = http.MethodPost
	Put     HttpMethod = http.MethodPut
	Patch   HttpMethod = http.MethodPatch
	Delete  HttpMethod = http.MethodDelete
	Head    HttpMethod = http.MethodHead
	Option  HttpMethod = http.MethodOptions
	Connect HttpMethod = http.MethodConnect
	Trace   HttpMethod = http.MethodTrace
)

type Middleware func(next http.Handler) http.Handler

type GroupCloser interface {
	Prefix(string)
	Middlewares(middleware ...Middleware) GroupCloser
}

type Router struct {
	routes []*Route

	mux Mux

	// children contains child Router(s) when a Router is a parentGroup
	children []*Router

	// prefix is used when a Router is a Router parentGroup and has an optional prefix
	prefix string

	// middleware router level middlewares. Will be applied to all route children and router children.
	middleware []Middleware

	// parentGroup is the parent Router holding the Router, when used in a parentGroup. Is nil in root router.
	parentGroup *Router
}

type Mux interface {
	AddRoute(route *Route)
	RemoveRoute(route *Route)
}

func NewRouter() *Router {
	return &Router{}
}

// adjustRoutePath adjusts the Route path to include prefixes from parent groups
func (r *Router) adjustRoutePath(route *Route) {

	// Here we walk up the tree until parent is nil
	curRouter := route.router

	var prefixPath string

	for curRouter != nil {
		prefixPath = path.Join(curRouter.prefix, prefixPath)
		curRouter = curRouter.parentGroup
	}

	route.FullPath = path.Join(prefixPath, route.Path)
}

// adjustRoutesPaths adjusts all the Router routes
func (r *Router) adjustRoutesPaths() {
	routes := r.Routes()
	for i := range r.Routes() {
		r.adjustRoutePath(routes[i])
	}
}

func (r *Router) AddRoute(route *Route) {
	route.router = r
	r.adjustRoutePath(route)
	r.routes = append(r.routes, route)
}

func (r *Router) Request(method HttpMethod, path string, handler http.HandlerFunc) *Route {
	route := NewRoute(method, path, handler)
	r.AddRoute(route)
	return route
}

func (r *Router) Get(path string, handler http.HandlerFunc) *Route {
	return r.Request(Get, path, handler)
}

func (r *Router) Post(path string, handler http.HandlerFunc) *Route {
	return r.Request(Post, path, handler)
}

func (r *Router) Put(path string, handler http.HandlerFunc) *Route {
	return r.Request(Put, path, handler)
}

func (r *Router) Patch(path string, handler http.HandlerFunc) *Route {
	return r.Request(Patch, path, handler)
}

func (r *Router) Delete(path string, handler http.HandlerFunc) *Route {
	return r.Request(Delete, path, handler)
}

func (r *Router) Connect(path string, handler http.HandlerFunc) *Route {
	return r.Request(Connect, path, handler)
}

func (r *Router) Trace(path string, handler http.HandlerFunc) *Route {
	return r.Request(Trace, path, handler)
}

func (r *Router) Head(path string, handler http.HandlerFunc) *Route {
	return r.Request(Head, path, handler)
}

func (r *Router) Option(path string, handler http.HandlerFunc) *Route {
	return r.Request(Option, path, handler)
}

// AddGroup adds the router as a sub Router
func (r *Router) AddGroup(gRouter *Router) {
	// Set the sub Router a reference to its parent Router
	gRouter.parentGroup = r
	r.children = append(r.children, gRouter)
}

// Group creates a new sub Router.
func (r *Router) Group(f func(gRouter *Router)) GroupCloser {
	groupRouter := NewRouter()
	r.AddGroup(groupRouter)
	f(groupRouter)
	return groupRouter
}

// Prefix sets the current Route prefix. All routes defined in sub Router(s) will be prefixed.
func (r *Router) Prefix(prefix string) {
	r.prefix = prefix
	// After setting the prefix, we need to compute all the new prefixed paths of the Route(s)
	r.adjustRoutesPaths()
}

func (r *Router) Middlewares(middleware ...Middleware) GroupCloser {
	r.middleware = append(r.middleware, middleware...)
	return r
}

// Routes returns current Router routes recursively.
func (r *Router) Routes() []*Route {
	var routes []*Route

	// Here we traverse the router routes using DFS approach. First we take the current route routes
	// then the children routers routes and so on.

	// we initialize a stack with the current router
	routersStack := []*Router{r}

	for len(routersStack) > 0 {

		// Pop a router from the stack (get and remove from slice)
		curRouter := routersStack[len(routersStack)-1]
		routersStack = routersStack[:len(routersStack)-1]

		// Push child routers if any
		if curRouter.children != nil {
			routersStack = append(routersStack, curRouter.children...)
		}

		// Take our routes
		for _, route := range curRouter.routes {
			routes = append(routes, route)
		}
	}

	return routes
}

func (r *Router) GetRoutMiddlewares(route *Route) []Middleware {
	curRouter := route.router
	middlewares := route.middlewares

	for curRouter != nil {
		// Prepend the parent middlewares
		middlewares = append(curRouter.middleware, middlewares...)
		curRouter = curRouter.parentGroup
	}

	return middlewares
}

func (r *Router) applyRouteMiddlewares(route *Route) {

	// Applies the middlewares starting from the router root up to the enclosing router middlewares. The applies the
	// route middlewares
	middlewares := r.GetRoutMiddlewares(route)
	wrappedHandler := route.handler

	for _, middleware := range middlewares {
		wrappedHandler = middleware(wrappedHandler)
	}

}
