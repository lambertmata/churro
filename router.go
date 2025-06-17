package churro

import (
	"context"
	"net/http"
	"path"
)

type HttpMethod string

const (
	MethodGet     HttpMethod = http.MethodGet
	MethodPost    HttpMethod = http.MethodPost
	MethodPut     HttpMethod = http.MethodPut
	MethodPatch   HttpMethod = http.MethodPatch
	MethodDelete  HttpMethod = http.MethodDelete
	MethodHead    HttpMethod = http.MethodHead
	MethodOption  HttpMethod = http.MethodOptions
	MethodConnect HttpMethod = http.MethodConnect
	MethodTrace   HttpMethod = http.MethodTrace
)

type Middleware func(next http.Handler) http.Handler

type RouterErrorHandler func(r *http.Request, w http.ResponseWriter, err error)

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
	// middlewares stores middlewares specific to this router level
	middlewares []Middleware
	// errorHandler is an optional error handler invoked when a handler or validator return error. Triggered only by errors occurring in type routes.
	errorHandler *RouterErrorHandler
}

type Mux interface {
	AddRoute(route *RouteHandler)
	RemoveRoute(method HttpMethod, path string)
	Match(method HttpMethod, path string) (*RouteHandler, error)
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

// collectMiddlewaresChain collects middlewares from root to current router
func (r *Router) collectMiddlewaresChain() []Middleware {
	var chain []Middleware
	var routers []*Router

	// Collect router hierarchy from current to root
	current := r
	for current != nil {
		routers = append(routers, current)
		current = current.parentGroup
	}

	// Apply middlewares from root to current (reverse order)
	for i := len(routers) - 1; i >= 0; i-- {
		chain = append(chain, routers[i].middlewares...)
	}

	return chain
}

func (r *Router) Middlewares(middleware ...Middleware) GroupCloser {
	//r.middlewares = append(r.middlewares, middleware...)
	r.middlewares = append(r.middlewares, middleware...)
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

	routeHandler, err := r.mux.Match(HttpMethod(req.Method), req.URL.Path)

	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	ctx := context.WithValue(req.Context(), RouterContext{}, routeHandler.ParamValues)

	r.applyMiddlewares(routeHandler.Handler, routeHandler.Middlewares).ServeHTTP(w, req.WithContext(ctx))

}

type RouterContext struct{}

func GetPathParams(req *http.Request) map[string]string {
	if ctx := req.Context().Value(RouterContext{}); ctx != nil {
		return ctx.(map[string]string)
	}
	return nil
}

func GetPathParam(req *http.Request, name string) string {
	if params := GetPathParams(req); params != nil {
		if val, ok := params[name]; ok {
			return val
		}
	}
	return ""
}

func (r *Router) ErrorHandler(handler RouterErrorHandler) {
	r.errorHandler = &handler
}
