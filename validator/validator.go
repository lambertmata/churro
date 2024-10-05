package validator

import (
	"errors"
	"fmt"
	"github.com/lambertmata/churro/utils"
	"reflect"
	"strconv"
	"strings"
)

type FieldValidationError struct {
	Field string
	Rule  string
	Err   error
}

func (e *FieldValidationError) Error() string {
	return fmt.Sprintf(
		"failed %s rule validation on %s field",
		strings.ToLower(e.Rule),
		strings.ToLower(e.Field),
	)
}

type RuleParamsError struct {
	Rule      string
	RawParams string
	Field     string
	Err       error
}

func (e *RuleParamsError) Error() string {
	return fmt.Sprintf("malformed %s field rule %s params: %s", e.Field, e.Rule, e.RawParams)
}

type UnknownRuleError struct {
	Field string
	Rule  string
}

func (e *UnknownRuleError) Error() string {
	return fmt.Sprintf("invalid rule %s for %s", e.Rule, e.Field)
}

type ValidationFunc func(field reflect.Value, params []string) bool

// Validator validates
// required,required_with,required_if
type Validator struct {
	rules map[string]ValidationFunc
}

func (v *Validator) RegisterRule(name string, fn ValidationFunc) {
	v.rules[name] = fn
}

func (v *Validator) RegisterRuleWithPattern(name string, fn ValidationFunc, pattern string) error {
	v.rules[name] = fn
	return nil
}

func (v *Validator) registerDefaultRules() error {
	v.RegisterRule("required", RequiredRule)
	v.RegisterRule("min", MinRule)
	v.RegisterRule("max", MaxRule)
	v.RegisterRule("email", EmailRule)
	v.RegisterRule("date", DateRule)
	v.RegisterRule("in", InArrayRule)
	v.RegisterRule("uuid", UUIDRule)
	return nil
}

func NewValidator() *Validator {
	validator := &Validator{}
	validator.rules = make(map[string]ValidationFunc)
	validator.registerDefaultRules()
	return validator
}

// ExtractPartsFromValidationString returns from
func (v *Validator) ExtractPartsFromValidationString(str string) (name string, params []string, error error) {

	parts := utils.SplitString(str, ":")
	partsLen := len(parts)

	if partsLen < 1 {
		return "", nil, errors.New("validation string is empty")
	}

	name = parts[0]

	if partsLen == 2 {
		params = utils.SplitString(parts[1], ",")
	}

	return name, params, nil
}

func (v *Validator) ValidateWithRules(fieldName string, input reflect.Value, rawRules string) error {

	validationRules := utils.SplitString(rawRules, "|")

	for _, validationRule := range validationRules {

		name, params, err := v.ExtractPartsFromValidationString(validationRule)

		if err != nil {
			return &RuleParamsError{fieldName, validationRule, rawRules, err}
		}

		ruleFn, ok := v.rules[name]

		if !ok {
			return &UnknownRuleError{fieldName, validationRule}
		}

		if !ruleFn(input, params) {
			return &FieldValidationError{fieldName, validationRule, err}
		}
	}
	return nil
}

func (v *Validator) Validate(input any) error {
	return v.validate(input, "", "")
}

func (v *Validator) validate(input any, fieldName, rulesString string) error {

	reflectVal := reflect.ValueOf(input)

	if reflectVal.Kind() == reflect.Ptr {
		reflectVal = reflectVal.Elem()
	}

	switch reflectVal.Kind() {

	case reflect.Slice, reflect.Array:

		if err := v.ValidateWithRules(fieldName, reflectVal, rulesString); err != nil {
			return err
		}

		for i := 0; i < reflectVal.Len(); i++ {
			if err := v.validate(reflectVal.Index(i).Interface(), fieldName+"["+strconv.Itoa(i)+"]", rulesString); err != nil {
				return err
			}
		}

	case reflect.Struct:

		for i := 0; i < reflectVal.NumField(); i++ {
			data := reflectVal.Field(i)
			field := reflectVal.Type().Field(i)
			rules := field.Tag.Get("validate")
			curFieldName := field.Name

			if fieldName != "" {
				curFieldName = fieldName + "." + curFieldName
			}

			if len(rules) == 0 {
				continue
			}

			if err := v.validate(data.Interface(), curFieldName, rules); err != nil {
				return err
			}
		}

	default:
		if rulesString != "" {
			if err := v.ValidateWithRules(fieldName, reflectVal, rulesString); err != nil {
				return err
			}
		}
	}

	return nil

}
