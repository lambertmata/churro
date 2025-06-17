package churro

import (
	"errors"
	"fmt"
	"github.com/lambertmata/churro/utils"
	"net/http"
	"regexp"
	"strings"
)

// ErrNodeNotFound is returned by [RadixTree.Search] and [RadixTree.FindInsertionNode] when a node to the given path
// does not exist. If nil does not mean the Route is found as the node could be just a linking node without an actual
// route. For examples a Route defined with /api/item/tags when looking for /api/items would still return a node.
var ErrNodeNotFound = errors.New("route node not found")

// ErrNodeRouteUndefined is returned by [RadixTree.Search] when a node is found, but [Node.Route] route was not defined.
var ErrNodeRouteUndefined = errors.New("route is undefined")

// ErrRoutedMethodNotImplemented is returned by [RadixTree.Search] when a route node is found but [Node.Route] does not
// not implement the requested method.
var ErrRoutedMethodNotImplemented = errors.New("route method not implemented")

// ErrPathParamMatcherNotDefined is returned by [RadixTree.Search] when a route node is found but [Node.Route] but the
// path param matcher is not defined.
var ErrPathParamMatcherNotDefined = errors.New("path param matcher not found")

type RouteHandler struct {
	Path    string
	Method  HttpMethod
	Matcher map[string]string
	Handler http.Handler
	// Node is the Node to which the route handler is attached to
	Node        *Node
	ParamValues map[string]string
	Middlewares []Middleware
}

type Node struct {
	RouteHandlers map[HttpMethod]*RouteHandler
	Prefix        string
	Children      []*Node
	// ChildrenMap provides O(1) lookup for children by prefix
	ChildrenMap   map[string]*Node
	ParamKey      *string
	Parent        *Node
}

func (n *Node) isLeaf() bool {
	return n.RouteHandlers != nil
}

func (n *Node) isPathParam() bool {
	return strings.HasPrefix(n.Prefix, "{") && strings.HasSuffix(n.Prefix, "}")
}

func (n *Node) Param() string {
	return strings.Trim(n.Prefix, "{}")
}

func (n *Node) isPathParamMatcher(method HttpMethod) bool {

	if !n.isPathParam() || n.RouteHandlers == nil || len(n.RouteHandlers) == 0 {
		return false
	}

	data, dataForMethodFound := n.RouteHandlers[method]

	if !dataForMethodFound || data == nil {
		return false
	}

	_, matcherForParamFound := (*data).Matcher[*n.ParamKey]

	return matcherForParamFound
}

func (n *Node) matchesPathParamMatcher(segment string, method HttpMethod) (bool, error) {
	data, ok := n.RouteHandlers[method]

	pattern, ok := (*data).Matcher[*n.ParamKey]

	if !ok {
		return false, ErrPathParamMatcherNotDefined
	}

	return regexp.MatchString(pattern, segment)
}

func NewNode(prefix string) *Node {
	node := &Node{
		Prefix:      prefix,
		ChildrenMap: make(map[string]*Node),
	}
	if node.isPathParam() {
		paramKey := node.Param()
		node.ParamKey = &paramKey
	}
	return node
}

type RadixTree struct {
	root *Node
}

// initializeChildrenMaps recursively initializes the ChildrenMap for all nodes in the tree
func initializeChildrenMaps(node *Node) {
	if node.ChildrenMap == nil {
		node.ChildrenMap = make(map[string]*Node)
	}

	// Add all children to the map
	for _, child := range node.Children {
		node.ChildrenMap[child.Prefix] = child
		// Recursively initialize children's maps
		initializeChildrenMaps(child)
	}
}

func NewRadixTree() *RadixTree {
	tree := &RadixTree{
		root: NewNode("/"),
	}
	// Initialize ChildrenMap for all nodes
	initializeChildrenMaps(tree.root)
	return tree
}

// FindInsertionNode searches the appropriate Node for insertion.
func (rt *RadixTree) FindInsertionNode(routable *RouteHandler) *Node {

	node, _ := rt.WalkSegments(routable.Path, true, func(node *Node, segment string) bool {
		return node.Prefix == segment
	})

	return node
}

func (rt *RadixTree) Insert(data *RouteHandler) {

	leaf := rt.FindInsertionNode(data)

	if leaf == nil {
		return
	}

	if leaf.RouteHandlers == nil {
		leaf.RouteHandlers = make(map[HttpMethod]*RouteHandler)
	}

	data.Node = leaf
	leaf.RouteHandlers[data.Method] = data
}

func (rt *RadixTree) RemoveRouteHandler(method HttpMethod, path string) (bool, error) {
	routeHandler, err := rt.SearchPath(path, method)

	if err != nil {
		return false, err
	}

	// If the route exists we have two cases
	// a) the node is a leaf and can be removed from the tree
	// b) the node is not a leaf, we just remove the route data

	node := routeHandler.Node

	if len((*node).Children) > 0 {
		delete(node.RouteHandlers, method)
	} else {
		// Update the parent's Children slice
		parent := node.Parent
		for i, child := range parent.Children {
			if child == node {
				parent.Children = append(parent.Children[:i], parent.Children[i+1:]...)
				// Also remove from the map
				delete(parent.ChildrenMap, node.Prefix)
				break
			}
		}
	}

	return true, nil
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

func (rt *RadixTree) Routes() []*RouteHandler {
	var routes []*RouteHandler
	rt.Traverse(func(node *Node) {
		for _, route := range node.RouteHandlers {
			routes = append(routes, route)
		}
	})
	return routes
}

// WalkSegments traverses all the nodes.
func (rt *RadixTree) WalkSegments(path string, createNodes bool, callback func(node *Node, segment string) bool) (*Node, error) {

	// Similarly to FindInsertionNode it will traverse the tree by path segments, but it will not create
	// intermediate nodes when missing.

	curNode := rt.root

	if path == "/" {
		return curNode, nil
	}

	segments := utils.SplitString(path, "/")

	for len(segments) > 0 && curNode != nil {

		segment := segments[0]
		segments = segments[1:]

		var segmentNode *Node

		// First check if we have a direct match in the ChildrenMap (O(1) lookup)
		if child, exists := curNode.ChildrenMap[segment]; exists {
			if callback(child, segment) {
				segmentNode = child
			}
		} else {
			// If not found in map, check for path parameters or other special cases
			// that require checking all children

			// First, check for path parameter nodes (they start with "{" and end with "}")
			var pathParamNodes []*Node
			for _, child := range curNode.Children {
				if child.isPathParam() {
					pathParamNodes = append(pathParamNodes, child)
				}
			}

			// Check path parameter nodes first
			for _, child := range pathParamNodes {
				if callback(child, segment) {
					segmentNode = child
					break // Found a match, no need to continue
				}
			}

			// If no path parameter node matched, check remaining children
			if segmentNode == nil {
				for _, child := range curNode.Children {
					if child.isPathParam() {
						continue // Already checked path parameter nodes
					}

					if callback(child, segment) {
						segmentNode = child
						break // Found a match, no need to continue
					}
				}
			}
		}

		// No node was there, so we have to create a new one and link it the to current node.segmentNode
		if createNodes && segmentNode == nil {
			segmentNode = NewNode(segment)
			segmentNode.Parent = curNode
			curNode.Children = append(curNode.Children, segmentNode)
			// Also add to the map for O(1) lookup
			curNode.ChildrenMap[segment] = segmentNode
		}

		curNode = segmentNode
	}

	if curNode == nil {
		return nil, ErrNodeNotFound
	}

	return curNode, nil
}

// SearchPath searches the appropriate Node for insertion.
func (rt *RadixTree) SearchPath(path string, method HttpMethod) (*RouteHandler, error) {

	// Similarly to FindInsertionNode it will traverse the tree by path segments, but it will not create
	// intermediate nodes when missing.
	var pathParamValues map[string]string = make(map[string]string)

	handlerNode, err := rt.WalkSegments(path, false, func(node *Node, segment string) bool {

		if node.Prefix == segment { // a)
			return true

		} else if node.isPathParam() && !node.isPathParamMatcher(method) { // b)
			pathParamValues[node.Param()] = segment
			return true

		} else if node.isPathParam() && node.isPathParamMatcher(method) { // c)
			matches, err := node.matchesPathParamMatcher(segment, method)

			if err != nil {
				return false
			}

			if matches {
				pathParamValues[node.Param()] = segment
				return true
			}
		}

		return false
	})

	if err != nil || handlerNode.RouteHandlers == nil {
		return nil, fmt.Errorf("search path %s failed %w", path, ErrNodeNotFound)
	}

	routeHandler, ok := handlerNode.RouteHandlers[method]

	if !ok {
		return nil, fmt.Errorf("search method %s for path %s failed %w", method, path, ErrRoutedMethodNotImplemented)
	}

	routeHandler.ParamValues = pathParamValues

	return routeHandler, nil
}

// Match finds the matched route handler and middlewares for the given method and path.
func (rt *RadixTree) Match(method HttpMethod, path string) (*RouteHandler, error) {

	routeHandler, err := rt.SearchPath(path, method)

	if err != nil {
		return nil, fmt.Errorf("failed to find matches for %s %s %w ", method, path, err)
	}

	return routeHandler, nil
}

func (rt *RadixTree) AddRoute(routeHandler *RouteHandler) {
	rt.Insert(routeHandler)
}

func (rt *RadixTree) RemoveRoute(method HttpMethod, path string) {
	rt.RemoveRouteHandler(method, path)
}
