package middleware

import (
	"net/http"

	"github.com/lexis-app/lexis-api/internal/modules/auth/domain"
	"github.com/lexis-app/lexis-api/internal/shared/httputil"
)

// RequireScope blocks the request unless the JWT-extracted scope set
// (see Auth + GetScopes) contains the named scope verbatim. It does NOT
// honour any "super" scope — every route declares the granular scope it
// needs, so a token cannot escalate by holding admin:full instead of the
// scope the route asked for.
//
// Place this AFTER middleware.Auth in the chain. Without prior scopes
// in context the check rejects, which is the right default for an
// authentication that does not happen.
//
// Refs: reflective-agent-defaults v1.3 Rule 4 (granular RBAC, no magic
// bypass).
func RequireScope(required domain.Scope) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			scopes := GetScopes(r.Context())
			for _, s := range scopes {
				if s == required {
					next.ServeHTTP(w, r)
					return
				}
			}
			httputil.WriteProblem(
				w,
				http.StatusForbidden,
				"Forbidden",
				"missing required scope: "+string(required),
			)
		})
	}
}
