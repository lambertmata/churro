package churro

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"reflect"
)

// ResponseMiddleware defines a function type for response middleware
type ResponseMiddleware func(w http.ResponseWriter, res *any)

// HandlerResponse wraps a response with middleware
type HandlerResponse[ResponseType any] struct {
	payload     ResponseType
	middlewares []ResponseMiddleware
}

// ResponseType returns the type of the response payload
func (hr HandlerResponse[ResponseType]) ResponseType() reflect.Type {
	return reflect.TypeOf(hr.payload)
}

// Middlewares returns the response middlewares
func (hr HandlerResponse[ResponseType]) Middlewares() []ResponseMiddleware {
	return hr.middlewares
}

// Payload returns the response payload
func (hr HandlerResponse[ResponseType]) Payload() any {
	return hr.payload
}

// responseTypeProvider interface for response handlers
type responseTypeProvider interface {
	ResponseType() reflect.Type
	Middlewares() []ResponseMiddleware
	Payload() any
}

// Response middleware functions

// WithStatusCode sets the HTTP status code
func WithStatusCode(statusCode int) ResponseMiddleware {
	return func(w http.ResponseWriter, res *any) {
		w.WriteHeader(statusCode)
	}
}

// WithHeader adds a custom header
func WithHeader(name, value string) ResponseMiddleware {
	return func(w http.ResponseWriter, res *any) {
		w.Header().Set(name, value)
	}
}

// WithContentType sets the Content-Type header
func WithContentType(contentType string) ResponseMiddleware {
	return WithHeader("Content-Type", contentType)
}

// WithJSONContentType sets Content-Type to application/json
func WithJSONContentType() ResponseMiddleware {
	return WithHeader("Content-Type", "application/json")
}

// WithWrappedData wraps the response in a "data" field
func WithWrappedData() ResponseMiddleware {
	return func(w http.ResponseWriter, res *any) {
		wrapped := map[string]any{
			"data": *res,
		}
		*res = wrapped
	}
}

// WithCacheControl sets cache control headers
func WithCacheControl(directive string) ResponseMiddleware {
	return WithHeader("Cache-Control", directive)
}

// WithNoCache sets no-cache headers
func WithNoCache() ResponseMiddleware {
	return WithCacheControl("no-cache, no-store, must-revalidate")
}

// WithCacheMaxAge sets cache max-age
func WithCacheMaxAge(seconds int) ResponseMiddleware {
	return WithCacheControl(fmt.Sprintf("max-age=%d", seconds))
}

// WriteResult writes a response to the HTTP response writer
// Handles different response types: io.Reader, string, []byte, and JSON
func WriteResult(w http.ResponseWriter, res any) error {
	// Handle nil response early to avoid reflection errors
	if res == nil {
		return nil
	}

	refRes := reflect.ValueOf(res)
	isResponseHandler := false

	if refRes.IsValid() && refRes.IsNil() {
		return nil
	}

	if refRes.Kind() == reflect.Ptr {
		if refRes.IsNil() {
			w.WriteHeader(http.StatusNoContent)
			return fmt.Errorf("failed to return nil response")
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

	// Handle different response types
	switch v := res.(type) {
	case io.Reader:
		if _, err := io.Copy(w, v); err != nil {
			return fmt.Errorf("failed to write reader data: %w", err)
		}
		return nil
	case string:
		if _, err := w.Write([]byte(v)); err != nil {
			return fmt.Errorf("failed to write string data: %w", err)
		}
		return nil
	case []byte:
		if _, err := w.Write(v); err != nil {
			return fmt.Errorf("failed to write binary data: %w", err)
		}
		return nil
	}

	// For all other types, use JSON encoder
	if err := json.NewEncoder(w).Encode(res); err != nil {
		return fmt.Errorf("failed to write JSON data: %w", err)
	}

	return nil
}

// SendJSON is a helper to send JSON responses
func SendJSON(w http.ResponseWriter, data any, middlewares ...ResponseMiddleware) error {
	middlewares = append(middlewares, WithJSONContentType(), WithWrappedData())

	response := &HandlerResponse[any]{
		payload:     data,
		middlewares: middlewares,
	}

	return WriteResult(w, response)
}

// SendString is a helper to send string responses
func SendString(w http.ResponseWriter, data string, middlewares ...ResponseMiddleware) error {
	response := &HandlerResponse[string]{
		payload:     data,
		middlewares: middlewares,
	}

	return WriteResult(w, response)
}

// SendBytes is a helper to send byte responses
func SendBytes(w http.ResponseWriter, data []byte, middlewares ...ResponseMiddleware) error {
	response := &HandlerResponse[[]byte]{
		payload:     data,
		middlewares: middlewares,
	}

	return WriteResult(w, response)
}

// SendStream is a helper to send streaming responses
func SendStream(w http.ResponseWriter, data io.Reader, middlewares ...ResponseMiddleware) error {
	response := &HandlerResponse[io.Reader]{
		payload:     data,
		middlewares: middlewares,
	}

	return WriteResult(w, response)
}
