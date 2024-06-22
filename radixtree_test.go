package churro

import (
	"fmt"
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
	tree.Insert(n1)
	tree.Insert(n2)
	tree.Insert(n3)
	tree.Insert(n4)
	tree.Insert(n5)
	tree.Insert(n6)

	inserted := tree.Nodes()

	if len(inserted) != 4 {
		t.Errorf("Inserted wrong number of nodes: got %d, want 6", len(inserted))
	}

	for _, n := range inserted {
		for method, route := range n.Data {
			slog.Info(fmt.Sprintf("%s %s", method, (*route).path()))
		}

	}

}
