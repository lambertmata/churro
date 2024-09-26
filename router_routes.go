package churro

import "net/http"

func (r *Router) Get(path string, handler http.HandlerFunc) *Route {
	return r.Request(MethodGet, path, handler)
}

func (r *Router) Post(path string, handler http.HandlerFunc) *Route {
	return r.Request(MethodPost, path, handler)
}

func (r *Router) Put(path string, handler http.HandlerFunc) *Route {
	return r.Request(MethodPut, path, handler)
}

func (r *Router) Patch(path string, handler http.HandlerFunc) *Route {
	return r.Request(MethodPatch, path, handler)
}

func (r *Router) Delete(path string, handler http.HandlerFunc) *Route {
	return r.Request(MethodDelete, path, handler)
}

func (r *Router) Connect(path string, handler http.HandlerFunc) *Route {
	return r.Request(MethodConnect, path, handler)
}

func (r *Router) Trace(path string, handler http.HandlerFunc) *Route {
	return r.Request(MethodTrace, path, handler)
}

func (r *Router) Head(path string, handler http.HandlerFunc) *Route {
	return r.Request(MethodHead, path, handler)
}

func (r *Router) Option(path string, handler http.HandlerFunc) *Route {
	return r.Request(MethodOption, path, handler)
}
