package middleware

import (
	"context"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/lexis-app/lexis-api/internal/modules/auth/domain"
	"github.com/lexis-app/lexis-api/internal/shared/httputil"
)

type contextKey string

const (
	UserIDKey contextKey = "userID"
	scopesKey contextKey = "scopes"
)

// Blacklist checks whether a user's tokens have been globally invalidated.
type Blacklist interface {
	IsBlacklisted(ctx context.Context, key string) (bool, error)
}

// Auth validates the bearer JWT and admits the request with sub + scope
// information attached to the context. The legacyCutoff parameter is
// reserved for issue #9 (hard cutoff for tokens without a scope claim);
// the current contract still grants migration defaults to such tokens.
func Auth(jwtSecret []byte, blacklist Blacklist, legacyCutoff time.Time) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			auth := r.Header.Get("Authorization")
			if auth == "" || !strings.HasPrefix(auth, "Bearer ") {
				httputil.WriteProblem(w, http.StatusUnauthorized, "Unauthorized", "missing or invalid authorization header")
				return
			}

			tokenStr := strings.TrimPrefix(auth, "Bearer ")
			token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
				if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, jwt.ErrSignatureInvalid
				}
				return jwtSecret, nil
			})
			if err != nil || !token.Valid {
				httputil.WriteProblem(w, http.StatusUnauthorized, "Unauthorized", "invalid or expired token")
				return
			}

			sub, err := token.Claims.GetSubject()
			if err != nil || sub == "" {
				httputil.WriteProblem(w, http.StatusUnauthorized, "Unauthorized", "invalid token claims")
				return
			}

			// Check if user's tokens have been globally invalidated (LogoutAll).
			// Uses key "user_revoked:{userID}" set by LogoutAll with TTL = accessTTL.
			if blacklist != nil {
				revoked, err := blacklist.IsBlacklisted(r.Context(), "user_revoked:"+sub)
				if err != nil {
					log.Printf("auth: blacklist check error: %v", err)
					httputil.WriteProblem(w, http.StatusServiceUnavailable, "Service Unavailable", "unable to verify token status")
					return
				}
				if revoked {
					httputil.WriteProblem(w, http.StatusUnauthorized, "Unauthorized", "all sessions have been revoked")
					return
				}
			}

			scopes, legacy := extractScopes(token)
			if legacy {
				log.Printf("auth: legacy token (sub=%s) granted default scopes — refresh to upgrade", sub)
				scopes = domain.DefaultUserScopes()
			}
			_ = legacyCutoff // honoured by a follow-up TDD cycle (issue #9)

			ctx := context.WithValue(r.Context(), UserIDKey, sub)
			ctx = context.WithValue(ctx, scopesKey, scopes)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func GetUserID(ctx context.Context) string {
	id, _ := ctx.Value(UserIDKey).(string)
	return id
}

// GetScopes returns the scopes admitted into the request context by the
// Auth middleware. Returns an empty slice when the token carried no
// scope claim (legacy issuance) — RequireScope handles those tokens
// uniformly by their absence.
func GetScopes(ctx context.Context) []domain.Scope {
	if v, ok := ctx.Value(scopesKey).([]domain.Scope); ok {
		return v
	}
	return nil
}

// WithScopes returns a context that carries the given scopes — symmetric
// to GetScopes. Production callers do not need this; the Auth middleware
// is the only normal writer. It exists so handler tests can stand up a
// scoped request context without spinning the full Auth chain (which
// would require a signing key, a JWT, and a parser per test).
func WithScopes(ctx context.Context, scopes []domain.Scope) context.Context {
	return context.WithValue(ctx, scopesKey, scopes)
}

// extractScopes pulls the "scope" claim from a parsed JWT and converts
// each entry to a typed domain.Scope. Unknown / malformed entries are
// silently dropped — the canonical authority on which scopes exist is
// the Scope constant set in auth/domain, not whatever the token carries.
//
// Returns (scopes, legacy). legacy=true means the token carried no scope
// claim — the caller (Auth) decides what to do with that based on its
// migration-cutoff configuration.
func extractScopes(token *jwt.Token) ([]domain.Scope, bool) {
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, false
	}
	raw, ok := claims["scope"].([]interface{})
	if !ok {
		return nil, true
	}
	out := make([]domain.Scope, 0, len(raw))
	for _, item := range raw {
		s, ok := item.(string)
		if !ok {
			continue
		}
		out = append(out, domain.Scope(s))
	}
	return out, false
}
