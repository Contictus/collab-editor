package auth

import (
	"errors"
	"net/mail"
	"strings"
	"unicode/utf8"
)

// Credential rules mirror credentialsSchema in shared (zod): email format,
// password 8..200 chars. Rune count approximates JS string length.

var (
	// ErrInvalidEmail mirrors the email shape failure.
	ErrInvalidEmail = errors.New("auth: invalid email")
	// ErrWeakPassword mirrors the password length failure.
	ErrWeakPassword = errors.New("auth: password must be 8..200 characters")
)

// ValidateCredentials checks email/password shapes at the edge (handlers run
// this before hashing or DB access, like the Server Actions run zod first).
func ValidateCredentials(email, password string) error {
	addr, err := mail.ParseAddress(email)
	if err != nil || addr.Address != email {
		return ErrInvalidEmail
	}
	domain := addr.Address[strings.LastIndex(addr.Address, "@")+1:]
	if !strings.Contains(domain, ".") {
		return ErrInvalidEmail
	}
	if n := utf8.RuneCountInString(password); n < 8 || n > 200 {
		return ErrWeakPassword
	}
	return nil
}
