// Package auth is the Go auth layer (F2), mirroring the Node contract:
//
//   - password.ts: argon2id PHC (node-rs defaults m=19456,t=2,p=1), verify
//     returns false on malformed input instead of throwing.
//   - protocol (jose): session JWT HS256, sub=user id, email claim, iat,
//     7-day expiry, raw UTF-8 JWT_SECRET. verifySession returns null on any
//     error or when sub/email are not strings.
//   - session.ts: `session` cookie, httpOnly, lax, path /, 7-day maxAge.
//   - shared credentialsSchema: email format, password 8..200 chars.
//
// REST and WS share one identity: the same token the cookie carries is the one
// the WS handshake verifies (F4).
package auth

// SessionUser is the authenticated identity carried in the JWT (shared.SessionUser).
type SessionUser struct {
	ID    string
	Email string
}
