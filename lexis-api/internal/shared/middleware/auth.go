package middleware

import (
	"context"
	"log/slog"
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
// information attached to the context. The legacyCutoff parameter governs
// migration-window behaviour for tokens that lack a scope claim:
//
//   - legacyCutoff.IsZero() (cutoff disabled) — no-scope tokens receive
//     domain.DefaultUserScopes() and a one-line log so the operator can
//     watch the migration tail clear.
//   - !legacyCutoff.IsZero() (cutoff active) — no-scope tokens are
//     rejected with 401 regardless of iat. Per issue #9 acceptance:
//     iat<cutoff and iat>=cutoff both reject, but the log lines
//     distinguish them so an operator can spot an issuer regression
//     (a token minted after cutoff with no scope claim should not occur).
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
					slog.Error("auth: blacklist check error", "error", err)
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
				if !legacyCutoff.IsZero() {
					rejectLegacy(sub, token.Claims, legacyCutoff)
					httputil.WriteProblem(w, http.StatusUnauthorized, "Unauthorized", "token missing scope claim — please re-authenticate")
					return
				}
				slog.Warn("auth: legacy token granted default scopes — refresh to upgrade", "sub", sub)
				scopes = domain.DefaultUserScopes()
			}

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

// rejectLegacy emits the audit log for a no-scope token rejected by the
// active cutoff. Two log shapes — pre-cutoff issuance is the expected
// tail of the migration; post-cutoff issuance is an issuer-side
// regression (the scope-aware generator should not be producing such
// tokens) and warrants a separate line so it is grep-able.
func rejectLegacy(sub string, claims jwt.Claims, cutoff time.Time) {
	cutoffStr := cutoff.UTC().Format(time.RFC3339)
	iat, err := claims.GetIssuedAt()
	if err != nil || iat == nil {
		slog.Warn("auth: legacy token (iat=missing) rejected — cutoff active", "sub", sub, "cutoff", cutoffStr)
		return
	}
	iatStr := iat.UTC().Format(time.RFC3339)
	if iat.Before(cutoff) {
		slog.Warn("auth: legacy token rejected — cutoff active", "sub", sub, "iat", iatStr, "cutoff", cutoffStr)
		return
	}
	slog.Error("auth: post-cutoff legacy token rejected — issuer regression, no scope claim should be possible", "sub", sub, "iat", iatStr)
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
