package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// SessionTTL mirrors the 7-day expiry in protocol signSession + session maxAge.
const SessionTTL = 7 * 24 * time.Hour

// ErrInvalidSession covers every rejection verifySession maps to null:
// bad signature, wrong alg, expired, malformed, or missing sub/email.
var ErrInvalidSession = errors.New("auth: invalid session")

type sessionClaims struct {
	jwt.RegisteredClaims
	Email string `json:"email"`
}

// SignSession mints the session JWT: HS256, sub=user id, email claim, iat,
// 7-day expiry, keyed by the raw UTF-8 JWT_SECRET — byte-for-byte the jose
// signSession contract. An empty secret is a hard error (mirrors secretKey()).
func SignSession(user SessionUser, secret string) (string, error) {
	if secret == "" {
		return "", errors.New("auth: JWT_SECRET is not set")
	}
	now := time.Now()
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, sessionClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   user.ID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(SessionTTL)),
		},
		Email: user.Email,
	})
	return tok.SignedString([]byte(secret))
}

// VerifySession checks the token and returns the identity. Any failure —
// signature, algorithm, expiry, shape — is ErrInvalidSession (null upstream).
func VerifySession(token, secret string) (*SessionUser, error) {
	if secret == "" {
		return nil, errors.New("auth: JWT_SECRET is not set")
	}
	parsed, err := jwt.ParseWithClaims(token, &sessionClaims{}, func(t *jwt.Token) (any, error) {
		// Reject alg confusion outright: only HS256 is ever minted.
		if t.Method.Alg() != jwt.SigningMethodHS256.Alg() {
			return nil, ErrInvalidSession
		}
		return []byte(secret), nil
	})
	if err != nil {
		return nil, ErrInvalidSession
	}
	claims, ok := parsed.Claims.(*sessionClaims)
	if !ok || !parsed.Valid || claims.Subject == "" || claims.Email == "" {
		return nil, ErrInvalidSession
	}
	return &SessionUser{ID: claims.Subject, Email: claims.Email}, nil
}
