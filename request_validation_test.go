package churro

import (
	"errors"
	"testing"
)

func TestWrapProblemDetailsError(t *testing.T) {

	err := wrapProblemDetailsError(nil)

	if err != nil {
		t.Fatalf("Error should be nil but was %v", err)
	}

	err = wrapProblemDetailsError(errors.New("test error"))

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
