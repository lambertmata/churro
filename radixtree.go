package churro

import (
	"regexp"
	"strings"
)

type Routable interface {
	path() string
	method() HttpMethod
	matcher() map[string]string
}

type Node struct {
	Data     map[HttpMethod]*Routable
	Prefix   string
	Children []*Node
	ParamKey *string
}

func (n *Node) isLeaf() bool {
	return n.Data != nil
}

func (n *Node) isPathParam() bool {
	return strings.HasPrefix(n.Prefix, ":")
}

func (n *Node) isPathParamMatcher(method HttpMethod) bool {
	if !n.isPathParam() || n.Data == nil || len(n.Data) == 0 {
		return false
	}

	data, ok := n.Data[method]

	if !ok || data == nil {
		return false
	}

	if _, ok := (*data).matcher()[*n.ParamKey]; !ok {
		return false
	}

	return true
}

func (n *Node) matchesPathParamMatcher(segment string, method HttpMethod) bool {
	data, ok := n.Data[method]

	pattern, ok := (*data).matcher()[*n.ParamKey]

	if !ok {
		return false
	}

	match, err := regexp.MatchString(pattern, segment)

	if err != nil {
		return false
	}

	return match
}

func NewNode(prefix string) *Node {
	node := &Node{Prefix: prefix}
	if node.isPathParam() {
		paramKey := strings.TrimPrefix(node.Prefix, ":")
		node.ParamKey = &paramKey
	}
	return node
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

// Search searches the appropriate Node for insertion.
func (rt *RadixTree) Search(path string, method HttpMethod) *Routable {

	// Similarly to FindInsertionNode it will traverse the tree by path segments, but it will not create
	// intermediate nodes when missing.

	segments := strings.Split(path, "/")
	curNode := rt.root

	for len(segments) > 0 && curNode != nil {

		segment := segments[0]
		segments = segments[1:]

		var segmentNode *Node

		for _, child := range curNode.Children {
			// Here we follow the path segments when one of the following cases is fulfilled
			// - the current node matches the segment (following the path)
			// - the current node is a path parameter (we can continue to the next)
			// - the current node is a path parameter with matcher (we can continue to the next if matching)
			if child.Prefix == segment ||
				(child.isPathParam() && !child.isPathParamMatcher(method)) ||
				(child.isPathParamMatcher(method) && child.matchesPathParamMatcher(segment, method)) {
				segmentNode = child
				break
			}
		}

		curNode = segmentNode
	}

	if curNode == nil || curNode.Data == nil {
		return nil
	}

	data, ok := curNode.Data[method]

	if !ok {
		return nil
	}

	return data
}
