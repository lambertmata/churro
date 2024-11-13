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

// WrapProblemDetailsError wraps non-nil error into ProblemDetailsError.
func WrapProblemDetailsError(err error) error {

	if err == nil {
		return nil
	}

	var fieldValidationError *validator.FieldValidationError
	var ruleParamsError *validator.RuleParamsError
	var unknownError *validator.UnknownRuleError

	problemDetailsError := ProblemDetailsError{
		Detail: err.Error(),
		Err:    err,
	}

	if errors.As(err, &fieldValidationError) {

		problemDetailsError.Status = http.StatusUnprocessableEntity
		problemDetailsError.Title = "Rule validation error"

	} else if errors.As(err, &ruleParamsError) {

		problemDetailsError.Status = http.StatusBadRequest
		problemDetailsError.Title = "Validation error"

	} else if errors.As(err, &unknownError) {

		problemDetailsError.Status = http.StatusInternalServerError
		problemDetailsError.Title = "Unknown validation error"

	} else {
		problemDetailsError.Status = http.StatusInternalServerError
		problemDetailsError.Title = "Internal server error"
	}

	return &problemDetailsError

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
			return WrapProblemDetailsError(err)
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
			return WrapProblemDetailsError(err)
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

	return WrapProblemDetailsError(validator.NewValidator().Validate(refHeader.Interface()))
}

// readValidatedQuery reads query parameters from req, copy the contents into `refQuery` struct and applies
// validation using validator.
func readValidatedQuery(req *http.Request, refQuery *reflect.Value) error {

	if err := reflector.ReadStringSlicesMapIntoStruct(refQuery, req.URL.Query()); err != nil {
		return fmt.Errorf("failed reading query params: %w", err)
	}

	return WrapProblemDetailsError(validator.NewValidator().Validate(refQuery.Interface()))
}

// readPathParams reads path parameters from req, copy the contents into `refPathParams` struct and applies
// validation using validator.
func readPathParams(req *http.Request, refPathParams *reflect.Value) error {

	if err := reflector.ReadStringMapIntoStruct(refPathParams, GetPathParams(req)); err != nil {
		return fmt.Errorf("failed reading path params: %w", err)
	}

	return WrapProblemDetailsError(validator.NewValidator().Validate(refPathParams.Interface()))
}
