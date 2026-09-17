package server

import "net/http"

// WithAPICORS permits credentialed browser access from one configured origin only for API routes.
// A single exact origin and route allowlist avoid reflecting arbitrary origins or enabling CORS for files.
func WithAPICORS(next http.Handler, allowedOrigin string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		isAPIRoute := r.URL.Path == "/auth" || (len(r.URL.Path) > len("/auth/") && r.URL.Path[:len("/auth/")] == "/auth/") || r.URL.Path == "/users/me" || r.URL.Path == "/users/me/avatar" || r.URL.Path == "/uploads"
		if isAPIRoute && r.Header.Get("Origin") == allowedOrigin {
			w.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Access-Control-Allow-Methods", "GET, PATCH, POST, PUT")
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
