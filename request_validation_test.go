package churro

import (
	"errors"
	"testing"
)

func TestCreateStructFromMapValuesBasic(t *testing.T) {

	type User struct {
		Name        string
		Roles       []string
		Permissions []string
		Email       string
	}

	userMap := map[string][]string{
		"name":        {"bob"},
		"roles":       {"power", "viewer"},
		"permissions": {},
		"email":       {"bob@example.com"},
	}

	userStruct := CreateStructFromMapValues[User](userMap)

	if userStruct.Name != "bob" {
		t.Fatalf("Name should be 'bob' but was %s", userStruct.Name)
	}

	if len(userStruct.Roles) != 2 {
		t.Fatalf("Roles should have 2 items but has %d", len(userStruct.Roles))
	}

	if len(userStruct.Permissions) != 0 {
		t.Fatalf("Permissions should be empty but was %v", userStruct.Permissions)
	}

	if userStruct.Email != "bob@example.com" {
		t.Fatalf("Email should be 'bob@example.com' but was %s", userStruct.Email)
	}

}

func TestCreateStructFromStructSliceToSoleValue(t *testing.T) {

	type User struct {
		Name       string
		Roles      string
		Permission string
		Email      string
	}

	userMap := map[string][]string{
		"name":       {"bob"},
		"roles":      {"power", "viewer"},
		"permission": {"view"},
		"email":      {"bob@example.com"},
	}

	userStruct := CreateStructFromMapValues[User](userMap)

	if userStruct.Name != "bob" {
		t.Fatalf("Name should be 'bob' but was %s", userStruct.Name)
	}

	if userStruct.Roles != "power" {
		t.Fatalf("Roles should be 'power' but was %v", userStruct.Roles)
	}

	if userStruct.Permission != "view" {
		t.Fatalf("Permission should be 'view' but was %v", userStruct.Permission)
	}

	if userStruct.Email != "bob@example.com" {
		t.Fatalf("Email should be 'bob@example.com' but was %s", userStruct.Email)
	}
}

func TestWrapProblemDetailsError(t *testing.T) {

	err := WrapProblemDetailsError(nil)

	if err != nil {
		t.Fatalf("Error should be nil but was %v", err)
	}

	err = WrapProblemDetailsError(errors.New("test error"))

	if err == nil {
		t.Fatalf("Error should not be nil but was %v", err)
	}

	var problemDetailsErr *ProblemDetailsError

	if !errors.As(err, &problemDetailsErr) {
		t.Fatalf("Error should be ProblemDetailsError but was %v", err)
	} else {
		problemDetailsErr = nil
	}

}
