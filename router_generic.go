package churro

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/lambertmata/churro/reflector"
	"log/slog"
	"net/http"
	"reflect"
	"io"
)

type ProblemDetailsError struct {
	Status int      `json:"status"`
	Type   string   `json:"type"`
	Title  string   `json:"title"`
	Detail string   `json:"detail"`
	Errors []string `json:"errors"`
	Err    error    `json:"-"`
}

func (e *ProblemDetailsError) Error() string {
	return e.Title
}

func NewError(status int, title string, err error) error {
	return &ProblemDetailsError{
		Status: status,
		Title:  title,
		Err:    err,
	}
}

// WriteResult writes res to response writer when type is []byte, JSON in all the other cases.
func WriteResult(w http.ResponseWriter, res any) error {

	refRes := reflect.ValueOf(res)

	isResponseHandler := false

	if refRes.IsNil() {
		return nil
	}

	if refRes.Kind() == reflect.Ptr {

		if refRes.IsNil() {
			w.WriteHeader(http.StatusNoContent)
			return errors.New("failed to return nil res")
		}

		isResponseHandler = refRes.Type().Implements(reflect.TypeOf((*responseTypeProvider)(nil)).Elem())
		refRes = refRes.Elem()
	}

	if isResponseHandler {
		hr := refRes.Interface().(responseTypeProvider)
		res = hr.Payload()

		for _, m := range hr.Middlewares() {
			m(w, &res)
		}
	}

	// Special case for byte slices (binary data)
	switch bytes := res.(type) {
	case io.Reader:
		if _, err := io.Copy(w, bytes); err != nil {
			return fmt.Errorf("failed to write reader data %w", err)
		}
		return nil
	case string:
		if _, err := w.Write([]byte(bytes)); err != nil {
			return fmt.Errorf("failed to write string data %w", err)
		}
		return nil
	case []byte:
		if _, err := w.Write(bytes); err != nil {
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

type RequestHandler[Response any] func() (func(ctx RequestContext) (Response, error), RequestContext)

// writeProblemDetailsError writes problem details error as a json response if err is ProblemDetailsError
func writeProblemDetailsError(w http.ResponseWriter, err error) {
	var problemDetailsError *ProblemDetailsError
	if err != nil && errors.As(err, &problemDetailsError) {
		if problemDetailsError.Type == "" {
			problemDetailsError.Type = "about:blank"
		}
		w.Header().Set("Content-Type", "application/problem+json")
		w.WriteHeader(problemDetailsError.Status)
		json.NewEncoder(w).Encode(problemDetailsError)
		return
	}
}

type GenericRouteHandler[RequestCtx RequestContext] func(ctx RequestCtx) error

func Request[RequestCtx RequestContext](router *Router, method HttpMethod, path string, handler GenericRouteHandler[RequestCtx]) *Route {

	// Here we allow a user to define a typed route handler using one of the available RequestContext types, depending
	// on which fields are needed.
	// We have:
	// - ContextWithBody to have typed Body
	// - ContextWithBodyAndQuery to have Body and Query typed
	// - RawContext to type Body, Query, Headers and PathParams
	// - Context to not type anything at all.
	// Since I still haven't found a way to do things using interfaces, I have to rely on reflection to derive the types
	// from the defined context type in the handler parameter.
	// We sample the parameter by taking the first handler parameter and accessing Body, QueryParams, PathParams and
	// Headers in Embedded RawContext.
	route := NewRoute(method, path, http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {

		// Sampling the type from the handler ctx parameter
		refCtx := reflector.NewValFromFuncParameter(reflect.ValueOf(handler), 0).Elem()

		// From the sampled value of the RequestContext type of the parameter, we get the RawContext and extract all the
		// fields that we need.
		refRawCtx := refCtx.Field(0)

		// Now we have to initialize all the fields which will be pointing to nil
		reflector.InitStructPointerField(&refRawCtx, "Body")
		reflector.InitStructPointerField(&refRawCtx, "QueryParams")
		reflector.InitStructPointerField(&refRawCtx, "PathParams")
		reflector.InitStructPointerField(&refRawCtx, "Headers")

		// The fields now are initialized and can be filled with data from the request payload
		refBody := refRawCtx.FieldByName("Body")
		refQueryParams := refRawCtx.FieldByName("QueryParams")
		refPathParams := refRawCtx.FieldByName("PathParams")
		refHeader := refRawCtx.FieldByName("Headers")

		hasBody := !reflector.IsAny(refBody)
		hasHeader := !reflector.IsAny(refHeader)
		hasQueryParams := !reflector.IsAny(refQueryParams)
		hasPathParams := !reflector.IsAny(refPathParams)

		var err error

		if hasPathParams {
			if pathParamsValidationErr := readPathParams(req, &refPathParams); pathParamsValidationErr != nil {
				err = errors.Join(pathParamsValidationErr)
			}
		}

		if hasHeader {
			if headerValidationErr := readValidatedHeader(req, &refHeader); headerValidationErr != nil {
				err = errors.Join(headerValidationErr)
			}
		}

		if hasBody {
			if bodyValidationErr := readValidatedBody(req, &refBody); bodyValidationErr != nil {
				err = errors.Join(err, bodyValidationErr)
			}
		}

		if hasQueryParams {
			if queryValidationErr := readValidatedQuery(req, &refQueryParams); queryValidationErr != nil {
				err = errors.Join(err, queryValidationErr)
			}

		}

		if err != nil {
			if errorHandler := router.errorHandler; errorHandler != nil {
				(*errorHandler)(req, w, err)
			}
			// If the handler returned an ProblemDetailsError, we write the response automatically
			writeProblemDetailsError(w, err)
			return
		}

		refCtx.FieldByName("Res").Set(reflect.ValueOf(w))
		refCtx.FieldByName("Req").Set(reflect.ValueOf(req))

		finalCtx := refCtx.Addr().Interface().(RequestCtx)
		handlerErr := handler(finalCtx)

		if handlerErr != nil {
			if errorHandler := router.errorHandler; errorHandler != nil {
				(*errorHandler)(req, w, handlerErr)
			}
			writeProblemDetailsError(w, handlerErr)
			return
		}

		if err := WriteResult(w, finalCtx.getResponse()); err != nil {
			slog.Error("failed to write response", "err", err.Error())
		}

	}))

	router.AddRoute(route)

	return route
}

func Get[Context RequestContext](router *Router, path string, handler GenericRouteHandler[Context]) *Route {
	return Request(router, MethodGet, path, handler)
}

func Put[Context RequestContext](router *Router, path string, handler GenericRouteHandler[Context]) *Route {
	return Request(router, MethodPut, path, handler)
}

func Post[Context RequestContext](router *Router, path string, handler GenericRouteHandler[Context]) *Route {
	return Request(router, MethodPost, path, handler)
}

func Patch[Context RequestContext](router *Router, path string, handler GenericRouteHandler[Context]) *Route {
	return Request(router, MethodPatch, path, handler)
}

func Delete[Context RequestContext](router *Router, path string, handler GenericRouteHandler[Context]) *Route {
	return Request(router, MethodDelete, path, handler)
}

func Connect[Context RequestContext](router *Router, path string, handler GenericRouteHandler[Context]) *Route {
	return Request(router, MethodConnect, path, handler)
}

func Trace[Context RequestContext](router *Router, path string, handler GenericRouteHandler[Context]) *Route {
	return Request(router, MethodTrace, path, handler)
}

func Head[Context RequestContext](router *Router, path string, handler GenericRouteHandler[Context]) *Route {
	return Request(router, MethodHead, path, handler)
}

func Option[Context RequestContext](router *Router, path string, handler GenericRouteHandler[Context]) *Route {
	return Request(router, MethodOption, path, handler)
}

type ResponseMiddleware func(w http.ResponseWriter, res *any)

type HandlerResponse[ResponseType any] struct {
	payload     ResponseType
	middlewares []ResponseMiddleware
}

func (hr HandlerResponse[ResponseType]) ResponseType() reflect.Type {
	return reflect.TypeOf(hr.payload)
}

func (hr HandlerResponse[ResponseType]) Middlewares() []ResponseMiddleware {
	return hr.middlewares
}

func (hr HandlerResponse[ResponseType]) Payload() any {
	return hr.payload
}

type responseTypeProvider interface {
	ResponseType() reflect.Type
	Middlewares() []ResponseMiddleware
	Payload() any
}

func WithStatusCode(statusCode int) ResponseMiddleware {
	return func(w http.ResponseWriter, res *any) {
		w.WriteHeader(statusCode)
	}
}

func WithHeader(name, value string) ResponseMiddleware {
	return func(w http.ResponseWriter, res *any) {
		w.Header().Set(name, value)
	}
}

func WithContentType(contentType string) ResponseMiddleware {
	return WithHeader("Content-Type", contentType)
}

func WithJSONContentType() ResponseMiddleware {
	return WithHeader("Content-Type", "application/json")
}

func WithWrappedData() ResponseMiddleware {
	return func(w http.ResponseWriter, res *any) {
		wrapped := map[string]any{
			"data": *res,
		}
		*res = wrapped
	}
}
