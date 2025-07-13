package churro

import (
	"fmt"
	"io"
	"net/http"
)

type RequestContext interface {
	GetBody() any
	GetHeaders() any
	GetPathParams() any
	GetQueryParams() any
	SetReq(req *http.Request)
	SetRes(w http.ResponseWriter)
	getResponse() any
}

type RawContext[Body, QueryParams, Headers, PathParams any] struct {
	Req         *http.Request
	Res         http.ResponseWriter
	Headers     *Headers
	PathParams  *PathParams
	QueryParams *QueryParams
	Body        *Body
	response    any
}

func (c *RawContext[Body, QueryParams, Headers, PathParams]) GetBody() any {
	if c.Body == nil {
		c.Body = new(Body)
	}
	return c.Body
}

func (c *RawContext[Body, QueryParams, Headers, PathParams]) GetHeaders() any {
	if c.Headers == nil {
		c.Headers = new(Headers)
	}
	return c.Headers
}

func (c *RawContext[Body, QueryParams, Headers, PathParams]) GetPathParams() any {
	return c.PathParams
}

func (c *RawContext[Body, QueryParams, Headers, PathParams]) GetQueryParams() any {
	return c.QueryParams
}

func (c *RawContext[Body, QueryParams, Headers, PathParams]) SetQueryParams(queryParams any) error {
	qp, ok := queryParams.(QueryParams)
	if !ok {
		return fmt.Errorf("invalid query params type: expected %T, got %T", *new(QueryParams), queryParams)
	}
	*c.QueryParams = qp
	return nil
}

func (c *RawContext[Body, QueryParams, Headers, PathParams]) SetHeaders(headers any) error {
	h, ok := headers.(Headers)
	if !ok {
		return fmt.Errorf("invalid headers type: expected %T, got %T", *new(Headers), headers)
	}
	*c.Headers = h
	return nil
}

func (c *RawContext[Body, QueryParams, Headers, PathParams]) SetPathParams(pathParams any) error {
	pp, ok := pathParams.(PathParams)
	if !ok {
		return fmt.Errorf("invalid path params type: expected %T, got %T", *new(PathParams), pathParams)
	}
	*c.PathParams = pp
	return nil
}
func (c *RawContext[Body, QueryParams, Headers, PathParams]) PathParam(name string) string {
	return GetPathParam(c.Req, name)
}

func (c *RawContext[Body, QueryParams, Headers, PathParams]) SetRes(res http.ResponseWriter) {
	c.Res = res
}

func (c *RawContext[Body, QueryParams, Headers, PathParams]) SetReq(req *http.Request) {
	c.Req = req
}

func (c *RawContext[Body, QueryParams, Headers, PathParams]) SendString(data string, middlewares ...ResponseMiddleware) error {
	c.response = &HandlerResponse[any]{
		payload:     data,
		middlewares: middlewares,
	}
	return nil
}

func (c *RawContext[Body, QueryParams, Headers, PathParams]) SendBytes(data []byte, middlewares ...ResponseMiddleware) error {
	c.response = &HandlerResponse[[]byte]{
		payload:     data,
		middlewares: middlewares,
	}
	return nil
}

func (c *RawContext[Body, QueryParams, Headers, PathParams]) SendJSON(data any, middlewares ...ResponseMiddleware) error {
	middlewares = append(middlewares, WithJSONContentType(), WithWrappedData())
	c.response = &HandlerResponse[any]{
		payload:     data,
		middlewares: middlewares,
	}
	return nil
}

func (c *RawContext[Body, QueryParams, Headers, PathParams]) SendStream(data io.Reader, middlewares ...ResponseMiddleware) error {
	c.response = &HandlerResponse[io.Reader]{
		payload:     data,
		middlewares: middlewares,
	}
	return nil
}

func (c *RawContext[Body, QueryParams, Headers, PathParams]) getResponse() any {
	return c.response
}

func (c *RawContext[Body, QueryParams, Headers, PathParams]) Response(data any, middleware ...ResponseMiddleware) error {
	c.response = data
	return nil
}

type Context struct {
	RawContext[any, any, any, any]
}

type ContextWithBody[Body any] struct {
	RawContext[Body, any, any, any]
}

type ContextWithBodyAndQuery[Body, QueryParams any] struct {
	RawContext[Body, QueryParams, any, any]
}
