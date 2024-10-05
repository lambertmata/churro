package middlewares

import (
	"context"
	"encoding/json"
	"net/http"
)

type Validator interface {
	Valid() *ValidationError
}

type ValidationError struct {
	Message string                 `json:"message"`
	Errors  map[string]interface{} `json:"errors"`
}

type ValidationContext struct{}

func ValidateJson[T Validator]() func(next http.Handler) http.Handler {

	return func(next http.Handler) http.Handler {

		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {

			var decodedBody T

			if bodyErr := json.NewDecoder(req.Body).Decode(&decodedBody); bodyErr != nil {
				http.Error(w, "Invalid JSON", http.StatusBadRequest)
				return
			}

			if validationError := decodedBody.Valid(); validationError != nil {

				w.WriteHeader(http.StatusUnprocessableEntity)

				if err := json.NewEncoder(w).Encode(validationError); err != nil {
					http.Error(w, "Failed serializing validation error", http.StatusInternalServerError)
				}

				return
			}

			ctx := context.WithValue(req.Context(), ValidationContext{}, decodedBody)

			next.ServeHTTP(w, req.WithContext(ctx))
		})
	}
}

func GetValidated[T any](r *http.Request) T {
	return r.Context().Value(ValidationContext{}).(T)
}
