package churro

import (
	"net/http"
)

type RequestContext interface {
	GetBody() any
	GetHeaders() any
	GetPathParams() any
	GetQueryParams() any
	SetReq(req *http.Request)
	SetRes(w http.ResponseWriter)
}

type RawContext[Body, QueryParams, Headers, PathParams any] struct {
	Req         *http.Request
	Res         http.ResponseWriter
	Headers     *Headers
	PathParams  *PathParams
	QueryParams *QueryParams
	Body        *Body
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

func (c *RawContext[Body, QueryParams, Headers, PathParams]) SetQueryParams(queryParams any) {
	*c.QueryParams = queryParams.(QueryParams)
}
func (c *RawContext[Body, QueryParams, Headers, PathParams]) SetHeaders(headers any) {
	*c.Headers = headers.(Headers)
}
func (c *RawContext[Body, QueryParams, Headers, PathParams]) SetPathParams(pathParams any) {
	*c.PathParams = pathParams.(PathParams)
}
func (c *RawContext[Body, QueryParams, Headers, PathParams]) GetPathParam(name string) string {
	return GetPathParam(c.Req, name)
}

func (c *RawContext[Body, QueryParams, Headers, PathParams]) SetRes(res http.ResponseWriter) {
	c.Res = res
}

func (c *RawContext[Body, QueryParams, Headers, PathParams]) SetReq(req *http.Request) {
	c.Req = req
}

type Context struct {
	RawContext[any, any, any, any]
	Req *http.Request
	Res http.ResponseWriter
}

type ContextWithBody[Body any] struct {
	RawContext[Body, any, any, any]
}

type ContextWithBodyAndQuery[Body, QueryParams any] struct {
	RawContext[Body, QueryParams, any, any]
}
