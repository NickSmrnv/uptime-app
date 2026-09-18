package server

import "net/http"

// WithAPICORS allows credentialed browser requests from one configured origin
// to browser-facing API endpoints.
func WithAPICORS(next http.Handler, allowedOrigin string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if isBrowserAPIRoute(r.URL.Path) && r.Header.Get("Origin") == allowedOrigin {
			w.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Access-Control-Allow-Methods", "DELETE, GET, PATCH, POST, PUT")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			w.Header().Add("Vary", "Origin")
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

func isBrowserAPIRoute(path string) bool {
	if path == "/auth" || (len(path) > len("/auth/") && path[:len("/auth/")] == "/auth/") {
		return true
	}
	switch path {
	case "/users/me", "/users/me/avatar", "/monitors", "/uploads":
		return true
	default:
		return len(path) > len("/monitors/") && path[:len("/monitors/")] == "/monitors/"
	}
}
