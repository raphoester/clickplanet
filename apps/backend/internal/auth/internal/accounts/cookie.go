package accounts

import (
	"fmt"
	"net/http"
	"time"
)

// CookieName is the one cookie this backend sets. No other module reads it.
const CookieName = "cp_sid"

// TokenFromCookies reads the session token out of a browser's Cookie header.
func TokenFromCookies(cookieHeader string) (*Token, error) {
	cookies, err := http.ParseCookie(cookieHeader)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrNoSessionCookie, err)
	}

	for _, cookie := range cookies {
		if cookie.Name == CookieName && cookie.Value != "" {
			return TokenOf(cookie.Value), nil
		}
	}

	return nil, ErrNoSessionCookie
}

// setCookie stores token until expiresAt: HttpOnly so no script reads it, Lax so no other site's form sends it.
func setCookie(token string, expiresAt, now time.Time) string {
	cookie := &http.Cookie{
		Name:     CookieName,
		Value:    token,
		Path:     "/",
		Expires:  expiresAt.UTC(),
		MaxAge:   int(expiresAt.Sub(now).Seconds()),
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	}

	return cookie.String()
}
