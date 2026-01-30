package router

import (
	"net/http"

	"quavixAI/pkg/response"
)

// ================================
// Context Abstraction
// ================================

type Context = response.Context

// ================================
// Handler Type
// ================================

type HandlerFunc func(Context) error

// ================================
// Middleware Type (CORE ABSTRACTION)
// ================================

// Middleware wraps a handler
type Middleware func(HandlerFunc) HandlerFunc

// ================================
// Router Core
// ================================

type Router struct {
	mux         *http.ServeMux
	middlewares []Middleware
}

func New() *Router {
	return &Router{
		mux:         http.NewServeMux(),
		middlewares: []Middleware{},
	}
}

// ================================
// Middleware
// ================================

func (r *Router) Use(m Middleware) {
	r.middlewares = append(r.middlewares, m)
}

// ================================
// Route Registration
// ================================

func (r *Router) handle(method, path string, h HandlerFunc) {
	h = r.applyMiddleware(h) // ← FIXED (no :=)

	r.mux.HandleFunc(path, func(w http.ResponseWriter, req *http.Request) {
		if req.Method != method {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		ctx := response.NewContext(w, req)
		if err := h(ctx); err != nil {
			_ = ctx.JSON(http.StatusInternalServerError, response.Error(err.Error()))
		}
	})
}

func (r *Router) GET(path string, h HandlerFunc) {
	r.handle(http.MethodGet, path, h)
}

func (r *Router) POST(path string, h HandlerFunc) {
	r.handle(http.MethodPost, path, h)
}

func (r *Router) PUT(path string, h HandlerFunc) {
	r.handle(http.MethodPut, path, h)
}

func (r *Router) DELETE(path string, h HandlerFunc) {
	r.handle(http.MethodDelete, path, h)
}

// ================================
// Groups
// ================================

type Group struct {
	prefix      string
	router      *Router
	middlewares []Middleware
}

func (r *Router) Group(prefix string) *Group {
	return &Group{
		prefix:      prefix,
		router:      r,
		middlewares: []Middleware{},
	}
}

func (g *Group) Use(m Middleware) {
	g.middlewares = append(g.middlewares, m)
}

func (g *Group) handle(method, path string, h HandlerFunc) {
	fullPath := g.prefix + path

	h = g.applyGroupMiddleware(h)
	g.router.handle(method, fullPath, h)
}

func (g *Group) GET(path string, h HandlerFunc) {
	g.handle(http.MethodGet, path, h)
}

func (g *Group) POST(path string, h HandlerFunc) {
	g.handle(http.MethodPost, path, h)
}

func (g *Group) PUT(path string, h HandlerFunc) {
	g.handle(http.MethodPut, path, h)
}

func (g *Group) DELETE(path string, h HandlerFunc) {
	g.handle(http.MethodDelete, path, h)
}

// ================================
// Middleware Application
// ================================

func (r *Router) applyMiddleware(h HandlerFunc) HandlerFunc {
	for i := len(r.middlewares) - 1; i >= 0; i-- {
		h = r.middlewares[i](h)
	}
	return h
}

func (g *Group) applyGroupMiddleware(h HandlerFunc) HandlerFunc {
	// group middleware
	for i := len(g.middlewares) - 1; i >= 0; i-- {
		h = g.middlewares[i](h)
	}
	// global router middleware
	for i := len(g.router.middlewares) - 1; i >= 0; i-- {
		h = g.router.middlewares[i](h)
	}
	return h
}

// ================================
// Server Hook
// ================================

func (r *Router) Handler() http.Handler {
	return r.mux
}
