package reflector

import (
	"reflect"
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
