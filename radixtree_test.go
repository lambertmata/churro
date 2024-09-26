package churro

import (
	"errors"
	"net/http"
	"testing"
)

func TestNewRadixTree(t *testing.T) {

	tree := NewRadixTree()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello World!"))
	})

	routes := []*Route{
		NewRoute("GET", "/", handler),
		NewRoute("GET", "api/users", handler),
		NewRoute("GET", "api/users/:id", handler),
		NewRoute("DELETE", "api/users/:id", handler),
		NewRoute("GET", "api/users/:id/items", handler),
		NewRoute("GET", "api/items", handler),
		NewRoute("POST", "api/items", handler),
		NewRoute("GET", "api/invoices/:any", handler).Matches(":any", `\w+`),
	}

	for _, route := range routes {
		tree.AddRoute(route.RouteHandler())
	}

	inserted := tree.Routes()

	if len(inserted) != len(routes) {
		t.Errorf("Inserted wrong number of nodes: got %d, want %d", len(inserted), len(routes))
	}

	table := []struct {
		path        string
		method      HttpMethod
		expectedErr error
	}{
		{"api/users/1", MethodGet, nil},
		{"/api/users/1", MethodGet, nil},
		{"/ws/channel/1", MethodGet, ErrNodeNotFound},
		{"ws/channel/1", MethodGet, ErrNodeNotFound},
		{"api/users/1/items", MethodGet, nil},
		{"/api/users/1/items", MethodGet, nil},
		{"api/invoices/111", MethodGet, nil},
		{"/api/invoices/111", MethodGet, nil},
		{"api/invoices/abc", MethodGet, nil},
		{"/api/invoices/abc", MethodGet, nil},
		{"/api/invoices/abc?active=true", MethodGet, nil},
		{"api/invoices", MethodGet, ErrNodeNotFound},
		{"api/users", MethodPost, ErrRoutedMethodNotImplemented},
		{"/api/users", MethodPost, ErrRoutedMethodNotImplemented},
		{"/", MethodGet, nil},
		{"/", MethodPost, ErrRoutedMethodNotImplemented},
	}

	for _, row := range table {
		_, err := tree.SearchPath(row.path, row.method)
		if !errors.Is(err, row.expectedErr) {
			t.Errorf("Wrong search result: got %v, want %v", err, row.expectedErr)
		}
	}

	tree.RemoveRoute(routes[0].Method, routes[0].fullPath)

	inserted = tree.Routes()

	if len(tree.Routes()) != len(routes)-1 {
		t.Errorf("Wrong number of nodes after removal: got %d, want %d", len(routes)-1, len(tree.Routes()))
	}

	if _, err := tree.SearchPath("api/users", "GET"); err != nil {
		t.Errorf("Search error, wanted: %v got: %v", ErrNodeRouteUndefined, err)
	}

}
