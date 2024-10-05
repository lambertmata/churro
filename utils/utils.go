package utils

import (
	"reflect"
	"slices"
	"strings"
)

func SplitString(s string, del string) []string {
	return slices.DeleteFunc(strings.Split(s, del), func(part string) bool {
		return part == ""
	})
}

func copy(refIn reflect.Value, refOut reflect.Value) {

	if refOut.Kind() != reflect.Ptr || refOut.IsNil() {
		return
	}

	refOut = refOut.Elem()

	for i := 0; i < refIn.NumField(); i++ {
		fieldIn := refIn.Field(i)
		fieldName := refIn.Type().Field(i).Name
		fieldOut := refOut.FieldByName(fieldName)

		if !fieldOut.IsValid() || !fieldOut.CanSet() {
			continue
		}

		if fieldIn.Kind() == reflect.Struct && fieldOut.Kind() == reflect.Struct {
			copy(fieldIn, fieldOut.Addr())
			continue
		}

		if fieldIn.Kind() == reflect.Ptr && fieldOut.Kind() == reflect.Ptr {
			if fieldIn.IsNil() {
				fieldOut.Set(reflect.Zero(fieldOut.Type()))
			} else {

				if fieldOut.IsNil() {
					fieldOut.Set(reflect.New(fieldIn.Type().Elem()))
				}
				copy(fieldIn.Elem(), fieldOut.Elem())
			}
			continue
		}

		fieldOut.Set(fieldIn)
	}
}

func As[In any, Out any](in In, out Out) *Out {

	refIn := reflect.ValueOf(in)
	refOut := reflect.ValueOf(&out)

	copy(refIn, refOut)

	return &out
}
