package validator

import (
	"reflect"
	"strings"
	"testing"
)

func TestNewValidator(t *testing.T) {
	v := NewValidator()
	if v == nil {
		t.Fatal("NewValidator() returned nil")
	}
	if v.rules == nil {
		t.Fatal("NewValidator() did not initialize rules map")
	}

	// Check that default rules are registered
	expectedRules := []string{"required", "min", "max", "email", "date", "in", "uuid"}
	for _, rule := range expectedRules {
		if _, exists := v.rules[rule]; !exists {
			t.Errorf("Default rule '%s' not registered", rule)
		}
	}
}

func TestRequiredRule(t *testing.T) {
	tests := []struct {
		name     string
		value    interface{}
		expected bool
	}{
		{"non-empty string", "hello", true},
		{"empty string", "", false},
		{"non-zero int", 42, true},
		{"zero int", 0, false},
		{"non-nil pointer", func() *string { s := "test"; return &s }(), true},
		{"nil pointer", (*string)(nil), false},
		{"non-empty slice", []string{"a"}, true},
		{"empty slice", []string{}, true},
		{"true bool", true, true},
		{"false bool", false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val := reflect.ValueOf(tt.value)
			result := RequiredRule(val, nil)
			if result != tt.expected {
				t.Errorf("RequiredRule() = %v, expected %v for value %v", result, tt.expected, tt.value)
			}
		})
	}
}

func TestEmailRule(t *testing.T) {
	tests := []struct {
		name     string
		email    string
		expected bool
	}{
		{"valid email", "test@example.com", true},
		{"valid email with subdomain", "user@mail.example.com", true},
		{"invalid email: no @", "testexample.com", false},
		{"invalid email: no domain", "test@", false},
		{"invalid email: no local part", "@example.com", false},
		{"empty string", "", false},
		{"just @", "@", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val := reflect.ValueOf(tt.email)
			result := EmailRule(val, nil)
			if result != tt.expected {
				t.Errorf("EmailRule() = %v, expected %v for email %s", result, tt.expected, tt.email)
			}
		})
	}
}

func TestMinRule(t *testing.T) {
	tests := []struct {
		name     string
		value    interface{}
		params   []string
		expected bool
	}{
		// Integer tests
		{"int above min", 10, []string{"5"}, true},
		{"int equal to min", 5, []string{"5"}, true},
		{"int below min", 3, []string{"5"}, false},

		// String length tests
		{"string above min length", "hello", []string{"3"}, true},
		{"string equal to min length", "hi", []string{"2"}, true},
		{"string below min length", "a", []string{"3"}, false},

		// Slice length tests
		{"slice above min length", []string{"a", "b", "c"}, []string{"2"}, true},
		{"slice equal to min length", []string{"a", "b"}, []string{"2"}, true},
		{"slice below min length", []string{"a"}, []string{"2"}, false},

		// Float tests
		{"float above min", 10.5, []string{"5"}, true},
		{"float below min", 3.2, []string{"5"}, false},

		// Error cases
		{"invalid params: empty", 10, []string{}, false},
		{"invalid params: non-numeric", 10, []string{"abc"}, false},
		{"invalid params: multiple", 10, []string{"5", "10"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val := reflect.ValueOf(tt.value)
			result := MinRule(val, tt.params)
			if result != tt.expected {
				t.Errorf("MinRule() = %v, expected %v for value %v with params %v", result, tt.expected, tt.value, tt.params)
			}
		})
	}
}

func TestMaxRule(t *testing.T) {
	tests := []struct {
		name     string
		value    interface{}
		params   []string
		expected bool
	}{
		// Integer tests
		{"int below max", 3, []string{"5"}, true},
		{"int equal to max", 5, []string{"5"}, true},
		{"int above max", 10, []string{"5"}, false},

		// String length tests
		{"string below max length", "hi", []string{"5"}, true},
		{"string equal to max length", "hello", []string{"5"}, true},
		{"string above max length", "hello world", []string{"5"}, false},

		// Float tests
		{"float below max", 3.2, []string{"5"}, true},
		{"float above max", 10.5, []string{"5"}, false},

		// Error cases
		{"invalid params: empty", 10, []string{}, false},
		{"invalid params: non-numeric", 10, []string{"abc"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val := reflect.ValueOf(tt.value)
			result := MaxRule(val, tt.params)
			if result != tt.expected {
				t.Errorf("MaxRule() = %v, expected %v for value %v with params %v", result, tt.expected, tt.value, tt.params)
			}
		})
	}
}

func TestDateRule(t *testing.T) {
	tests := []struct {
		name     string
		date     string
		params   []string
		expected bool
	}{
		{"valid date default format", "2023-01-15 00:00:00", []string{}, true},
		{"valid date custom format", "15/01/2023", []string{"02/01/2006"}, true},
		{"invalid date default format", "2023-13-45", []string{}, false},
		{"invalid date custom format", "2023/01/15", []string{"02/01/2006"}, false},
		{"empty date", "", []string{}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val := reflect.ValueOf(tt.date)
			result := DateRule(val, tt.params)
			if result != tt.expected {
				t.Errorf("DateRule() = %v, expected %v for date %s with params %v", result, tt.expected, tt.date, tt.params)
			}
		})
	}
}

func TestInArrayRule(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		params   []string
		expected bool
	}{
		{"value in array", "apple", []string{"apple", "banana", "orange"}, true},
		{"value not in array", "grape", []string{"apple", "banana", "orange"}, false},
		{"empty array", "apple", []string{}, false},
		{"empty value in array", "", []string{"", "apple"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val := reflect.ValueOf(tt.value)
			result := InArrayRule(val, tt.params)
			if result != tt.expected {
				t.Errorf("InArrayRule() = %v, expected %v for value %s with params %v", result, tt.expected, tt.value, tt.params)
			}
		})
	}
}

func TestUUIDRule(t *testing.T) {
	tests := []struct {
		name     string
		uuid     string
		expected bool
	}{
		{"valid UUID v4", "550e8400-e29b-41d4-a716-446655440000", true},
		{"valid UUID v1", "6ba7b810-9dad-11d1-80b4-00c04fd430c8", true},
		{"invalid UUID: wrong format", "550e8400-e29b-41d4-a716", false},
		{"invalid UUID: wrong characters", "550e8400-e29b-41d4-a716-44665544000g", false},
		{"empty string", "", false},
		{"not a UUID", "hello-world", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val := reflect.ValueOf(tt.uuid)
			result := UUIDRule(val, nil)
			if result != tt.expected {
				t.Errorf("UUIDRule() = %v, expected %v for UUID %s", result, tt.expected, tt.uuid)
			}
		})
	}
}

func TestExtractPartsFromValidationString(t *testing.T) {
	v := NewValidator()

	tests := []struct {
		name           string
		validationStr  string
		expectedName   string
		expectedParams []string
		expectError    bool
	}{
		{"rule without params", "required", "required", nil, false},
		{"rule with single param", "min:5", "min", []string{"5"}, false},
		{"rule with multiple params", "in:apple,banana,orange", "in", []string{"apple", "banana", "orange"}, false},
		{"empty string", "", "", nil, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			name, params, err := v.ExtractPartsFromValidationString(tt.validationStr)

			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none")
				return
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if name != tt.expectedName {
				t.Errorf("Expected name %s, got %s", tt.expectedName, name)
			}

			if len(params) != len(tt.expectedParams) {
				t.Errorf("Expected %d params, got %d", len(tt.expectedParams), len(params))
				return
			}

			for i, param := range params {
				if param != tt.expectedParams[i] {
					t.Errorf("Expected param[%d] = %s, got %s", i, tt.expectedParams[i], param)
				}
			}
		})
	}
}

func TestValidateStruct(t *testing.T) {
	type User struct {
		Name  string `validate:"required|min:2|max:50"`
		Email string `validate:"required|email"`
		Age   int    `validate:"min:18|max:120"`
	}

	tests := []struct {
		name        string
		user        User
		expectError bool
	}{
		{
			name: "valid user",
			user: User{
				Name:  "John Doe",
				Email: "john@example.com",
				Age:   25,
			},
			expectError: false,
		},
		{
			name: "invalid email",
			user: User{
				Name:  "John Doe",
				Email: "invalid-email",
				Age:   25,
			},
			expectError: true,
		},
		{
			name: "name too short",
			user: User{
				Name:  "J",
				Email: "john@example.com",
				Age:   25,
			},
			expectError: true,
		},
		{
			name: "age too young",
			user: User{
				Name:  "John Doe",
				Email: "john@example.com",
				Age:   16,
			},
			expectError: true,
		},
	}

	v := NewValidator()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.Validate(tt.user)
			if tt.expectError && err == nil {
				t.Errorf("Expected validation error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected validation error: %v", err)
			}
		})
	}
}

func TestValidateNestedStruct(t *testing.T) {
	type Address struct {
		Street string `validate:"required"`
		City   string `validate:"required"`
	}

	type User struct {
		Name    string  `validate:"required"`
		Address Address `validate:""`
	}

	tests := []struct {
		name        string
		user        User
		expectError bool
	}{
		{
			name: "valid nested struct",
			user: User{
				Name: "John",
				Address: Address{
					Street: "123 Main St",
					City:   "Anytown",
				},
			},
			expectError: false,
		},
		{
			name: "invalid nested struct",
			user: User{
				Name: "John",
				Address: Address{
					Street: "", // Missing required field
					City:   "Anytown",
				},
			},
			expectError: true,
		},
	}

	v := NewValidator()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.Validate(tt.user)
			if tt.expectError && err == nil {
				t.Errorf("Expected validation error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected validation error: %v", err)
			}
		})
	}
}

func TestErrorTypes(t *testing.T) {
	t.Run("FieldValidationError", func(t *testing.T) {
		err := &FieldValidationError{
			Field: "Name",
			Rule:  "Required",
		}
		expected := "failed required rule validation on name field"
		if err.Error() != expected {
			t.Errorf("Expected error message '%s', got '%s'", expected, err.Error())
		}
	})

	t.Run("RuleParamsError", func(t *testing.T) {
		err := &RuleParamsError{
			Rule:      "min",
			RawParams: "abc",
			Field:     "Age",
		}
		expected := "malformed Age field rule min params: abc"
		if err.Error() != expected {
			t.Errorf("Expected error message '%s', got '%s'", expected, err.Error())
		}
	})

	t.Run("UnknownRuleError", func(t *testing.T) {
		err := &UnknownRuleError{
			Field: "Name",
			Rule:  "unknown",
		}
		expected := "invalid rule unknown for Name"
		if err.Error() != expected {
			t.Errorf("Expected error message '%s', got '%s'", expected, err.Error())
		}
	})
}

func TestValidateStructWithJSONTags(t *testing.T) {
	type TestStruct struct {
		ItemCount int    `json:"item_count" validate:"required|min:1"`
		UserName  string `json:"user_name" validate:"required"`
		Age       int    `validate:"required|min:18"` // No JSON tag
	}

	tests := []struct {
		name          string
		data          TestStruct
		expectError   bool
		expectedField string // Expected field name in error message
	}{
		{
			name: "invalid ItemCount",
			data: TestStruct{
				ItemCount: 0,
				UserName:  "john",
				Age:       20,
			},
			expectError:   true,
			expectedField: "item_count",
		},
		{
			name: "invalid UserName",
			data: TestStruct{
				ItemCount: 5,
				UserName:  "",
				Age:       20,
			},
			expectError:   true,
			expectedField: "user_name",
		},
		{
			name: "invalid Age",
			data: TestStruct{
				ItemCount: 5,
				UserName:  "john",
				Age:       15,
			},
			expectError:   true,
			expectedField: "age",
		},
		{
			name: "valid struct",
			data: TestStruct{
				ItemCount: 5,
				UserName:  "john",
				Age:       20,
			},
			expectError: false,
		},
	}

	v := NewValidator()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.Validate(tt.data)
			if tt.expectError && err == nil {
				t.Errorf("Expected validation error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected validation error: %v", err)
			}
			if tt.expectError && err != nil {
				errorMsg := err.Error()
				if !strings.Contains(errorMsg, tt.expectedField) {
					t.Errorf("Expected error message to contain '%s', got '%s'", tt.expectedField, errorMsg)
				}
			}
		})
	}
}

func BenchmarkValidateStruct(b *testing.B) {
	type User struct {
		Name  string `validate:"required|min:2|max:50"`
		Email string `validate:"required|email"`
		Age   int    `validate:"min:18|max:120"`
	}

	user := User{
		Name:  "John Doe",
		Email: "john@example.com",
		Age:   25,
	}

	v := NewValidator()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		v.Validate(user)
	}
}
