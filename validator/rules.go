package validator

import (
	"fmt"
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
func RequiredRule(fieldVal reflect.Value, _ []string) (bool, error) {
	return fieldVal.IsValid() && ((fieldVal.Kind() == reflect.Ptr && !fieldVal.IsNil()) || !fieldVal.IsZero()), nil
}

// EmailRule Value under validation must be set and non-empty.
func EmailRule(fieldVal reflect.Value, _ []string) (bool, error) {
	_, err := mail.ParseAddress(fieldVal.String())
	return err == nil, nil
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

func MinRule(field reflect.Value, params []string) (bool, error) {

	if len(params) != 1 {
		return false, fmt.Errorf("min rule requires exactly one parameter")
	}

	switch field.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		minVal, err := strconv.ParseInt(params[0], 10, 64)
		if err != nil {
			return false, fmt.Errorf("min rule parameter '%s' must be a valid integer", params[0])
		}
		return field.Int() >= minVal, nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		minVal, err := strconv.ParseUint(params[0], 10, 64)
		if err != nil {
			return false, fmt.Errorf("min rule parameter '%s' must be a valid unsigned integer", params[0])
		}
		return field.Uint() >= minVal, nil
	case reflect.Float32, reflect.Float64:
		minVal, err := strconv.ParseFloat(params[0], 64)
		if err != nil {
			return false, fmt.Errorf("min rule parameter '%s' must be a valid number", params[0])
		}
		return field.Float() >= minVal, nil
	case reflect.String, reflect.Slice, reflect.Map, reflect.Array:
		minVal, err := strconv.Atoi(params[0])
		if err != nil {
			return false, fmt.Errorf("min rule parameter '%s' must be a valid integer for length validation", params[0])
		}
		return field.Len() >= minVal, nil
	default:
		return false, fmt.Errorf("min rule cannot be applied to field of type %s", field.Kind())
	}

}

func MaxRule(field reflect.Value, params []string) (bool, error) {

	if len(params) != 1 {
		return false, fmt.Errorf("max rule requires exactly one parameter")
	}

	switch field.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		maxVal, err := strconv.ParseInt(params[0], 10, 64)
		if err != nil {
			return false, fmt.Errorf("max rule parameter '%s' must be a valid integer", params[0])
		}
		return field.Int() <= maxVal, nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		maxVal, err := strconv.ParseUint(params[0], 10, 64)
		if err != nil {
			return false, fmt.Errorf("max rule parameter '%s' must be a valid unsigned integer", params[0])
		}
		return field.Uint() <= maxVal, nil
	case reflect.Float32, reflect.Float64:
		maxVal, err := strconv.ParseFloat(params[0], 64)
		if err != nil {
			return false, fmt.Errorf("max rule parameter '%s' must be a valid number", params[0])
		}
		return field.Float() <= maxVal, nil
	case reflect.String, reflect.Slice, reflect.Map, reflect.Array:
		maxVal, err := strconv.Atoi(params[0])
		if err != nil {
			return false, fmt.Errorf("max rule parameter '%s' must be a valid integer for length validation", params[0])
		}
		return field.Len() <= maxVal, nil
	default:
		return false, fmt.Errorf("max rule cannot be applied to field of type %s", field.Kind())
	}

}

// DateRule validates date in YYYY-MM-DD format (2006-01-02)
func DateRule(fieldVal reflect.Value, params []string) (bool, error) {
	if len(params) > 0 {
		return false, fmt.Errorf("date rule does not accept parameters, use date_format rule for custom formats")
	}
	
	layout := "2006-01-02"
	
	if _, err := time.Parse(layout, fieldVal.String()); err != nil {
		return false, fmt.Errorf("must be a valid date in YYYY-MM-DD format, got: %s", fieldVal.String())
	}
	return true, nil
}

// DateFormatRule validates date with custom format
func DateFormatRule(fieldVal reflect.Value, params []string) (bool, error) {
	if len(params) != 1 {
		return false, fmt.Errorf("date_format rule requires exactly one parameter (layout format)")
	}
	
	layout := params[0]
	
	if _, err := time.Parse(layout, fieldVal.String()); err != nil {
		return false, fmt.Errorf("must be a valid date in format %s, got: %s", layout, fieldVal.String())
	}
	return true, nil
}

// InArrayRule DateRule Value under validation must be set and non-empty.
func InArrayRule(field reflect.Value, params []string) (bool, error) {

	if len(params) == 0 {
		return false, fmt.Errorf("in rule requires at least one parameter")
	}

	return slices.Contains(params, field.String()), nil
}

func UUIDRule(field reflect.Value, params []string) (bool, error) {
	rfc4122UUID := `^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[1-5][0-9a-fA-F]{3}-[89aAbB][0-9a-fA-F]{3}-[0-9a-fA-F]{12}$`
	matches, _ := regexp.MatchString(rfc4122UUID, field.String())
	return matches, nil
}

// BooleanRule Value under validation must be a valid boolean.
func BooleanRule(field reflect.Value, _ []string) (bool, error) {
	return field.Kind() == reflect.Bool, nil
}

// NumberRule Value under validation must be numeric (for numeric types) or contain only digits (for strings).
func NumberRule(field reflect.Value, _ []string) (bool, error) {
	if field.Kind() == reflect.String {
		// For strings, check if all characters are digits
		str := field.String()
		if str == "" {
			return false, nil
		}
		for _, char := range str {
			if char < '0' || char > '9' {
				return false, nil
			}
		}
		return true, nil
	}
	// For other types, check if they are numeric
	return IsNumber(field, nil), nil
}

// ASCIIRule Value under validation must contain only ASCII characters.
func ASCIIRule(field reflect.Value, _ []string) (bool, error) {
	str := field.String()
	for _, char := range str {
		if char > unicode.MaxASCII {
			return false, nil
		}
	}
	return true, nil
}

// AlphaNumRule Value under validation must contain only alphanumeric characters.
func AlphaNumRule(field reflect.Value, _ []string) (bool, error) {
	str := field.String()
	for _, char := range str {
		if !unicode.IsLetter(char) && !unicode.IsNumber(char) {
			return false, nil
		}
	}
	return true, nil
}

// URLRule Value under validation must be a valid URL.
func URLRule(field reflect.Value, _ []string) (bool, error) {
	_, err := url.ParseRequestURI(field.String())
	return err == nil, nil
}

// HexColorRule Value under validation must be a valid hex color code.
func HexColorRule(field reflect.Value, _ []string) (bool, error) {
	hexColor := `^#([A-Fa-f0-9]{6}|[A-Fa-f0-9]{3})$`
	matches, _ := regexp.MatchString(hexColor, field.String())
	return matches, nil
}
