package middleware

import "net/http"

// SecurityHeaders sets a conservative set of security response headers suitable
// for a JSON/SSE API. It does not set Strict-Transport-Security: HSTS must only
// be sent over HTTPS and is the responsibility of the TLS-terminating reverse
// proxy in front of the API (see the production deploy docs).
//
//   - X-Content-Type-Options: nosniff  — disable MIME sniffing.
//   - X-Frame-Options: DENY            — disallow framing (clickjacking).
//   - Referrer-Policy: no-referrer     — never leak the URL as a referrer.
//   - Content-Security-Policy          — the API serves no HTML and loads no
//     resources, so the most restrictive policy applies.
func SecurityHeaders() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h := w.Header()
			h.Set("X-Content-Type-Options", "nosniff")
			h.Set("X-Frame-Options", "DENY")
			h.Set("Referrer-Policy", "no-referrer")
			h.Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'")
			next.ServeHTTP(w, r)
		})
	}
}
