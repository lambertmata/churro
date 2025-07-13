package churro

import (
	"errors"
	"fmt"
	"github.com/lambertmata/churro/reflector"
	"log/slog"
	"net/http"
	"reflect"
)

type RequestHandler[Response any] func() (func(ctx RequestContext) (Response, error), RequestContext)

type GenericRouteHandler[RequestCtx RequestContext] func(ctx RequestCtx) error

func Request[RequestCtx RequestContext](router *Router, method HTTPMethod, path string, handler GenericRouteHandler[RequestCtx]) *Route {

	// Create an HTTP handler that uses reflection to populate typed context fields
	// from request data (body, query params, headers, path params) and validates them.
	route := NewRoute(method, path, http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {

		// Get the context type from the handler parameter
		refCtx := reflector.NewValFromFuncParameter(reflect.ValueOf(handler), 0).Elem()

		// Extract the embedded RawContext
		refRawCtx := refCtx.Field(0)

		// Initialize pointer fields
		reflector.InitStructPointerField(&refRawCtx, "Body")
		reflector.InitStructPointerField(&refRawCtx, "QueryParams")
		reflector.InitStructPointerField(&refRawCtx, "PathParams")
		reflector.InitStructPointerField(&refRawCtx, "Headers")

		// Get references to typed fields for population
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
			// Write structured error response
			if !WriteProblemDetails(w, err) {
				// Fallback for non-structured errors
				WriteProblemDetails(w, NewBadRequestError("Request validation failed"))
			}
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
			if !WriteProblemDetails(w, handlerErr) {
				// Fallback for non-structured errors
				WriteProblemDetails(w, NewInternalServerError(handlerErr.Error()))
			}
			return
		}

		if err := WriteResult(w, finalCtx.getResponse()); err != nil {
			slog.Error("failed to write response", "err", err.Error())
		}

	}))

	if err := router.AddRoute(route); err != nil {
		panic(fmt.Sprintf("failed to add generic route %s %s: %v", method, path, err))
	}

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

func Options[Context RequestContext](router *Router, path string, handler GenericRouteHandler[Context]) *Route {
	return Request(router, MethodOptions, path, handler)
}
