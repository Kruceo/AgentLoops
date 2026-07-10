package server

import (
	"net/http"
	"os"

	apperrors "agentloops/core/errors"
)

// CORS middleware adds CORS headers to all responses and handles preflight requests.
func CORS(next http.Handler) http.Handler {
	origin := os.Getenv("ALLOWED_ORIGIN")
	if origin == "" {
		origin = "http://localhost:3000"
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// AuthMiddleware returns a middleware that validates Bearer tokens.
// If token is empty, auth is skipped (dev mode).
func AuthMiddleware(token string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if token == "" {
				next.ServeHTTP(w, r)
				return
			}

			authHeader := r.Header.Get("Authorization")
			expected := "Bearer " + token
			if authHeader != expected {
				handleError(w, apperrors.ErrUnauthorized)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// NewServer creates an HTTP server with all routes registered.
func NewServer(addr string, h *Handler) *http.Server {
	router := NewRouter()

	authToken := os.Getenv("AUTH_TOKEN")
	router.Use(AuthMiddleware(authToken))
	router.Use(CORS)

	h.RegisterRoutes(router)

	return &http.Server{
		Addr:    addr,
		Handler: router,
	}
}
