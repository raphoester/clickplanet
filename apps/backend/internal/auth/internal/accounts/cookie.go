package accounts

import (
	"fmt"
	"net/http"
	"time"
)

// CookieName is the session cookie. No other module reads it.
const CookieName = "cp_sid"

// TokenFromCookies reads the session token out of a browser's Cookie header.
func TokenFromCookies(cookieHeader string) (*Token, error) {
	value, found := CookieValue(cookieHeader, CookieName)
	if !found {
		return nil, ErrNoSessionCookie
	}
	return TokenOf(value), nil
}

func CookieValue(cookieHeader string, name string) (string, bool) {
	cookies, err := http.ParseCookie(cookieHeader)
	if err != nil {
		return "", false
	}

	for _, cookie := range cookies {
		if cookie.Name == name && cookie.Value != "" {
			return cookie.Value, true
		}
	}
	return "", false
}

func ClearCookie() string {
	return ExpireCookie(CookieName)
}

// SetCookie stores value until expiresAt: HttpOnly so no script reads it, Lax so no other site's form sends it.
func SetCookie(name string, value string, expiresAt, now time.Time) string {
	return fmt.Sprint(&http.Cookie{
		Name:     name,
		Value:    value,
		Path:     "/",
		Expires:  expiresAt.UTC(),
		MaxAge:   int(expiresAt.Sub(now).Seconds()),
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})
}

func ExpireCookie(name string) string {
	return fmt.Sprint(&http.Cookie{
		Name:     name,
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})
}
