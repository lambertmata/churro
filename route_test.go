package churro

import (
	"net/http"
	"testing"
)

func TestNewRoutePrefix(t *testing.T) {
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	table := []struct {
		path       string
		wantedPath string
	}{
		{"/", "/"},
		{"api", "/api"},
		{"/api/v1", "/api/v1"},
		{"//api/v2", "/api/v2"},
		{"", "/"},
	}

	for _, row := range table {
		route := NewRoute(MethodGet, row.path, h)
		if route.fullPath != row.wantedPath {
			t.Errorf("wanted %s, got %s", row.wantedPath, route.fullPath)
		}
	}

}

func TestRouteHttpMethods(t *testing.T) {
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	table := []struct {
		method       HttpMethod
		wantedMethod string
	}{
		{MethodGet, "GET"},
		{MethodPost, "POST"},
		{MethodPut, "PUT"},
		{MethodPatch, "PATCH"},
		{MethodHead, "HEAD"},
		{MethodOption, "OPTIONS"},
		{MethodConnect, "CONNECT"},
		{MethodTrace, "TRACE"},
	}

	for _, row := range table {
		route := NewRoute(row.method, "/", h)
		if string(route.Method) != row.wantedMethod {
			t.Errorf("wanted %s, got %s", row.wantedMethod, route.Method)
		}
	}
}

func TestDefineParamMatchers(t *testing.T) {
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	route := NewRoute(MethodGet, "/users/:id", h)

	if route.Matchers != nil {
		t.Errorf("New route initial route.Matchers = %v, want nil", route.Matchers)
	}

	route.Matches(":id", `\w+`)

	if route.Matchers == nil {
		t.Errorf("New route initial route.Matchers = %v, want not nil", route.Matchers)
	}

}
