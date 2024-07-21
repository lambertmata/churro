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

// Router is used mainly to build routes.
type Router struct {
	routes []*Route
	mux    Mux
	// children contains child Router(s) when a Router is a parentGroup
	children []*Router
	// prefix is used when a Router is a Router parentGroup and has an optional prefix
	prefix string
	// fullPrefix is computed
	fullPrefix string
	// parentGroup is the parent Router holding the Router, when used in a parentGroup. Is nil in root router.
	parentGroup *Router
}

type Mux interface {
	AddRoute(route *RouteHandler)
	RemoveRoute(method HttpMethod, path string)
	Match(method HttpMethod, path string) (*RouteHandler, []Middleware, error)
	AddMiddleware(method HttpMethod, path string, middleware ...Middleware) error
	AddRouterMiddleware(path string, middleware ...Middleware) error
}

func NewRouter() *Router {
	return &Router{
		mux: NewRadixTree(),
	}
}

func UpdateRouterPrefixes(router *Router) {
	curNode := router

	var fullPrefix string

	for curNode != nil {
		fullPrefix = path.Join(curNode.prefix, fullPrefix)
		curNode = curNode.parentGroup
	}

	router.fullPrefix = fullPrefix
}

func UpdateRouteFullPaths(route *Route) {
	curNode := route.router

	var fullPrefix string

	for curNode != nil {
		fullPrefix = path.Join(curNode.prefix, fullPrefix)
		curNode = curNode.parentGroup
	}

	route.fullPath = path.Join(fullPrefix, route.path)
}

// UpdateRoutesPaths adjusts all the Router routes to include prefixes from parent groups
func (r *Router) UpdateRoutesPaths() {
	for _, route := range r.Routes() {
		UpdateRouteFullPaths(route)
		r.mux.RemoveRoute(route.Method, route.fullPath)
		r.mux.AddRoute(route.RouteHandler())
	}
}

func (r *Router) AddRoute(route *Route) {
	route.router = r
	r.routes = append(r.routes, route)
	UpdateRouteFullPaths(route)
	r.mux.AddRoute(route.RouteHandler())
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
	groupRouter := &Router{mux: r.mux}
	r.AddGroup(groupRouter)
	f(groupRouter)
	return groupRouter
}

// Prefix sets the current Router prefix. All routes defined in sub Router(s) will be prefixed.
func (r *Router) Prefix(prefix string) {
	r.prefix = prefix
	UpdateRouterPrefixes(r)
	// After setting the prefix, we need to compute all the new prefixed paths of the Route(s)
	r.UpdateRoutesPaths()
}

func (r *Router) Middlewares(middleware ...Middleware) GroupCloser {
	//r.middlewares = append(r.middlewares, middleware...)
	r.mux.AddRouterMiddleware(r.fullPrefix, middleware...)
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
		// Take our routes {
		routes = append(routes, curRouter.routes...)
	}
	return routes
}

// applyMiddlewares takes and [http.Handler] and runs all the given middlewares.
func (r *Router) applyMiddlewares(handler http.Handler, middlewares []Middleware) http.Handler {
	for _, middleware := range middlewares {
		handler = middleware(handler)
	}
	return handler
}

func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {

	routeHandler, middlewares, err := r.mux.Match(HttpMethod(req.Method), req.URL.Path)

	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	r.applyMiddlewares(routeHandler.Handler, middlewares).ServeHTTP(w, req)

}
