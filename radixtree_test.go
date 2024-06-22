package churro

import (
	"testing"
)

type TestNode struct {
	Path string
}

func (n *TestNode) path() string {
	return n.Path
}

func (n *TestNode) String() string {
	return n.Path
}

func TestNewRadixTree(t *testing.T) {
	tree := NewRadixTree()
	n1 := &TestNode{"api/users"}
	n2 := &TestNode{"api/users/:id"}
	n3 := &TestNode{"api/users/:id/items"}
	n4 := &TestNode{"api/items"}
	tree.Insert(n1)
	tree.Insert(n2)
	tree.Insert(n3)
	tree.Insert(n4)

	inserted := tree.Nodes()

	if len(inserted) != 4 {
		t.Errorf("Inserted wrong number of nodes: got %d, want 4", len(inserted))
	}

}
