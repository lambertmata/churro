package churro

import (
	"strings"
)

type Routable interface {
	path() string
	method() HttpMethod
}

type Node struct {
	Data     map[HttpMethod]*Routable
	Prefix   string
	Children []*Node
}

func (n *Node) isLeaf() bool {
	return n.Data != nil
}

func NewNode(prefix string) *Node {
	return &Node{Prefix: prefix}
}

type RadixTree struct {
	root *Node
}

func NewRadixTree() *RadixTree {
	root := &Node{Prefix: "/"}
	return &RadixTree{root: root}
}

// FindInsertionNode searches the appropriate Node for insertion.
func (rt *RadixTree) FindInsertionNode(routable Routable) *Node {

	// Splitting the path into segments.
	segments := strings.Split(routable.path(), "/")
	curNode := rt.root

	// Starting from the root, descends the tree following the path segments, creating intermediary segment nodes when
	// missing, and returns the node where to attach the data to.
	for len(segments) > 0 {

		// On each iteration the first path segment is dequeued, until we have none.
		segment := segments[0]
		segments = segments[1:]

		var segmentNode *Node

		// Check if the current segment is at the current tree level.
		for _, child := range curNode.Children {
			if child.Prefix == segment {
				segmentNode = child
				break
			}
		}

		// No node was there, so we have to create a new one and link it the to current node.
		if segmentNode == nil {
			segmentNode = NewNode(segment)
			curNode.Children = append(curNode.Children, segmentNode)
		}

		// We can proceed by inspecting the segment node found
		curNode = segmentNode
	}

	return curNode
}

func (rt *RadixTree) Insert(data Routable) {

	leaf := rt.FindInsertionNode(data)

	if leaf == nil {
		return
	}

	if leaf.Data == nil {
		leaf.Data = make(map[HttpMethod]*Routable)
	}

	leaf.Data[data.method()] = &data
}

func (rt *RadixTree) Traverse(callback func(node *Node)) {

	stack := []*Node{rt.root}

	for len(stack) > 0 {

		curNode := stack[len(stack)-1]
		stack = stack[:len(stack)-1]

		if curNode.isLeaf() {
			callback(curNode)
		}

		stack = append(stack, curNode.Children...)
	}
}

func (rt *RadixTree) Nodes() []*Node {
	var nodes []*Node
	rt.Traverse(func(node *Node) {
		nodes = append(nodes, node)
	})
	return nodes
}
