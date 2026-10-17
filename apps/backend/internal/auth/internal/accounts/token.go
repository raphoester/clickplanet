package accounts

import (
	"crypto/sha256"
	"fmt"
	"net/http"
	"time"
)

type Token struct {
	value string
	hash  TokenHash
}

func TokenOf(value string) *Token {
	sum := sha256.Sum256([]byte(value))
	return &Token{value: value, hash: sum[:]}
}

func (t *Token) Hash() TokenHash {
	return t.hash
}

// Strict: the browser sends it only on a call from the site itself, never on one another site starts.
func (t *Token) StrictCookie(name string, expiresAt, now time.Time) string {
	return fmt.Sprint(&http.Cookie{
		Name:     name,
		Value:    t.value,
		Path:     "/",
		Expires:  expiresAt.UTC(),
		MaxAge:   int(expiresAt.Sub(now).Seconds()),
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
	})
}

func ExpiredStrictCookie(name string) string {
	return fmt.Sprint(&http.Cookie{
		Name:     name,
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
	})
}
