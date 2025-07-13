// Package churro is a simple HTTP router with support for middleware,
// path parameters, and type-safe handlers.
//
// Basic usage:
//
//	router := churro.NewRouter()
//	router.Get("/users/{id}", func(w http.ResponseWriter, r *http.Request) {
//		userID := churro.GetPathParam(r, "id")
//		// handle request...
//	})
//	http.ListenAndServe(":8080", router)
//
// Type-safe handlers with automatic validation:
//
//	type CreateUser struct {
//		Name  string `json:"name" validate:"required"`
//		Email string `json:"email" validate:"required,email"`
//	}
//
//	churro.Post(router, "/users", func(ctx *churro.ContextWithBody[CreateUser]) error {
//		return ctx.SendJSON(map[string]string{"id": "123"})
//	})
//
// Features include path parameters, route groups, middleware support,
// request validation, and multiple response types.
package churro

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"path"
)

type HTTPMethod string

const (
	MethodGet     HTTPMethod = http.MethodGet
	MethodPost    HTTPMethod = http.MethodPost
	MethodPut     HTTPMethod = http.MethodPut
	MethodPatch   HTTPMethod = http.MethodPatch
	MethodDelete  HTTPMethod = http.MethodDelete
	MethodHead    HTTPMethod = http.MethodHead
	MethodOptions HTTPMethod = http.MethodOptions
	MethodConnect HTTPMethod = http.MethodConnect
	MethodTrace   HTTPMethod = http.MethodTrace
)

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
	// notFoundHandler is called when no route matches the request (404)
	notFoundHandler http.Handler
	// methodNotAllowedHandler is called when route exists but method is not allowed (405)
	methodNotAllowedHandler http.Handler
}

type Mux interface {
	AddRoute(route *RouteHandler) error
	RemoveRoute(method HTTPMethod, path string)
	Match(method HTTPMethod, path string) (*RouteHandler, error)
}

// Config holds configuration options for the Router
type Config struct {
	MaxRoutes              int
	CaseSensitive          bool
	StrictSlash            bool
	HandleMethodNotAllowed bool
}

// DefaultConfig returns a Config with sensible defaults
func DefaultConfig() Config {
	return Config{
		MaxRoutes:              1000,
		CaseSensitive:          false,
		StrictSlash:            false,
		HandleMethodNotAllowed: false,
	}
}

// RouterOption defines a function type for configuring Router options
type RouterOption func(*Router)

// WithMux sets a custom Mux implementation for the router
func WithMux(mux Mux) RouterOption {
	return func(r *Router) {
		r.mux = mux
	}
}

// WithConfig sets configuration options for the router
func WithConfig(config Config) RouterOption {
	return func(r *Router) {
		// Store config for future use
		// For now, we'll just validate the config
		if config.MaxRoutes <= 0 {
			config.MaxRoutes = DefaultConfig().MaxRoutes
		}
	}
}

// NewRouter creates a new Router with default RadixTree mux
func NewRouter(opts ...RouterOption) *Router {
	r := &Router{
		mux: NewRadixTree(),
	}

	// Apply options
	for _, opt := range opts {
		opt(r)
	}

	return r
}

// NewRouterWithMux creates a new Router with a custom Mux implementation
func NewRouterWithMux(mux Mux) *Router {
	if mux == nil {
		mux = NewRadixTree()
	}
	return &Router{
		mux: mux,
	}
}

func updateRouterPrefixes(router *Router) {
	curNode := router

	var fullPrefix string

	for curNode != nil {
		fullPrefix = path.Join(curNode.prefix, fullPrefix)
		curNode = curNode.parentGroup
	}

	router.fullPrefix = fullPrefix
}

func updateRouteFullPaths(route *Route) {
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
		updateRouteFullPaths(route)
		r.mux.RemoveRoute(route.Method, route.fullPath)
		if err := r.mux.AddRoute(route.RouteHandler()); err != nil {
			panic(fmt.Sprintf("failed to update route %s %s: %v", route.Method, route.fullPath, err))
		}
	}
}

func (r *Router) AddRoute(route *Route) error {
	route.router = r
	r.routes = append(r.routes, route)
	updateRouteFullPaths(route)
	return r.mux.AddRoute(route.RouteHandler())
}

func (r *Router) Request(method HTTPMethod, path string, handler http.HandlerFunc) *Route {
	route := NewRoute(method, path, handler)
	if err := r.AddRoute(route); err != nil {
		// Log the error but don't break the API and return the route anyway
		// Return route even if adding fails to maintain API consistency
		panic(fmt.Sprintf("failed to add route %s %s: %v", method, path, err))
	}
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
	updateRouterPrefixes(r)
	// Update all route paths to include the new prefix
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

	// Update all existing routes in the mux to include the new middlewares
	r.updateRoutesMiddlewares()

	return r
}

// updateRoutesMiddlewares updates all existing routes in the mux with the current middleware chain
func (r *Router) updateRoutesMiddlewares() {
	for _, route := range r.Routes() {
		if route.router != nil && route.router.mux != nil {
			route.router.mux.RemoveRoute(route.Method, route.fullPath)
			if err := route.router.mux.AddRoute(route.RouteHandler()); err != nil {
				panic(fmt.Sprintf("failed to update route middlewares %s %s: %v", route.Method, route.fullPath, err))
			}
		}
	}
}

// Routes returns current Router routes recursively.
func (r *Router) Routes() []*Route {
	var routes []*Route

	// Traverse router tree using DFS to collect all routes
	// Initialize stack with current router
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
	// Apply middlewares in reverse order so the first middleware added becomes the outermost wrapper
	for i := len(middlewares) - 1; i >= 0; i-- {
		handler = middlewares[i](handler)
	}
	return handler
}

// defaultNotFoundHandler returns a default 404 handler
func (r *Router) defaultNotFoundHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		err := NewNotFoundError("")
		WriteProblemDetails(w, err)
	})
}

// defaultMethodNotAllowedHandler returns a default 405 handler
func (r *Router) defaultMethodNotAllowedHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		err := NewMethodNotAllowedError("")
		WriteProblemDetails(w, err)
	})
}

func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	routeHandler, err := r.mux.Match(HTTPMethod(req.Method), req.URL.Path)

	if err != nil {
		// Determine error type and use appropriate handler
		var handler http.Handler

		if errors.Is(err, errRoutedMethodNotImplemented) {
			// 405 Method Not Allowed
			if r.methodNotAllowedHandler != nil {
				handler = r.methodNotAllowedHandler
			} else {
				handler = r.defaultMethodNotAllowedHandler()
			}
		} else {
			// 404 Not Found (errNodeNotFound or other errors)
			if r.notFoundHandler != nil {
				handler = r.notFoundHandler
			} else {
				handler = r.defaultNotFoundHandler()
			}
		}

		// Apply global middleware to error handlers so logging etc. works
		globalMiddlewares := r.collectMiddlewaresChain()
		r.applyMiddlewares(handler, globalMiddlewares).ServeHTTP(w, req)
		return
	}

	ctx := WithPathParams(req.Context(), routeHandler.ParamValues)
	r.applyMiddlewares(routeHandler.Handler, routeHandler.Middlewares).ServeHTTP(w, req.WithContext(ctx))
}

type contextKey string

const (
	pathParamsKey contextKey = "churro:path_params"
)

// WithPathParams adds path parameters to the request context
func WithPathParams(ctx context.Context, params map[string]string) context.Context {
	return context.WithValue(ctx, pathParamsKey, params)
}

// PathParamsFromContext retrieves path parameters from the request context
func PathParamsFromContext(ctx context.Context) (map[string]string, bool) {
	params, ok := ctx.Value(pathParamsKey).(map[string]string)
	return params, ok
}

// GetPathParams retrieves path parameters from the request context
func GetPathParams(req *http.Request) map[string]string {
	if params, ok := PathParamsFromContext(req.Context()); ok {
		return params
	}
	return nil
}

// GetPathParam retrieves a specific path parameter from the request context
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

// NotFoundHandler sets a custom handler for 404 (Not Found) responses.
// The handler will still run through the middleware chain.
func (r *Router) NotFoundHandler(handler http.Handler) {
	r.notFoundHandler = handler
}

// MethodNotAllowedHandler sets a custom handler for 405 (Method Not Allowed) responses.
// The handler will still run through the middleware chain.
func (r *Router) MethodNotAllowedHandler(handler http.Handler) {
	r.methodNotAllowedHandler = handler
}
