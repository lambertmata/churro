package churro

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/lambertmata/churro/reflector"
	"github.com/lambertmata/churro/validator"
	"net/http"
	"reflect"
	"strings"
)

// collectValidationErrors extracts all validation errors from a compound error
func collectValidationErrors(err error) []string {
	if err == nil {
		return nil
	}

	var errorList []string

	// Handle compound errors by extracting individual error messages
	if joinedErr, ok := err.(interface{ Unwrap() []error }); ok {
		for _, subErr := range joinedErr.Unwrap() {
			errorList = append(errorList, collectValidationErrors(subErr)...)
		}
		return errorList
	}

	// Extract field validation errors with context
	var fieldValidationError *validator.FieldValidationError
	if errors.As(err, &fieldValidationError) {
		errorList = append(errorList, fmt.Sprintf("Field '%s': %s", fieldValidationError.Field, fieldValidationError.Error()))
		return errorList
	}

	// Add the error message as-is if it's not a field validation error
	errorList = append(errorList, err.Error())
	return errorList
}

// wrapProblemDetailsError wraps non-nil error into ProblemDetailsError.
func wrapProblemDetailsError(err error) error {

	if err == nil {
		return nil
	}

	var fieldValidationError *validator.FieldValidationError
	var ruleParamsError *validator.RuleParamsError
	var unknownError *validator.UnknownRuleError

	// Collect detailed error messages
	errorMessages := collectValidationErrors(err)

	if errors.As(err, &fieldValidationError) {
		return NewValidationError(errorMessages).WithCause(err)
	} else if errors.As(err, &ruleParamsError) {
		return NewProblemDetails(http.StatusBadRequest, "Validation Error", "Request validation failed").
			WithType("validation-error").
			WithExtension("errors", errorMessages).
			WithCause(err)
	} else if errors.As(err, &unknownError) {
		return NewProblemDetails(http.StatusInternalServerError, "Unknown Validation Error", err.Error()).
			WithType("validation-error").
			WithCause(err)
	}

	// Generic error fallback
	return NewInternalServerError(err.Error()).WithCause(err)

}

func readValidatedBody(req *http.Request, bodyRef *reflect.Value) error {

	contentTypeParts := strings.Split(req.Header.Get("Content-Type"), ";")

	var contentType string

	if len(contentTypeParts) > 0 {
		contentType = contentTypeParts[0]
	}

	switch contentType {
	case "application/json":
		if err := json.NewDecoder(req.Body).Decode(bodyRef.Interface()); err != nil {

			var typeError *json.UnmarshalTypeError

			if errors.As(err, &typeError) {
				return &ProblemDetailsError{
					Status: http.StatusBadRequest,
					Title:  "Validation error",
					Detail: fmt.Sprintf("Expected type %s for %s field", typeError.Type, strings.ToLower(typeError.Field)),
					Err:    err,
				}

			}

			return &ProblemDetailsError{
				Status: http.StatusBadRequest,
				Title:  "Validation error",
				Detail: "Expected valid JSON",
				Err:    err,
			}

		}

		if err := validator.NewValidator().Validate(bodyRef.Interface()); err != nil {
			return wrapProblemDetailsError(err)
		}

	case "multipart/form-data":
		if err := req.ParseMultipartForm(10 << 20); err != nil {

			return &ProblemDetailsError{
				Status: http.StatusBadRequest,
				Title:  "Validation error",
				Detail: "Failed parsing multipart form",
				Err:    err,
			}
		}

		if err := reflector.ReadStringSlicesMapIntoStruct(bodyRef, req.MultipartForm.Value); err != nil {
			return &ProblemDetailsError{
				Status: http.StatusInternalServerError,
				Title:  "Validation error",
				Detail: "Failed parsing multipart form",
				Err:    err,
			}
		}

		for key, _ := range req.MultipartForm.File {

			file, _, err := req.FormFile(key)

			if err != nil {
				continue
			}

			err = reflector.FillStructFieldWithReaderBytes(bodyRef, key, file)

		}

		if err := validator.NewValidator().Validate(bodyRef.Elem().Interface()); err != nil {
			return wrapProblemDetailsError(err)
		}

	default:
		return &ProblemDetailsError{
			Status: http.StatusBadRequest,
			Title:  "Bad Request",
			Detail: "Content type not supported: " + contentType,
		}
	}

	return nil
}

func readValidatedHeader(req *http.Request, refHeader *reflect.Value) error {

	if err := reflector.ReadStringSlicesMapIntoStruct(refHeader, req.Header); err != nil {
		return fmt.Errorf("failed reading header values: %w", err)
	}

	return wrapProblemDetailsError(validator.NewValidator().Validate(refHeader.Interface()))
}

// readValidatedQuery reads query parameters from req, copy the contents into `refQuery` struct and applies
// validation using validator.
func readValidatedQuery(req *http.Request, refQuery *reflect.Value) error {

	if err := reflector.ReadStringSlicesMapIntoStruct(refQuery, req.URL.Query()); err != nil {
		return fmt.Errorf("failed reading query params: %w", err)
	}

	return wrapProblemDetailsError(validator.NewValidator().Validate(refQuery.Interface()))
}

// readPathParams reads path parameters from req, copy the contents into `refPathParams` struct and applies
// validation using validator.
func readPathParams(req *http.Request, refPathParams *reflect.Value) error {

	if err := reflector.ReadStringMapIntoStruct(refPathParams, GetPathParams(req)); err != nil {
		return fmt.Errorf("failed reading path params: %w", err)
	}

	return wrapProblemDetailsError(validator.NewValidator().Validate(refPathParams.Interface()))
}
