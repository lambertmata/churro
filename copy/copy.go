package copy

import (
	"errors"
	"reflect"
)

func As(in, out any) error {

	refIn := reflect.ValueOf(in)
	refOut := reflect.ValueOf(out)

	if refIn.Kind() == reflect.Ptr {
		refIn = refIn.Elem()
	}

	if refOut.Kind() != reflect.Ptr || refOut.IsNil() {
		return errors.New("out must be a non-nil pointer")
	}

	copy(refIn, refOut)

	return nil
}

func copy(inVal, outVal reflect.Value) {
	// Ensure destination is a pointer and not nil
	if outVal.Kind() != reflect.Ptr || outVal.IsNil() {
		return
	}

	// Get the actual value for destination pointer
	outVal = reflect.Indirect(outVal)

	if inVal.Kind() == reflect.Slice || inVal.Kind() == reflect.Array {
		copySlice(inVal, outVal)
		return
	}

	if inVal.Kind() == reflect.Struct {
		copyStructFields(inVal, outVal)
	}
}

func copySlice(inVal, outVal reflect.Value) {
	if outVal.Kind() != reflect.Slice && outVal.Kind() != reflect.Array || outVal.Kind() != inVal.Kind() {
		return
	}

	newSlice := reflect.MakeSlice(outVal.Type(), inVal.Len(), inVal.Cap())

	for i := 0; i < inVal.Len(); i++ {
		inElem := inVal.Index(i)
		outElem := newSlice.Index(i)

		if inElem.Kind() == reflect.Ptr && inElem.IsNil() {
			continue
		}

		if inElem.Type() == outElem.Type() {
			outElem.Set(inElem)
			continue
		}

		if outElem.Kind() == reflect.Ptr {
			newElem := reflect.New(outElem.Type().Elem())
			outElem = newElem.Elem()
			outElem.Set(newElem)
		}

		copy(inElem, outElem.Addr())

	}
	outVal.Set(newSlice)
}

func copyStructFields(inVal, outVal reflect.Value) {
	outType := outVal.Type()

	// Iterate through fields in the destination struct
	for i := 0; i < outType.NumField(); i++ {
		outField := outVal.Field(i)
		outFieldType := outType.Field(i)

		// Find corresponding field in source struct
		inFieldName := outFieldType.Name
		inField := inVal.FieldByName(inFieldName)

		if !inField.IsValid() || !outField.CanSet() {
			continue
		}

		// Handle different field types
		switch outField.Kind() {
		case reflect.Ptr:
			copyPointerField(inField, outField)
		case reflect.Array:
			fallthrough
		case reflect.Slice:
			copySlice(inField, outField)
		case reflect.Struct:
			if inField.Type() == outField.Type() {
				outField.Set(inField)
			} else {
				// Handle nested struct with different types
				newStruct := reflect.New(outField.Type())
				copy(inField, newStruct)
				outField.Set(newStruct.Elem())
			}
		default:
			// Handle basic types
			if inField.Type() == outField.Type() {
				outField.Set(inField)
			}
		}
	}
}

func copyPointerField(inVal, outVal reflect.Value) {
	if !inVal.IsValid() || !outVal.CanSet() {
		return
	}

	// Handle nil source pointer
	if inVal.Kind() == reflect.Ptr && inVal.IsNil() {
		outVal.Set(reflect.Zero(outVal.Type()))
		return
	}

	// Create new pointer if needed
	if outVal.IsNil() {
		outVal.Set(reflect.New(outVal.Type().Elem()))
	}

	// Copy the pointed-to values
	if inVal.Kind() == reflect.Ptr {
		inElem := inVal.Elem()
		outElem := outVal.Elem()

		if inElem.Type() == outElem.Type() {
			outElem.Set(inElem)
		} else {
			copy(inVal, outVal.Addr())
		}
	} else {
		outElem := outVal.Elem()
		if inVal.Type() == outElem.Type() {
			outElem.Set(inVal)
		}
	}
}
