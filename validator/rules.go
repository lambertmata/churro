package validator

import (
	"net/mail"
	"net/url"
	"reflect"
	"regexp"
	"slices"
	"strconv"
	"time"
	"unicode"
)

// RequiredRule Value under validation must be set and non-empty.
func RequiredRule(fieldVal reflect.Value, _ []string) bool {
	return fieldVal.IsValid() && ((fieldVal.Kind() == reflect.Ptr && !fieldVal.IsNil()) || !fieldVal.IsZero())
}

// EmailRule Value under validation must be set and non-empty.
func EmailRule(fieldVal reflect.Value, _ []string) bool {
	_, err := mail.ParseAddress(fieldVal.String())
	return err == nil
}

func IsPrimaryType(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Bool,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr,
		reflect.Float32, reflect.Float64,
		reflect.Complex64, reflect.Complex128,
		reflect.String:
		return true
	default:
		return false
	}
}

func IsAbsolute(fieldVal reflect.Value, _ []string) bool {
	switch fieldVal.Kind() {
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return true
	}
	return false
}

func IsInteger(fieldVal reflect.Value, _ []string) bool {
	switch fieldVal.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return true
	}
	return false
}

func IsFloat(fieldVal reflect.Value, params []string) bool {
	switch fieldVal.Kind() {
	case reflect.Float32, reflect.Float64:
		return true
	}
	return false
}

func IsNumber(fieldVal reflect.Value, params []string) bool {
	return IsInteger(fieldVal, params) || IsFloat(fieldVal, params)
}

func MinRule(field reflect.Value, params []string) bool {

	if len(params) != 1 {
		return false
	}

	minVal, err := strconv.Atoi(params[0])

	if err != nil {
		return false
	}

	switch field.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return field.Int() >= int64(minVal)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return field.Uint() >= uint64(minVal)
	case reflect.Float32, reflect.Float64:
		return field.Float() >= float64(minVal)
	case reflect.String, reflect.Slice, reflect.Map, reflect.Array:
		return field.Len() >= minVal
	default:
		return false
	}

}

func MaxRule(field reflect.Value, params []string) bool {

	if len(params) != 1 {
		return false
	}

	maxVal, err := strconv.Atoi(params[0])

	if err != nil {
		return false
	}

	switch field.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return field.Int() <= int64(maxVal)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return field.Uint() <= uint64(maxVal)
	case reflect.Float32, reflect.Float64:
		return field.Float() <= float64(maxVal)
	case reflect.String, reflect.Slice, reflect.Map, reflect.Array:
		return field.Len() <= maxVal
	default:
		return false
	}

}

// DateRule Value under validation must be set and non-empty.
func DateRule(fieldVal reflect.Value, params []string) bool {
	var layout string
	if len(params) == 1 {
		layout = params[0]
	} else {
		layout = "2006-01-02 00:00:00"
	}
	if _, err := time.Parse(layout, fieldVal.String()); err != nil {
		return false
	}
	return true
}

// InArrayRule DateRule Value under validation must be set and non-empty.
func InArrayRule(field reflect.Value, params []string) bool {

	if len(params) == 0 {
		return false
	}

	return slices.Contains(params, field.String())
}

func UUIDRule(field reflect.Value, params []string) bool {
	rfc4122UUID := `^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[1-5][0-9a-fA-F]{3}-[89aAbB][0-9a-fA-F]{3}-[0-9a-fA-F]{12}$`
	matches, _ := regexp.MatchString(rfc4122UUID, field.String())
	return matches
}

// BooleanRule Value under validation must be a valid boolean.
func BooleanRule(field reflect.Value, _ []string) bool {
	return field.Kind() == reflect.Bool
}

// NumberRule Value under validation must be a numeric type.
func NumberRule(field reflect.Value, _ []string) bool {
	return IsNumber(field, nil)
}

// ASCIIRule Value under validation must contain only ASCII characters.
func ASCIIRule(field reflect.Value, _ []string) bool {
	str := field.String()
	for _, char := range str {
		if char > unicode.MaxASCII {
			return false
		}
	}
	return true
}

// AlphaNumRule Value under validation must contain only alphanumeric characters.
func AlphaNumRule(field reflect.Value, _ []string) bool {
	str := field.String()
	for _, char := range str {
		if !unicode.IsLetter(char) && !unicode.IsNumber(char) {
			return false
		}
	}
	return true
}

// URLRule Value under validation must be a valid URL.
func URLRule(field reflect.Value, _ []string) bool {
	_, err := url.ParseRequestURI(field.String())
	return err == nil
}

// HexColorRule Value under validation must be a valid hex color code.
func HexColorRule(field reflect.Value, _ []string) bool {
	hexColor := `^#([A-Fa-f0-9]{6}|[A-Fa-f0-9]{3})$`
	matches, _ := regexp.MatchString(hexColor, field.String())
	return matches
}
