package churro

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"reflect"
)

type Context[BodyType, QueryType, HeadersType, PathParamsType any] struct {
	Req                 *http.Request
	Res                 http.ResponseWriter
	ValidatedBody       BodyType
	ValidatedQuery      QueryType
	ValidatedHeaders    HeadersType
	ValidatedPathParams PathParamsType
}

func (c *Context[BodyType, QueryType, HeadersType, PathParams]) PathParam(name string) string {
	return GetPathParam(c.Req, name)
}

func (c *Context[BodyType, QueryType, HeadersType, PathParams]) PathParams() map[string]string {
	return GetPathParams(c.Req)
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

// WriteResult writes res to response writer when type is []byte, JSON in all the other cases.
func WriteResult(w http.ResponseWriter, res any) error {

	refRes := reflect.ValueOf(res)

	if refRes.Kind() == reflect.Ptr {
		refRes = refRes.Elem()
	}

	// Special case for byte slices (binary data)
	if refRes.Kind() == reflect.Slice && refRes.Type().Elem().Kind() == reflect.Uint8 {
		// Write bytes directly
		if _, err := w.Write(res.([]byte)); err != nil {
			return fmt.Errorf("failed to write binary data %w", err)
		}
		return nil
	}

	// For all other types, use JSON encoder
	if err := json.NewEncoder(w).Encode(res); err != nil {
		return fmt.Errorf("failed to write json data %w", err)
	}

	return nil
}

func Request[Body, Query, Headers, Response, PathParams any](router *Router, method HttpMethod, path string, handler func(ctx *Context[Body, Query, Headers, PathParams]) (Response, error)) *Route {

	route := NewRoute(method, path, http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {

		ctx := Context[Body, Query, Headers, PathParams]{
			Req: req,
			Res: w,
		}
		var problemDetailsError *ProblemDetailsError

		var err error

		defer func() {
			if err != nil && errors.As(err, &problemDetailsError) {
				w.Header().Set("Content-Type", "application/problem+json")
				w.Header().Set("Content-Type", "application/problem+json")
				w.WriteHeader(problemDetailsError.Status)
				json.NewEncoder(w).Encode(problemDetailsError)
				return
			}
		}()

		if pathParamsValidationErr := ReadValidatedPathParams(req, &ctx.ValidatedPathParams); pathParamsValidationErr != nil {
			err = pathParamsValidationErr
			return
		}

		if headerValidationErr := ReadValidatedHeader(req, &ctx.ValidatedHeaders); headerValidationErr != nil {
			err = headerValidationErr
			return
		}

		if bodyValidationErr := ReadValidatedBody(req, &ctx.ValidatedBody); bodyValidationErr != nil {
			err = bodyValidationErr
			return
		}

		if queryValidationErr := ReadValidatedQuery(req, &ctx.ValidatedQuery); queryValidationErr != nil {
			err = queryValidationErr
			return
		}

		res, err := handler(&ctx)

		if err != nil {
			slog.Error("error", "err", err.Error())
			return
		}

		if err := WriteResult(w, res); err != nil {
			slog.Error("failed to write response", "err", err.Error())
		}

	}))

	route.reqType = reflect.TypeOf((*Body)(nil))
	route.resType = reflect.TypeOf((*Response)(nil))
	route.headerType = reflect.TypeOf((*Headers)(nil))
	route.queryType = reflect.TypeOf((*Query)(nil))
	route.pathParamsType = reflect.TypeOf((*PathParams)(nil))

	router.AddRoute(route)

	return route
}

func Get[Body, Query, Headers, Response, PathParams any](router *Router, path string, handler func(ctx *Context[Body, Query, Headers, PathParams]) (Response, error)) *Route {
	return Request(router, MethodGet, path, handler)
}

func Put[Body, Query, Headers, Response, PathParams any](router *Router, path string, handler func(ctx *Context[Body, Query, Headers, PathParams]) (Response, error)) *Route {
	return Request(router, MethodPut, path, handler)
}

func Post[Body, Query, Headers, Response, PathParams any](router *Router, path string, handler func(ctx *Context[Body, Query, Headers, PathParams]) (Response, error)) *Route {
	return Request(router, MethodPost, path, handler)
}

func Patch[Body, Query, Headers, Response, PathParams any](router *Router, path string, handler func(ctx *Context[Body, Query, Headers, PathParams]) (Response, error)) *Route {
	return Request(router, MethodPatch, path, handler)
}

func Delete[Body, Query, Headers, Response, PathParams any](router *Router, path string, handler func(ctx *Context[Body, Query, Headers, PathParams]) (Response, error)) *Route {
	return Request(router, MethodDelete, path, handler)
}

func Connect[Body, Query, Headers, Response, PathParams any](router *Router, path string, handler func(ctx *Context[Body, Query, Headers, PathParams]) (Response, error)) *Route {
	return Request(router, MethodConnect, path, handler)
}

func Trace[Body, Query, Headers, Response, PathParams any](router *Router, path string, handler func(ctx *Context[Body, Query, Headers, PathParams]) (Response, error)) *Route {
	return Request(router, MethodTrace, path, handler)
}

func Head[Body, Query, Headers, Response, PathParams any](router *Router, path string, handler func(ctx *Context[Body, Query, Headers, PathParams]) (Response, error)) *Route {
	return Request(router, MethodHead, path, handler)
}

func Option[Body, Query, Headers, Response, PathParams any](router *Router, path string, handler func(ctx *Context[Body, Query, Headers, PathParams]) (Response, error)) *Route {
	return Request(router, MethodOption, path, handler)
}
