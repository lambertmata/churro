package churro

import (
	"net/http"
	"testing"
)

func TestCreateRouteBasic(t *testing.T) {
	m1 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r)
		})
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	route := NewRoute(Get, "", handler)
	if route == nil {
		t.Error("route is nil")
	}

	if route.Path != "/" {
		t.Errorf("route.Path is \"\"; want \\ ")
	}

	route.Middlewares(m1)

	if len(route.middlewares) != 1 {
		t.Fatalf("route.Middlewares is %d; want 1", len(route.middlewares))
	}

}
