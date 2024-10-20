package churro

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/lambertmata/churro/validator"
	"io"
	"mime/multipart"
	"net/http"
	"reflect"
	"strconv"
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

	if errors.As(err, &fieldValidationError) {

		return &ProblemDetailsError{
			Status: http.StatusUnprocessableEntity,
			Title:  "Validation error",
			Detail: err.Error(),
			Err:    err,
		}

	} else if errors.As(err, &ruleParamsError) {

		return &ProblemDetailsError{
			Status: http.StatusUnprocessableEntity,
			Title:  "Validation error",
			Detail: err.Error(),
			Err:    err,
		}

	} else if errors.As(err, &unknownError) {

		return &ProblemDetailsError{
			Status: http.StatusInternalServerError,
			Title:  "Validation error",
			Detail: err.Error(),
			Err:    err,
		}

	}

	return &ProblemDetailsError{
		Status: http.StatusInternalServerError,
		Title:  "Validation error",
		Detail: "An unhandled validation error occurred",
		Err:    err,
	}

}

func ConvertNumericStringValIntoNumberOutputVal(refInputVal, refOutputVal reflect.Value) error {
	if refInputVal.Kind() != reflect.String {
		return nil
	}
	switch refOutputVal.Kind() {
	case reflect.Int:
		val, _ := strconv.Atoi(refInputVal.String())
		refOutputVal.SetInt(int64(val))
		break
	case reflect.Float64:
		val, _ := strconv.ParseFloat(refInputVal.String(), 64)
		refOutputVal.SetFloat(val)
		break
	}
	return nil
}

func ReadMapValuesIntoStruct(refStruct *reflect.Value, values map[string][]string) {

	refOutputPtr := refStruct.Elem()

	for i := 0; i < refOutputPtr.NumField(); i++ {

		outputField := refOutputPtr.Field(i)
		fieldName := strings.ToLower(refOutputPtr.Type().Field(i).Name)

		if !outputField.IsValid() {
			continue
		}

		inputVal, ok := values[fieldName]

		if !ok {
			continue
		}

		refInputVal := reflect.ValueOf(inputVal)

		inputIsSlice := refInputVal.Kind() == reflect.Slice || refInputVal.Kind() == reflect.Array
		outputIsSlice := outputField.Kind() == reflect.Slice || outputField.Kind() == reflect.Array

		toBeAssigned := refInputVal

		if !outputIsSlice && inputIsSlice && refInputVal.Len() > 0 {
			toBeAssigned = refInputVal.Index(0)
		}

		ConvertNumericStringValIntoNumberOutputVal(toBeAssigned, outputField)

		if !toBeAssigned.Type().AssignableTo(outputField.Type()) || !refOutputPtr.CanSet() {
			continue
		}

		refOutputPtr.Field(i).Set(toBeAssigned)
	}
}

// CreateStructFromMapValues populates an Output struct with values from the provided values map.
// It matches the struct's fields (case-insensitively) with the keys in the values map,
// filling only those fields that have corresponding keys.
// Any unmatched fields in the Output struct will be ignored.
// If a matching field in the Output struct is defined as a non-slice type,
// only the first element from the corresponding value will be copied.
func CreateStructFromMapValues[Output any](values map[string][]string) Output {

	var output Output

	refOutputPtr := reflect.ValueOf(&output).Elem()

	ReadMapValuesIntoStruct(&refOutputPtr, values)

	return output
}

func ReadValidatedBody(req *http.Request, bodyRef *reflect.Value) error {

	contentTypeParts := strings.Split(req.Header.Get("Content-Type"), ";")
	contentType := contentTypeParts[0]

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
				Detail: "Failed reading multipart form",
				Err:    err,
			}
		}

		res := CreateStructFromMapValues[any](req.MultipartForm.Value)

		for key, _ := range req.MultipartForm.File {

			file, _, err := req.FormFile(key)

			if err != nil {
				continue
			}

			FillStructFieldWithFile(&res, key, file)

		}

		if err := validator.NewValidator().Validate(res); err != nil {
			return WrapProblemDetailsError(err)
		}

		bodyRef.Elem().Set(reflect.ValueOf(res))

	default:
		return &ProblemDetailsError{
			Status: http.StatusBadRequest,
			Title:  "Bad Request",
			Detail: "Content type not supported: " + contentType,
		}
	}

	return nil
}

func isByteSlice(field reflect.Value) bool {
	return field.Kind() == reflect.Slice && field.Kind() == reflect.Uint8
}

func isIOReader(field reflect.Value) bool {
	return field.Type().Implements(reflect.TypeOf((*io.Reader)(nil)).Elem())
}

func FillStructFieldWithFile[Body any](body *Body, field string, file multipart.File) error {

	if body == nil {
		return errors.New("body is nil")
	}

	if file == nil {
		return errors.New("file is nil")
	}

	bodyRef := reflect.ValueOf(body).Elem()

	outputField := bodyRef.FieldByName(field)

	if !outputField.IsValid() {
		field = strings.ToUpper(string(field[0])) + field[1:]
		outputField = bodyRef.FieldByName(field)
	}

	if !outputField.IsValid() {
		return fmt.Errorf("field %s is not valid", field)
	}

	if !outputField.CanSet() {
		return fmt.Errorf("field %s can not be set", field)
	}

	if isIOReader(outputField) {

		outputField.Set(reflect.ValueOf(file))

	} else if isByteSlice(outputField) {

		bytes, err := io.ReadAll(file)

		if err != nil {
			return fmt.Errorf("could not read file for field %s: %w", field, err)
		}

		outputField.Set(reflect.ValueOf(bytes))
	}

	return nil
}

func ReadValidatedHeader(req *http.Request, refHeader *reflect.Value) error {

	header := refHeader.Interface()

	if header == nil {
		return nil
	}

	refHeaderType := refHeader.Type().Elem()

	if refHeaderType == nil || refHeaderType.Kind() != reflect.Struct {
		return nil
	}

	ReadMapValuesIntoStruct(refHeader, req.Header)

	return WrapProblemDetailsError(validator.NewValidator().Validate(header))
}

// ReadValidatedQuery reads query parameters from req, copy the contents into `refQuery` struct and applies
// validation using validator.
func ReadValidatedQuery(req *http.Request, refQuery *reflect.Value) error {

	if refQuery == nil || refQuery.Kind() != reflect.Struct {
		return nil
	}

	query := refQuery.Interface()

	if query == nil {
		return nil
	}

	ReadMapValuesIntoStruct(refQuery, req.URL.Query())

	return WrapProblemDetailsError(validator.NewValidator().Validate(refQuery.Interface()))
}

// ReadPathParamsIntoStruct reads path parameters from req, copy the contents into `refPathParams` struct and applies
// validation using validator.
func ReadPathParamsIntoStruct(req *http.Request, refPathParams *reflect.Value) error {

	if refPathParams == nil || refPathParams.Kind() != reflect.Struct {
		return nil
	}

	pathParams := refPathParams.Interface()

	if pathParams == nil {
		return nil
	}

	pathParamsToValues := make(map[string][]string)

	for key, val := range GetPathParams(req) {
		pathParamsToValues[key] = []string{val}
	}

	ReadMapValuesIntoStruct(refPathParams, pathParamsToValues)

	return WrapProblemDetailsError(validator.NewValidator().Validate(pathParams))
}
