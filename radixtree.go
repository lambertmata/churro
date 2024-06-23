package churro

import (
	"errors"
	"regexp"
	"strings"
)

// ErrNodeNotFound is returned by [RadixTree.Search] and [RadixTree.FindInsertionNode] a node containing a Route is
// not found, meaning the Route does not exist.
var ErrNodeNotFound = errors.New("route node not found")

// ErrNodeRouteUndefined is returned by [RadixTree.Search] when a node is found, but [Node.Route] is not set in the
// returned node.
var ErrNodeRouteUndefined = errors.New("route is undefined")

// ErrRoutedMethodNotImplemented is returned by [RadixTree.Search] when a route node is found but [Node.Route] does not
// not implement the requested method.
var ErrRoutedMethodNotImplemented = errors.New("route method not implemented")

// ErrPathParamMatcherNotDefined is returned by [RadixTree.Search] when a route node is found but [Node.Route] but the
// path param matcher is not defined.
var ErrPathParamMatcherNotDefined = errors.New("path param matcher not found")

type Routable interface {
	path() string
	method() HttpMethod
	matcher() map[string]string
}

type Node struct {
	Route    map[HttpMethod]*Routable
	Prefix   string
	Children []*Node
	ParamKey *string
}

func (n *Node) isLeaf() bool {
	return n.Route != nil
}

func (n *Node) isPathParam() bool {
	return strings.HasPrefix(n.Prefix, ":")
}

func (n *Node) isPathParamMatcher(method HttpMethod) bool {

	if !n.isPathParam() || n.Route == nil || len(n.Route) == 0 {
		return false
	}

	data, dataForMethodFound := n.Route[method]

	if !dataForMethodFound || data == nil {
		return false
	}

	_, matcherForParamFound := (*data).matcher()[*n.ParamKey]

	return matcherForParamFound
}

func (n *Node) matchesPathParamMatcher(segment string, method HttpMethod) (bool, error) {
	data, ok := n.Route[method]

	pattern, ok := (*data).matcher()[*n.ParamKey]

	if !ok {
		return false, ErrPathParamMatcherNotDefined
	}

	return regexp.MatchString(pattern, segment)
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

	if leaf.Route == nil {
		leaf.Route = make(map[HttpMethod]*Routable)
	}

	leaf.Route[data.method()] = &data
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
func (rt *RadixTree) Search(path string, method HttpMethod) (*Routable, error) {

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
			// a) the current node matches the segment (following the path)
			// b) the current node is a path parameter (we can continue to the next)
			// c) the current node is a path parameter with matcher (we can continue to the next if matching)

			if child.Prefix == segment { // a)
				segmentNode = child
				break

			} else if child.isPathParam() && !child.isPathParamMatcher(method) { // b)
				segmentNode = child
				break

			} else if child.isPathParam() && child.isPathParamMatcher(method) { // c)
				matches, err := child.matchesPathParamMatcher(segment, method)
				if err != nil {
					return nil, err
				}
				if matches {
					segmentNode = child
					break
				}
			}

		}

		curNode = segmentNode
	}

	// Nothing found
	if curNode == nil {
		return nil, ErrNodeNotFound
	}

	// Node was found, but Route is not defined (should not happen)
	if curNode.Route == nil {
		return nil, ErrNodeRouteUndefined
	}

	data, ok := curNode.Route[method]

	// Route was found, but the supplied method is not implemented
	if !ok {
		return nil, ErrRoutedMethodNotImplemented
	}

	return data, nil
}
