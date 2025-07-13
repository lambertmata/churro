package main

import (
	"bytes"
	"fmt"
	"net/http/httptest"

	churro "github.com/lambertmata/churro"
)

type CreateUserRequest struct {
	Name  string `json:"name" validate:"required"`
	Email string `json:"email" validate:"required,email"`
	Age   int    `json:"age" validate:"min=18"`
}

func main() {
	router := churro.NewRouter()

	// Test route with validation
	churro.Post(router, "/users", func(ctx *churro.ContextWithBody[CreateUserRequest]) error {
		return ctx.SendJSON(map[string]string{"message": "User created successfully"})
	})

	// Test invalid JSON
	invalidJSON := `{"name": "John", "email": "invalid-email", "age": 15`
	req := httptest.NewRequest("POST", "/users", bytes.NewReader([]byte(invalidJSON)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	fmt.Println("Testing invalid JSON:")
	router.ServeHTTP(w, req)
	fmt.Printf("Status: %d\n", w.Code)
	fmt.Printf("Body: %s\n", w.Body.String())

	// Test validation errors
	validJSON := `{"name": "", "email": "invalid-email", "age": 15}`
	req2 := httptest.NewRequest("POST", "/users", bytes.NewReader([]byte(validJSON)))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()

	fmt.Println("\nTesting validation errors:")
	router.ServeHTTP(w2, req2)
	fmt.Printf("Status: %d\n", w2.Code)
	fmt.Printf("Body: %s\n", w2.Body.String())

	// Test 404
	req3 := httptest.NewRequest("GET", "/nonexistent", nil)
	w3 := httptest.NewRecorder()

	fmt.Println("\nTesting 404:")
	router.ServeHTTP(w3, req3)
	fmt.Printf("Status: %d\n", w3.Code)
	fmt.Printf("Body: %s\n", w3.Body.String())

	// Test method not allowed
	req4 := httptest.NewRequest("DELETE", "/users", nil)
	w4 := httptest.NewRecorder()

	fmt.Println("\nTesting 405:")
	router.ServeHTTP(w4, req4)
	fmt.Printf("Status: %d\n", w4.Code)
	fmt.Printf("Body: %s\n", w4.Body.String())
}
