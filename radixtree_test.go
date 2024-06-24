package churro

import (
	"errors"
	"testing"
)

type TestNode struct {
	Path    string
	Method  HttpMethod
	Matcher map[string]string
}

func (n *TestNode) path() string {
	return n.Path
}

func (n *TestNode) method() HttpMethod {
	return n.Method
}

func (n *TestNode) matcher() map[string]string {
	return n.Matcher
}

func (n *TestNode) String() string {
	return n.Path
}

func TestNewRadixTree(t *testing.T) {

	tree := NewRadixTree()

	table := []*TestNode{
		{"api/users", "GET", nil},
		{"api/users/:id", "GET", nil},
		{"api/users/:id", "DELETE", nil},
		{"api/users/:id/items", "GET", nil},
		{"api/items", "GET", nil},
		{"api/items", "POST", nil},
		{"api/invoices/:any", "GET", map[string]string{"any": `\w+`}},
	}

	for _, node := range table {
		tree.Insert(node)
	}

	inserted := tree.Routes()

	if len(inserted) != len(table) {
		t.Errorf("Inserted wrong number of nodes: got %d, want %d", len(table), len(inserted))
	}

	if _, err := tree.Search("api/users/1", "GET"); err != nil {
		t.Errorf("Search failed, wanted %s got nil: %v", table[0].Path, err)
	}

	if _, err := tree.Search("api/users/1/items", "GET"); err != nil {
		t.Errorf("Search failed, wanted %s got nil: %v", table[3].Path, err)
	}

	if _, err := tree.Search("api/invoices/111", "GET"); err != nil {
		t.Errorf("Search failed, wanted %s got nil: %v", table[4].Path, err)
	}

	if _, err := tree.Search("api/invoices/abc", "GET"); err != nil {
		t.Errorf("Search error, wanted %s got nil: %v", table[6].Path, err)
	}

	if _, err := tree.Search("api/invoices", "GET"); !errors.Is(err, ErrNodeRouteUndefined) {
		t.Errorf("Search error, wanted: %v got: %v", ErrNodeRouteUndefined, err)
	}

	tree.Remove(table[0])

	if len(tree.Routes()) != len(table)-1 {
		t.Errorf("Wrong number of nodes after removal: got %d, want %d", len(table)-1, len(tree.Routes()))
	}

	if _, err := tree.Search("api/users", "GET"); !errors.Is(err, ErrNodeRouteUndefined) {
		t.Errorf("Search error, wanted: %v got: %v", ErrNodeRouteUndefined, err)
	}

}
