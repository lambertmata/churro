package main

import (
	"fmt"
	"github.com/lambertmata/churro"
	"net/http"
	"net/http/httptest"
)

func TestMiddleware() churro.Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			fmt.Printf("MIDDLEWARE CALLED: %s %s\n", req.Method, req.URL.Path)
			next.ServeHTTP(w, req)
		})
	}
}

func main() {
	router := churro.NewRouter()

	// Add middleware
	router.Middlewares(TestMiddleware())

	// Add a simple route
	router.Get("/test", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("test response"))
	})

	// Test 404 case
	req404 := httptest.NewRequest("GET", "/nonexistent", nil)
	w404 := httptest.NewRecorder()

	fmt.Println("Testing 404 case:")
	router.ServeHTTP(w404, req404)
	fmt.Printf("Status: %d\n", w404.Code)
	fmt.Printf("Body: %s\n", w404.Body.String())

	// Test 405 case
	req405 := httptest.NewRequest("DELETE", "/test", nil)
	w405 := httptest.NewRecorder()

	fmt.Println("\nTesting 405 case:")
	router.ServeHTTP(w405, req405)
	fmt.Printf("Status: %d\n", w405.Code)
	fmt.Printf("Body: %s\n", w405.Body.String())

	// Test successful case
	reqOK := httptest.NewRequest("GET", "/test", nil)
	wOK := httptest.NewRecorder()

	fmt.Println("\nTesting successful case:")
	router.ServeHTTP(wOK, reqOK)
	fmt.Printf("Status: %d\n", wOK.Code)
	fmt.Printf("Body: %s\n", wOK.Body.String())
}
