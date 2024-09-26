package churro

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
)

type Context[BodyType any, QueryType any, HeadersType any] struct {
	Req     *http.Request
	Res     http.ResponseWriter
	Body    BodyType
	Query   QueryType
	Headers HeadersType
}

type ProblemDetailsError struct {
	Status int      `json:"status"`
	Type   string   `json:"type"`
	Title  string   `json:"title"`
	Detail string   `json:"detail"`
	Errors []string `json:"errors"`
	Err    error
}

func (e *ProblemDetailsError) Error() string {
	return e.Title
}

func Request[Body, Query, Headers, Response any](router *Router, method HttpMethod, path string, handler func(ctx *Context[Body, Query, Headers]) (Response, error)) *Route {

	route := NewRoute(method, path, http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {

		ctx := Context[Body, Query, Headers]{
			Req: req,
			Res: w,
		}
		var problemDetailsError *ProblemDetailsError

		var err error

		if headerValidationErr := ReadValidatedHeader(req, &ctx.Headers); headerValidationErr != nil {

			err = headerValidationErr

		} else if bodyValidationErr := ReadValidatedBody(req, &ctx.Body); bodyValidationErr != nil {

			err = bodyValidationErr

		} else if queryValidationErr := ReadValidatedQuery(req, &ctx.Query); queryValidationErr != nil {

			err = queryValidationErr

		}

		if err != nil && errors.As(err, &problemDetailsError) {
			w.Header().Set("Content-Type", "application/problem+json")
			w.Header().Set("Content-Type", "application/problem+json")
			w.WriteHeader(problemDetailsError.Status)
			json.NewEncoder(w).Encode(problemDetailsError)
			return
		}

		res, err := handler(&ctx)

		if err != nil {
			slog.Error("error", "err", err.Error())
			return
		}

		json.NewEncoder(w).Encode(res)

	}))

	router.AddRoute(route)

	return route
}

func Get[Body, Query, Headers, Response any](router *Router, path string, handler func(ctx *Context[Body, Query, Headers]) (Response, error)) *Route {
	return Request(router, MethodGet, path, handler)
}

func Put[Body, Query, Headers, Response any](router *Router, path string, handler func(ctx *Context[Body, Query, Headers]) (Response, error)) *Route {
	return Request(router, MethodPut, path, handler)
}

func Post[Body, Query, Headers, Response any](router *Router, path string, handler func(ctx *Context[Body, Query, Headers]) (Response, error)) *Route {
	return Request(router, MethodPost, path, handler)
}

func Patch[Body, Query, Headers, Response any](router *Router, path string, handler func(ctx *Context[Body, Query, Headers]) (Response, error)) *Route {
	return Request(router, MethodPatch, path, handler)
}

func Delete[Body, Query, Headers, Response any](router *Router, path string, handler func(ctx *Context[Body, Query, Headers]) (Response, error)) *Route {
	return Request(router, MethodDelete, path, handler)
}

func Connect[Body, Query, Headers, Response any](router *Router, path string, handler func(ctx *Context[Body, Query, Headers]) (Response, error)) *Route {
	return Request(router, MethodConnect, path, handler)
}

func Trace[Body, Query, Headers, Response any](router *Router, path string, handler func(ctx *Context[Body, Query, Headers]) (Response, error)) *Route {
	return Request(router, MethodTrace, path, handler)
}

func Head[Body, Query, Headers, Response any](router *Router, path string, handler func(ctx *Context[Body, Query, Headers]) (Response, error)) *Route {
	return Request(router, MethodHead, path, handler)
}

func Option[Body, Query, Headers, Response any](router *Router, path string, handler func(ctx *Context[Body, Query, Headers]) (Response, error)) *Route {
	return Request(router, MethodOption, path, handler)
}
