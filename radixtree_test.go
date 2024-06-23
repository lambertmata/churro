package churro

import (
	"log/slog"
	"testing"
)

type TestNode struct {
	Path   string
	Method HttpMethod
}

func (n *TestNode) path() string {
	return n.Path
}

func (n *TestNode) method() HttpMethod {
	return n.Method
}

func (n *TestNode) matcher() map[string]string {
	return map[string]string{"any": `\d+`}
}

func (n *TestNode) String() string {
	return n.Path
}

func TestNewRadixTree(t *testing.T) {
	tree := NewRadixTree()
	n1 := &TestNode{"api/users", "GET"}
	n2 := &TestNode{"api/users/:id", "GET"}
	n3 := &TestNode{"api/users/:id", "DELETE"}
	n4 := &TestNode{"api/users/:id/items", "GET"}
	n5 := &TestNode{"api/items", "GET"}
	n6 := &TestNode{"api/items", "POST"}
	n7 := &TestNode{"api/invoices/:any", "GET"}
	tree.Insert(n1)
	tree.Insert(n2)
	tree.Insert(n3)
	tree.Insert(n4)
	tree.Insert(n5)
	tree.Insert(n6)
	tree.Insert(n7)

	inserted := tree.Nodes()

	if len(inserted) != 7 {
		t.Errorf("Inserted wrong number of nodes: got %d, want 6", len(inserted))
	}

	s1 := tree.Search("api/users/:id", "GET")

	if s1 != nil {
		slog.Info((*s1).path())
	}

	s2 := tree.Search("api/users/1/items", "GET")

	if s2 != nil {
		slog.Info((*s2).path())
	}

	s3 := tree.Search("api/invoices/111", "GET")

	if s3 != nil {
		slog.Info((*s3).path())
	}

}
