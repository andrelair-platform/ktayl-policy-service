package middleware

import (
	"net/http"
	"strings"
)

// RequireScope returns middleware that rejects the request with 403 if the scope claim
// stored by ValidateJWT does not contain the required scope string.
func RequireScope(required string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			scope := ScopeFromContext(r.Context())
			for _, s := range strings.Fields(scope) {
				if s == required {
					next.ServeHTTP(w, r)
					return
				}
			}
			writeAuthProblem(w, http.StatusForbidden, "insufficient_scope", "required scope: "+required)
		})
	}
}
