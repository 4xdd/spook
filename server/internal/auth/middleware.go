package auth

import (
	"net/http"
	"strings"

	"github.com/spook/server/internal/httpx"
)

const headerName = "X-Spook-Access-Key"

// Require wraps next so every /api/ request must present a valid access key.
// Static assets, the SPA, and /health stay open so the gate UI can load.
func Require(keys *Keys, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !needsKey(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}
		if r.Method == http.MethodOptions {
			next.ServeHTTP(w, r)
			return
		}
		key := extractKey(r)
		if !keys.Valid(key) {
			w.Header().Set("WWW-Authenticate", `Bearer realm="spook"`)
			httpx.WriteError(w, http.StatusUnauthorized, "access key required")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func needsKey(path string) bool {
	return strings.HasPrefix(path, "/api/")
}

func extractKey(r *http.Request) string {
	if auth := r.Header.Get("Authorization"); strings.HasPrefix(strings.ToLower(auth), "bearer ") {
		return strings.TrimSpace(auth[7:])
	}
	if key := strings.TrimSpace(r.Header.Get(headerName)); key != "" {
		return key
	}
	if key := strings.TrimSpace(r.URL.Query().Get("key")); key != "" {
		return key
	}
	return strings.TrimSpace(r.URL.Query().Get("access_key"))
}
