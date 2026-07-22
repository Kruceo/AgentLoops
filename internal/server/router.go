package server

import (
	"context"
	"net/http"
	"strings"
)

// contextKey is used for storing path parameters in the request context.
type contextKey string

const paramsKey contextKey = "route_params"

// route represents a registered route with method, pattern, and handler.
type route struct {
	method  string
	pattern string // e.g., "/api/tasks/:id"
	handler http.HandlerFunc
}

// Router is a minimal HTTP request router supporting path parameters.
type Router struct {
	routes     []route
	middleware []func(http.Handler) http.Handler
}

// NewRouter creates a new Router.
func NewRouter() *Router {
	return &Router{}
}

// Use adds middleware to the router.
func (rt *Router) Use(mw func(http.Handler) http.Handler) {
	rt.middleware = append(rt.middleware, mw)
}

// HandleFunc registers a handler for a given method and pattern.
// Patterns support :param segments for path parameters.
func (rt *Router) HandleFunc(method, pattern string, handler http.HandlerFunc) {
	rt.routes = append(rt.routes, route{
		method:  method,
		pattern: pattern,
		handler: handler,
	})
}

// GET is a shortcut for HandleFunc("GET", ...).
func (rt *Router) GET(pattern string, handler http.HandlerFunc) {
	rt.HandleFunc("GET", pattern, handler)
}

// POST is a shortcut for HandleFunc("POST", ...).
func (rt *Router) POST(pattern string, handler http.HandlerFunc) {
	rt.HandleFunc("POST", pattern, handler)
}

// PUT is a shortcut for HandleFunc("PUT", ...).
func (rt *Router) PUT(pattern string, handler http.HandlerFunc) {
	rt.HandleFunc("PUT", pattern, handler)
}

// DELETE is a shortcut for HandleFunc("DELETE", ...).
func (rt *Router) DELETE(pattern string, handler http.HandlerFunc) {
	rt.HandleFunc("DELETE", pattern, handler)
}

// ServeHTTP implements the http.Handler interface.
func (rt *Router) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Build the final handler by wrapping with middleware
	var handler http.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for _, route := range rt.routes {
			if route.method != r.Method {
				continue
			}

			params, ok := matchPattern(route.pattern, r.URL.Path)
			if !ok {
				continue
			}

			if len(params) > 0 {
				ctx := context.WithValue(r.Context(), paramsKey, params)
				r = r.WithContext(ctx)
			}

			route.handler(w, r)
			return
		}

		// No route matched
		http.NotFound(w, r)
	})

	// Apply middleware in reverse order (first added = outermost)
	for i := len(rt.middleware) - 1; i >= 0; i-- {
		handler = rt.middleware[i](handler)
	}

	handler.ServeHTTP(w, r)
}

// GetParam extracts a path parameter from the request context.
func GetParam(r *http.Request, name string) string {
	params, ok := r.Context().Value(paramsKey).(map[string]string)
	if !ok {
		return ""
	}
	return params[name]
}

// matchPattern checks if a URL path matches a pattern and extracts parameters.
// Pattern example: "/api/tasks/:id" matches "/api/tasks/abc123" with {"id": "abc123"}.
func matchPattern(pattern, path string) (map[string]string, bool) {
	patternParts := splitPath(pattern)
	pathParts := splitPath(path)

	if len(patternParts) != len(pathParts) {
		return nil, false
	}

	params := make(map[string]string)

	for i, pp := range patternParts {
		if strings.HasPrefix(pp, ":") {
			// This is a parameter
			params[pp[1:]] = pathParts[i]
		} else if pp != pathParts[i] {
			return nil, false
		}
	}

	return params, true
}

// splitPath splits a URL path into non-empty segments.
func splitPath(path string) []string {
	var parts []string
	for _, part := range strings.Split(path, "/") {
		if part != "" {
			parts = append(parts, part)
		}
	}
	return parts
}
