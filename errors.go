package churro

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
)

// ProblemDetailsError implements RFC 7807 Problem Details for HTTP APIs
// https://tools.ietf.org/html/rfc7807
type ProblemDetailsError struct {
	// Type contains a URI reference that identifies the problem type
	Type string `json:"type"`
	// Title is a short, human-readable summary of the problem type
	Title string `json:"title"`
	// Status is the HTTP status code for this occurrence of the problem
	Status int `json:"status"`
	// Detail is a human-readable explanation specific to this occurrence
	Detail string `json:"detail,omitempty"`
	// Instance is a URI reference that identifies the specific occurrence
	Instance string `json:"instance,omitempty"`
	// Errors contains validation or other detailed errors
	Errors []string `json:"errors,omitempty"`
	// Extensions allows for additional problem-specific fields
	Extensions map[string]interface{} `json:"-"`
	// Err is the underlying error (not serialized)
	Err error `json:"-"`
}

// Error implements the error interface
func (e *ProblemDetailsError) Error() string {
	if e.Detail != "" {
		return e.Detail
	}
	return e.Title
}

// Unwrap returns the underlying error
func (e *ProblemDetailsError) Unwrap() error {
	return e.Err
}

// MarshalJSON implements custom JSON marshaling to include extensions
func (e *ProblemDetailsError) MarshalJSON() ([]byte, error) {
	type Alias ProblemDetailsError

	// Create a map for the base fields
	result := map[string]interface{}{
		"type":   e.Type,
		"title":  e.Title,
		"status": e.Status,
	}

	// Add optional fields
	if e.Detail != "" {
		result["detail"] = e.Detail
	}
	if e.Instance != "" {
		result["instance"] = e.Instance
	}
	if len(e.Errors) > 0 {
		result["errors"] = e.Errors
	}

	// Add extensions
	for key, value := range e.Extensions {
		result[key] = value
	}

	return json.Marshal(result)
}

// NewProblemDetails creates a new ProblemDetailsError with sensible defaults
func NewProblemDetails(status int, title, detail string) *ProblemDetailsError {
	return &ProblemDetailsError{
		Type:   "about:blank",
		Title:  title,
		Status: status,
		Detail: detail,
	}
}

// NewValidationError creates a validation error with detailed field errors
func NewValidationError(errors []string) *ProblemDetailsError {
	return &ProblemDetailsError{
		Type:   "validation-error",
		Title:  "Validation Failed",
		Status: http.StatusBadRequest,
		Detail: "Request validation failed",
		Errors: errors,
	}
}

// NewNotFoundError creates a 404 error
func NewNotFoundError(detail string) *ProblemDetailsError {
	if detail == "" {
		detail = "The requested resource was not found"
	}
	return &ProblemDetailsError{
		Type:   "about:blank",
		Title:  "Not Found",
		Status: http.StatusNotFound,
		Detail: detail,
	}
}

// NewMethodNotAllowedError creates a 405 error
func NewMethodNotAllowedError(detail string) *ProblemDetailsError {
	if detail == "" {
		detail = "The requested method is not allowed for this resource"
	}
	return &ProblemDetailsError{
		Type:   "about:blank",
		Title:  "Method Not Allowed",
		Status: http.StatusMethodNotAllowed,
		Detail: detail,
	}
}

// NewInternalServerError creates a 500 error
func NewInternalServerError(detail string) *ProblemDetailsError {
	if detail == "" {
		detail = "An internal server error occurred"
	}
	return &ProblemDetailsError{
		Type:   "about:blank",
		Title:  "Internal Server Error",
		Status: http.StatusInternalServerError,
		Detail: detail,
	}
}

// NewBadRequestError creates a 400 error
func NewBadRequestError(detail string) *ProblemDetailsError {
	if detail == "" {
		detail = "The request is malformed or invalid"
	}
	return &ProblemDetailsError{
		Type:   "about:blank",
		Title:  "Bad Request",
		Status: http.StatusBadRequest,
		Detail: detail,
	}
}

// NewUnauthorizedError creates a 401 error
func NewUnauthorizedError(detail string) *ProblemDetailsError {
	if detail == "" {
		detail = "Authentication is required to access this resource"
	}
	return &ProblemDetailsError{
		Type:   "about:blank",
		Title:  "Unauthorized",
		Status: http.StatusUnauthorized,
		Detail: detail,
	}
}

// NewForbiddenError creates a 403 error
func NewForbiddenError(detail string) *ProblemDetailsError {
	if detail == "" {
		detail = "Access to this resource is forbidden"
	}
	return &ProblemDetailsError{
		Type:   "about:blank",
		Title:  "Forbidden",
		Status: http.StatusForbidden,
		Detail: detail,
	}
}

// NewConflictError creates a 409 error
func NewConflictError(detail string) *ProblemDetailsError {
	if detail == "" {
		detail = "The request conflicts with the current state of the resource"
	}
	return &ProblemDetailsError{
		Type:   "about:blank",
		Title:  "Conflict",
		Status: http.StatusConflict,
		Detail: detail,
	}
}

// WithType sets the problem type URI
func (e *ProblemDetailsError) WithType(problemType string) *ProblemDetailsError {
	e.Type = problemType
	return e
}

// WithInstance sets the problem instance URI
func (e *ProblemDetailsError) WithInstance(instance string) *ProblemDetailsError {
	e.Instance = instance
	return e
}

// WithExtension adds a custom extension field
func (e *ProblemDetailsError) WithExtension(key string, value interface{}) *ProblemDetailsError {
	if e.Extensions == nil {
		e.Extensions = make(map[string]interface{})
	}
	e.Extensions[key] = value
	return e
}

// WithCause wraps an underlying error
func (e *ProblemDetailsError) WithCause(err error) *ProblemDetailsError {
	e.Err = err
	return e
}

// WriteProblemDetails writes a ProblemDetailsError as an HTTP response
func WriteProblemDetails(w http.ResponseWriter, err error) bool {
	var problemDetails *ProblemDetailsError
	if errors.As(err, &problemDetails) {
		if problemDetails.Type == "" {
			problemDetails.Type = "about:blank"
		}

		w.Header().Set("Content-Type", "application/problem+json")
		w.WriteHeader(problemDetails.Status)

		if encErr := json.NewEncoder(w).Encode(problemDetails); encErr != nil {
			// Fallback to a simple error response if JSON encoding fails
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
		return true
	}
	return false
}

// NewError creates a new ProblemDetailsError (for backward compatibility)
// Deprecated: Use NewProblemDetails instead
func NewError(status int, title string, err error) error {
	return &ProblemDetailsError{
		Type:   "about:blank",
		Title:  title,
		Status: status,
		Err:    err,
	}
}

// IsError checks if an error is a ProblemDetailsError
func IsError(err error) bool {
	var problemDetails *ProblemDetailsError
	return errors.As(err, &problemDetails)
}

// GetErrorStatus returns the HTTP status code from a ProblemDetailsError, or 500 for other errors
func GetErrorStatus(err error) int {
	var problemDetails *ProblemDetailsError
	if errors.As(err, &problemDetails) {
		return problemDetails.Status
	}
	return http.StatusInternalServerError
}

// ErrorFromStatus creates a generic error based on HTTP status code
func ErrorFromStatus(status int, detail string) *ProblemDetailsError {
	var title string
	switch status {
	case http.StatusBadRequest:
		title = "Bad Request"
	case http.StatusUnauthorized:
		title = "Unauthorized"
	case http.StatusForbidden:
		title = "Forbidden"
	case http.StatusNotFound:
		title = "Not Found"
	case http.StatusMethodNotAllowed:
		title = "Method Not Allowed"
	case http.StatusConflict:
		title = "Conflict"
	case http.StatusUnprocessableEntity:
		title = "Unprocessable Entity"
	case http.StatusInternalServerError:
		title = "Internal Server Error"
	default:
		title = fmt.Sprintf("HTTP %d", status)
	}

	return NewProblemDetails(status, title, detail)
}
