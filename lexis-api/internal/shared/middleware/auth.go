package middleware

import (
	"context"
	"log"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"

	"github.com/lexis-app/lexis-api/internal/shared/httputil"
)

type contextKey string

const UserIDKey contextKey = "userID"

// Blacklist checks whether a user's tokens have been globally invalidated.
type Blacklist interface {
	IsBlacklisted(ctx context.Context, key string) (bool, error)
}

func Auth(jwtSecret []byte, blacklist Blacklist) func(http.Handler) http.Handler {
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

			ctx := context.WithValue(r.Context(), UserIDKey, sub)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func GetUserID(ctx context.Context) string {
	id, _ := ctx.Value(UserIDKey).(string)
	return id
}
