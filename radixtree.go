package churro

import (
	"errors"
	"fmt"
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
	Path        string
	Method      HttpMethod
	Matcher     map[string]string
	Middlewares []Middleware
	Handler     http.Handler
	// Node is the Node to which the route handler is attached to
	Node *Node
}

type Node struct {
	RouteHandlers map[HttpMethod]*RouteHandler
	Middleware    []Middleware
	Prefix        string
	Children      []*Node
	ParamKey      *string
	Parent        *Node
}

func (n *Node) isLeaf() bool {
	return n.RouteHandlers != nil
}

func (n *Node) isPathParam() bool {
	return strings.HasPrefix(n.Prefix, ":")
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
	return &RadixTree{
		root: NewNode("/"),
	}
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
		siblings := (*node.Parent).Children
		for i, child := range siblings {
			if child == node {
				siblings = append(siblings[:i], siblings[i+1:]...)
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

	segments := SplitString(path, "/")

	for len(segments) > 0 && curNode != nil {

		segment := segments[0]
		segments = segments[1:]

		var segmentNode *Node

		for _, child := range curNode.Children {
			// Here we follow the path segments when one of the following cases is fulfilled
			// a) the current node matches the segment (following the path)
			// b) the current node is a path parameter (we can continue to the next)
			// c) the current node is a path parameter with matcher (we can continue to the next if matching)

			if !callback(child, segment) {
				//return nil, ErrNodeNotFound
				continue
			}

			segmentNode = child
		}

		// No node was there, so we have to create a new one and link it the to current node.segmentNode
		if createNodes && segmentNode == nil {
			segmentNode = NewNode(segment)
			segmentNode.Parent = curNode
			curNode.Children = append(curNode.Children, segmentNode)
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

	handlerNode, err := rt.WalkSegments(path, false, func(node *Node, segment string) bool {

		if node.Prefix == segment { // a)

			return true

		} else if node.isPathParam() && !node.isPathParamMatcher(method) { // b)

			return true

		} else if node.isPathParam() && node.isPathParamMatcher(method) { // c)

			matches, err := node.matchesPathParamMatcher(segment, method)

			if err != nil {
				return false
			}

			if matches {
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

	return routeHandler, nil
}

func (rt *RadixTree) GetRouteMiddlewares(routeHandler *RouteHandler) []Middleware {

	curNode := routeHandler.Node
	middlewares := routeHandler.Middlewares

	for curNode != nil {
		middlewaresCount := len(curNode.Middleware)
		for i := middlewaresCount - 1; i >= 0; i-- {
			middlewares = append(middlewares, curNode.Middleware[i])
		}
		curNode = curNode.Parent
	}

	return middlewares
}

// Match finds the matched route handler and middlewares for the given method and path.
func (rt *RadixTree) Match(method HttpMethod, path string) (*RouteHandler, []Middleware, error) {

	routeHandler, err := rt.SearchPath(path, method)

	if err != nil {
		return nil, nil, fmt.Errorf("failed to find matches for %s %s %w ", method, path, err)
	}

	return routeHandler, rt.GetRouteMiddlewares(routeHandler), nil
}

func (rt *RadixTree) AddRoute(routeHandler *RouteHandler) {
	rt.Insert(routeHandler)
}

func (rt *RadixTree) RemoveRoute(method HttpMethod, path string) {
	rt.RemoveRouteHandler(method, path)
}

func (rt *RadixTree) AddMiddleware(method HttpMethod, path string, middleware ...Middleware) error {
	routeHandler, err := rt.SearchPath(path, method)
	if err != nil {
		return fmt.Errorf("set route middleware failed %w", err)
	}
	routeHandler.Middlewares = append(routeHandler.Middlewares, middleware...)
	return nil
}

func (rt *RadixTree) AddRouterMiddleware(path string, middleware ...Middleware) error {

	node, err := rt.WalkSegments(path, false, func(node *Node, segment string) bool {
		return node.Prefix == segment
	})

	if err != nil {
		return fmt.Errorf("set router middleware failed %w", err)
	}

	node.Middleware = append(node.Middleware, middleware...)
	return nil
}
