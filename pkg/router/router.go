package router

import (
	"context"
	"net/http"
	"regexp"
	"strings"
	"sync"
)

// Route represents an HTTP route with its handler and metadata.
type Route struct {
	Method      string
	Pattern     string
	Handler     http.Handler
	Middlewares []Middleware
	Params      []string
	Regex       *regexp.Regexp
}

// Router is a custom HTTP router that supports pattern matching
// with parameters and middleware.
type Router struct {
	mu         sync.RWMutex
	routes     map[string][]Route
	middleware []Middleware
	notFound   http.Handler
	notAllowed http.Handler
	paramCache map[string]*regexp.Regexp
}

// New creates a new Router instance.
func New() *Router {
	return &Router{
		routes:   make(map[string][]Route),
		notFound: http.NotFoundHandler(),
		notAllowed: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		}),
		paramCache: make(map[string]*regexp.Regexp),
	}
}

// GET is a shortcut for adding a route with GET method.
func (r *Router) GET(pattern string, handler http.Handler) {
	r.AddRoute(http.MethodGet, pattern, handler)
}

// POST is a shortcut for adding a route with POST method.
func (r *Router) POST(pattern string, handler http.Handler) {
	r.AddRoute(http.MethodPost, pattern, handler)
}

// PUT is a shortcut for adding a route with PUT method.
func (r *Router) PUT(pattern string, handler http.Handler) {
	r.AddRoute(http.MethodPut, pattern, handler)
}

// DELETE is a shortcut for adding a route with DELETE method.
func (r *Router) DELETE(pattern string, handler http.Handler) {
	r.AddRoute(http.MethodDelete, pattern, handler)
}

// PATCH is a shortcut for adding a route with PATCH method.
func (r *Router) PATCH(pattern string, handler http.Handler) {
	r.AddRoute(http.MethodPatch, pattern, handler)
}

// OPTIONS is a shortcut for adding a route with OPTIONS method.
func (r *Router) OPTIONS(pattern string, handler http.Handler) {
	r.AddRoute(http.MethodOptions, pattern, handler)
}

// HEAD is a shortcut for adding a route with HEAD method.
func (r *Router) HEAD(pattern string, handler http.Handler) {
	r.AddRoute(http.MethodHead, pattern, handler)
}

// AddRoute adds a new route with the specified method and pattern.
func (r *Router) AddRoute(method, pattern string, handler http.Handler) {
	r.mu.Lock()
	defer r.mu.Unlock()

	params, re := r.compilePattern(pattern)
	route := Route{
		Method:      method,
		Pattern:     pattern,
		Handler:     handler,
		Params:      params,
		Regex:       re,
		Middlewares: nil,
	}
	r.routes[method] = append(r.routes[method], route)
}

// compilePattern converts a route pattern to a regex and extracts parameter names.
func (r *Router) compilePattern(pattern string) ([]string, *regexp.Regexp) {
	var params []string
	pattern = "^" + pattern

	// Replace :name parameters with named capture groups
	pattern = strings.ReplaceAll(pattern, ":id", "(?P<id>[^/]+)")
	pattern = strings.ReplaceAll(pattern, ":name", "(?P<name>[^/]+)")
	pattern = strings.ReplaceAll(pattern, ":slug", "(?P<slug>[^/]+)")
	pattern = strings.ReplaceAll(pattern, ":uuid", "(?P<uuid>[0-9a-fA-F-]+)")
	pattern = strings.ReplaceAll(pattern, ":int", "(?P<int>[0-9]+)")
	pattern = strings.ReplaceAll(pattern, ":commentId", "(?P<commentId>[^/]+)")

	// Extract parameter names from the pattern
	for _, part := range strings.Split(pattern, "/") {
		if strings.HasPrefix(part, ":") {
			params = append(params, strings.TrimPrefix(part, ":"))
		}
	}

	// Handle wildcard patterns
	if strings.HasSuffix(pattern, "/*") {
		pattern = strings.TrimSuffix(pattern, "/*") + "(?P<wildcard>/.*)"
	}

	pattern += "$"

	// Use cached regex if available
	if re, ok := r.paramCache[pattern]; ok {
		return params, re
	}

	re := regexp.MustCompile(pattern)
	r.paramCache[pattern] = re
	return params, re
}

// ServeHTTP implements http.Handler interface.
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	r.mu.RLock()
	middleware := r.middleware
	r.mu.RUnlock()

	methodRoutes := r.routes[req.Method]
	if len(methodRoutes) == 0 {
		r.notAllowed.ServeHTTP(w, req)
		return
	}

	for _, route := range methodRoutes {
		params := r.matchRoute(req.URL.Path, route)
		if params != nil {
			// Create context with params
			ctx := WithParams(req.Context(), params)

			// Build handler chain: global middleware + route handler
			handler := route.Handler
			for i := len(route.Middlewares) - 1; i >= 0; i-- {
				handler = route.Middlewares[i](handler)
			}
			for i := len(middleware) - 1; i >= 0; i-- {
				handler = middleware[i](handler)
			}

			handler.ServeHTTP(w, req.WithContext(ctx))
			return
		}
	}

	r.notFound.ServeHTTP(w, req)
}

// matchRoute checks if the URL path matches the route pattern.
func (r *Router) matchRoute(path string, route Route) Params {
	if route.Regex == nil {
		return nil
	}

	matches := route.Regex.FindStringSubmatch(path)
	if matches == nil {
		return nil
	}

	params := make(Params)
	for i, name := range route.Regex.SubexpNames() {
		if name != "" && i < len(matches) {
			params[name] = matches[i]
		}
	}

	return params
}

// Use adds a middleware to the router's global middleware chain.
func (r *Router) Use(middlewares ...Middleware) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.middleware = append(r.middleware, middlewares...)
}

// SetNotFoundHandler sets the handler for routes that don't match.
func (r *Router) SetNotFoundHandler(handler http.Handler) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.notFound = handler
}

// SetMethodNotAllowedHandler sets the handler for methods that don't match.
func (r *Router) SetMethodNotAllowedHandler(handler http.Handler) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.notAllowed = handler
}

// Routes returns a copy of all registered routes.
func (r *Router) Routes() []Route {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var routes []Route
	for _, methodRoutes := range r.routes {
		routes = append(routes, methodRoutes...)
	}
	return routes
}

// Handler is an adapter that allows using a function as an http.Handler.
type Handler func(http.ResponseWriter, *http.Request)

// ServeHTTP implements http.Handler.
func (h Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h(w, r)
}

// Context extends http.Request with additional helper methods.
type Context struct {
	http.ResponseWriter
	Request *http.Request
}

// Param returns the value of the URL parameter with the given key.
func (c *Context) Param(key string) string {
	params, ok := ParamsFromContext(c.Request.Context())
	if !ok {
		return ""
	}
	return params.Get(key)
}

// StatusCode returns the HTTP status code that was written.
func (c *Context) StatusCode() int {
	if sc, ok := c.ResponseWriter.(interface {
		GetStatus() int
	}); ok {
		return sc.GetStatus()
	}
	return http.StatusOK
}

// Redirect redirects the request to the given URL.
func (c *Context) Redirect(url string, code int) {
	http.Redirect(c.ResponseWriter, c.Request, url, code)
}

// WithContext returns a new request with the given context.
func (c *Context) WithContext(ctx context.Context) *http.Request {
	return c.Request.WithContext(ctx)
}
