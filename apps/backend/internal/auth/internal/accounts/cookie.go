package accounts

import (
	"fmt"
	"net/http"
	"time"
)

const CookieName = "cp_sid"

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

func ExpiredSessionCookie() string {
	return ExpiredCookie(CookieName)
}

func Cookie(name string, value string, expiresAt, now time.Time) string {
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

func ExpiredCookie(name string) string {
	return fmt.Sprint(&http.Cookie{
		Name:     name,
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})
}
