package churro

import (
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"testing"
)

type RouteCases struct {
	Method string
	Path   string
}

func TestRouterRouteCreateBasic(t *testing.T) {
	r := NewRouter()

	cases := []RouteCases{
		{"GET", "/users"},
		{"GET", "/users/:id"},
		{"PATCH", "/users"},
		{"POST", "/users/:id"},
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {})

	r.Get("users", handler)
	r.Get("users/:id", handler)
	r.Patch("users", handler)
	r.Post("users/:id", handler)

	if len(r.routes) < 4 {
		t.Fatal("Expected exactly 4 routes, got ", len(r.routes))
	}

	for i, c := range cases {
		route := r.routes[i]
		if route.Method != HttpMethod(c.Method) {
			t.Errorf("%d: want method %s, got %s", i, c.Method, route.Method)
		}
		if route.Path != c.Path {
			t.Errorf("%d: want path %s, got %s", i, c.Path, route.Path)
		}
	}

}

func TestRouterRouteGroupsBasic(t *testing.T) {
	r := NewRouter()

	handler := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
	})

	r.Get("health", handler)

	r.Group(func(g1 *Router) {

		g1.Get("users", handler)
		g1.Get("users/:id", handler)
		g1.Patch("users", handler)
		g1.Post("users/:id", handler)

		g1.Group(func(g2 *Router) {
			g2.Get(":id/items", handler)
		}).Prefix("users")

	}).Prefix("v1")

	for _, route := range r.Routes() {
		slog.Info(fmt.Sprintf("Testing route %s -> %s", route.FullPath, route.Path))
	}
}

func TestRouterRouteMiddlewaresGroupsBasic(t *testing.T) {
	r := NewRouter()

	handler := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {})

	var executionOrder []string

	makeNamedMiddlewares := func(name string, executionOrder *[]string) Middleware {
		return func(next http.Handler) http.Handler {
			*executionOrder = append(*executionOrder, name)
			return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				next.ServeHTTP(w, req)
			})
		}
	}

	r.Middlewares(
		makeNamedMiddlewares("0", &executionOrder),
		makeNamedMiddlewares("1", &executionOrder),
	)

	if len(r.middleware) != 2 {
		t.Fatal("Expected exactly 2 routes, got ", len(r.middleware))
	}

	r.Group(func(g1 *Router) {

		route := g1.Get("orders", handler)

		route.Middlewares(
			makeNamedMiddlewares("4", &executionOrder),
			makeNamedMiddlewares("5", &executionOrder),
		)

		if len(route.middlewares) != 2 {
			t.Fatal("Expected exactly 2 routes, got ", len(r.middleware))
		}

	}).Middlewares(
		makeNamedMiddlewares("2", &executionOrder),
		makeNamedMiddlewares("3", &executionOrder),
	).Prefix("admin")

	route := r.Routes()[0]

	r.applyRouteMiddlewares(route)

	middlewares := r.GetRoutMiddlewares(route)

	if len(middlewares) != 6 {
		t.Fatalf("want 6 routes, got %d", len(middlewares))
	}

	if len(executionOrder) != 6 {
		t.Fatalf("want 6 executed middlewares, got %d", len(executionOrder))
	}

	for i, m := range executionOrder {
		if strconv.Itoa(i) != m {
			t.Errorf("%d: want %s, got %s", i, strconv.Itoa(i), m)
		}

	}

}
