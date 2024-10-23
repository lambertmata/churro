package reflector

import (
	"errors"
	"fmt"
	"io"
	"reflect"
	"strconv"
	"strings"
)

func InitFields(refVal *reflect.Value) {

	for i := 0; i < refVal.NumField(); i++ {

		rField := refVal.Field(i)

		if !rField.CanSet() {
			continue
		}

		if rField.Kind() == reflect.Pointer {

			rFieldVal := reflect.Zero(rField.Type().Elem())

			if rField.IsNil() {
				// Initialize the pointer field with a new value of the appropriate type
				rField.Set(reflect.New(rField.Type().Elem()))
			}

			if rField.CanSet() {
				//slog.Info("setting", "field", refVal.Type().Field(i).Name, "val", rFieldVal)
				rField.Elem().Set(rFieldVal)
			}

		} else {

			//	rField.Elem().Set(reflect.Zero(rField.Type()))
		}

	}
}

func ReadStringSlicesMapIntoStruct(refStruct *reflect.Value, values map[string][]string) error {

	if refStruct == nil || (refStruct.Kind() != reflect.Pointer || refStruct.Elem().Kind() != reflect.Struct) {
		return errors.New("ReadStringSlicesMapIntoStruct: refStruct must be a struct")
	}

	if refStruct.Interface() == nil {
		return errors.New("ReadStringSlicesMapIntoStruct: refStruct must have a non-nil value")
	}

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

	return nil
}

func ReadStringMapIntoStruct(refStruct *reflect.Value, values map[string]string) error {

	pathParamsToValues := make(map[string][]string)

	for key, val := range values {
		pathParamsToValues[key] = []string{val}
	}

	return ReadStringSlicesMapIntoStruct(refStruct, pathParamsToValues)
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

	ReadStringSlicesMapIntoStruct(&refOutputPtr, values)

	return output
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

func IsByteSlice(field reflect.Value) bool {
	return field.Kind() == reflect.Slice && field.Kind() == reflect.Uint8
}

func IsIOReader(field reflect.Value) bool {
	return field.Type().Implements(reflect.TypeOf((*io.Reader)(nil)).Elem())
}

func FillStructFieldWithReaderBytes(refStruct *reflect.Value, field string, reader io.Reader) error {

	if refStruct == nil {
		return errors.New("struct is nil")
	}

	if reader == nil {
		return errors.New("file is nil")
	}

	outputField := refStruct.Elem().FieldByName(field)

	if !outputField.IsValid() {
		field = strings.ToUpper(string(field[0])) + field[1:]
		outputField = refStruct.Elem().FieldByName(field)
	}

	if !outputField.IsValid() {
		return fmt.Errorf("field %s is not valid", field)
	}

	if !outputField.CanSet() {
		return fmt.Errorf("field %s can not be set", field)
	}

	if IsIOReader(outputField) {

		outputField.Set(reflect.ValueOf(reader))

	} else if IsByteSlice(outputField) {

		bytes, err := io.ReadAll(reader)

		if err != nil {
			return fmt.Errorf("could not read file for field %s: %w", field, err)
		}

		outputField.Set(reflect.ValueOf(bytes))
	}

	return nil
}

func NewValFromFuncParameter(refFunc reflect.Value, i int) reflect.Value {
	paramsInType := refFunc.Type()
	// Get first parameter which is the ctx
	paramCtx := paramsInType.In(i).Elem()
	// Create new ctx
	return reflect.New(paramCtx)
}
