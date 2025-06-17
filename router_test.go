package churro

import (
	"errors"
	"net/http"
	"net/http/httptest"
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
		{"GET", "/users/{id}"},
		{"PATCH", "/users"},
		{"POST", "/users/{id}"},
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {})

	r.Get("users", handler)
	r.Get("users/{id}", handler)
	r.Patch("users", handler)
	r.Post("users/{id}", handler)

	if len(r.routes) < 4 {
		t.Fatal("Expected exactly 4 routes, got ", len(r.routes))
	}

	for i, c := range cases {
		route := r.routes[i]
		if route.Method != HttpMethod(c.Method) {
			t.Errorf("%d: want method %s, got %s", i, c.Method, route.Method)
		}
		if route.path != c.Path {
			t.Errorf("%d: want path %s, got %s", i, c.Path, route.path)
		}
	}

}

func TestRouterRouteGroupsBasic(t *testing.T) {
	r := NewRouter()

	handler := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
	})

	r.Get("health", handler)

	r.Group(func(g1 *Router) {

		g1.Option("users", handler)
		g1.Get("users", handler)
		g1.Delete("users/{id}", handler)
		g1.Patch("users", handler)
		g1.Post("users/{id}", handler)

		g1.Group(func(g2 *Router) {
			g2.Get("{id}/items", handler)
			g2.Get("/", handler)
		}).Prefix("users")

	}).Prefix("v1")

	r.Head("v1/users/{id}", handler)

	if len(r.Routes()) < 9 {
		t.Fatal("Expected at least 8 routes, got ", len(r.Routes()))
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

	r.Get("health", handler)

	r.Group(func(g1 *Router) {

		route := g1.Get("orders", handler)

		route.Middlewares(
			makeNamedMiddlewares("4", &executionOrder),
			makeNamedMiddlewares("5", &executionOrder),
		)

		g1.Post("orders", handler)
		g1.Get("/", handler)

	}).Middlewares(
		makeNamedMiddlewares("2", &executionOrder),
		makeNamedMiddlewares("3", &executionOrder),
	).Prefix("admin")

}

func TestRouterMatchers(t *testing.T) {
	r := NewRouter()

	handler := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {})

	r.Get("users", handler)
	r.Get("users/{id}", handler)
	r.Patch("users", handler)
	r.Post("users/{id}", handler)
	r.Get("users/{any}", handler).Matches("any", ".*")

}

func TestGrouped(t *testing.T) {
	r := NewRouter()

	handler := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {})

	r.Get("/api", handler)
	r.Get("/", handler)
	r.Get("ws/", handler)

	r.Group(func(gRouter *Router) {
		gRouter.Get("/", handler)
	}).Prefix("/ws")

	r.Get("/ws/channels/{channel}", handler)

	route, _ := r.mux.Match(MethodGet, "/")
	if route == nil {
		t.Fatal("Expected route to exist")
	}

	route, _ = r.mux.Match(MethodGet, "/ws/channels/1")

	if route == nil {
		t.Fatal("Expected route to exist")
	}

	route, err := r.mux.Match(MethodPost, "/ws/private-channels")

	if err == nil || !errors.Is(err, ErrNodeNotFound) {
		t.Fatalf("Expected ErrRoutedMethodNotImplemented, got %v", err)
	}

	route, err = r.mux.Match(MethodPost, "/ws/private-channels")

	if err == nil || !errors.Is(err, ErrNodeNotFound) {
		t.Fatalf("Expected ErrNodeRouteUndefined, got %v", err)
	}
}

func TestReadPathParams(t *testing.T) {
	r := NewRouter()

	r.Group(func(g *Router) {

		g.Get("/users/{id}", func(w http.ResponseWriter, req *http.Request) {
			id := GetPathParam(req, "id")
			if id != "1" {
				t.Errorf("Expected id to b 1, got none")
			}
		})

		g.Get("/users/{user-id}/orders/{order-id}", func(w http.ResponseWriter, req *http.Request) {
			if GetPathParam(req, "user-id") != "1" {
				t.Errorf("Expected id to b 1, got none")
			}
			if GetPathParam(req, "order-id") != "100" {
				t.Errorf("Expected id to b 100, got none")
			}
		})

	}).Prefix("/api")

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/users/1", nil)
	r.ServeHTTP(w, req)

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/users/1/orders/100", nil)
	r.ServeHTTP(w, req)
}
