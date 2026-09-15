package accounts

import (
	"net/http"
	"time"
)

// CookieName is the one cookie this backend sets. No other module reads it.
const CookieName = "cp_sid"

// TokenFrom reads the session token out of a browser's Cookie header.
func TokenFrom(cookieHeader string) (string, bool) {
	cookies, err := http.ParseCookie(cookieHeader)
	if err != nil {
		return "", false
	}

	for _, cookie := range cookies {
		if cookie.Name == CookieName && cookie.Value != "" {
			return cookie.Value, true
		}
	}

	return "", false
}

// SetCookie is the header that stores token until expiresAt: HttpOnly so no script reads it, Lax so no other site's form sends it.
func SetCookie(token string, expiresAt, now time.Time) string {
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
