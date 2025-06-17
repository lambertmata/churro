package reflector

import (
	"errors"
	"fmt"
	"io"
	"reflect"
	"strconv"
	"strings"
)

// InitStructPointerField Initializes a nil pointer field in a struct
func InitStructPointerField(refVal *reflect.Value, name string) {

	if refVal.Kind() == reflect.Ptr && refVal.Elem().Kind() != reflect.Struct {
		return
	}

	rField := refVal.FieldByName(name)

	// Skip if the field cannot be set or is not a Pointer
	if !rField.CanSet() || rField.Kind() != reflect.Pointer || !rField.IsNil() {
		return
	}

	// If the pointer is nil, initialize it
	rField.Set(reflect.New(rField.Type().Elem()))

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
	paramCtx := paramsInType.In(i)

	if paramCtx.Kind() == reflect.Pointer {
		paramCtx = paramCtx.Elem()
	}

	// Create new ctx
	return reflect.New(paramCtx)
}

func IsAny(refVal reflect.Value) bool {
	return refVal.Type() == reflect.TypeOf((*interface{})(nil))
}
