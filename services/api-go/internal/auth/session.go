package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"time"
)

// SessionCookie mirrors SESSION_COOKIE in protocol ('session').
const SessionCookie = "session"

// SessionMaxAge mirrors the 7-day cookie maxAge in session.ts.
const SessionMaxAge = 7 * 24 * time.Hour

// SetSession writes the session cookie: httpOnly, lax, path /, 7-day maxAge.
// secure mirrors the Node behavior (on in production, off in dev).
func SetSession(w http.ResponseWriter, token string, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookie,
		Value:    token,
		Path:     "/",
		MaxAge:   int(SessionMaxAge.Seconds()),
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
}

// ClearSession deletes the session cookie (mirrors clearSession).
func ClearSession(w http.ResponseWriter, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookie,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
}

// TokenFromRequest extracts the session cookie value, or "" when absent.
func TokenFromRequest(r *http.Request) string {
	c, err := r.Cookie(SessionCookie)
	if err != nil || c.Value == "" {
		return ""
	}
	return c.Value
}

type ctxKey struct{}

// AuthenticateRequest verifies the request's session cookie and returns the user.
// No database lookup — like getSession, the JWT itself is the identity.
func AuthenticateRequest(r *http.Request, secret string) (*SessionUser, error) {
	token := TokenFromRequest(r)
	if token == "" {
		return nil, ErrInvalidSession
	}
	return VerifySession(token, secret)
}

// Middleware enforces auth (401 JSON on missing/invalid session) and stashes
// the user in the request context for handlers.
func Middleware(secret string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, err := AuthenticateRequest(r, secret)
		if err != nil {
			w.Header().Set("content-type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), ctxKey{}, user)))
	})
}

// UserFromContext returns the middleware-stashed user.
func UserFromContext(ctx context.Context) (*SessionUser, bool) {
	u, ok := ctx.Value(ctxKey{}).(*SessionUser)
	return u, ok
}
