package authread

import (
	"crypto/sha256"
	"net/http"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

const SessionCookie = "cp_sid"

// The sessions table's key for what the browser sent: accounts.TokenOf hashes it the same way for the commands.
func TokenHash(cookieHeader string) ([]byte, bool) {
	cookies, err := http.ParseCookie(cookieHeader)
	if err != nil {
		return nil, false
	}

	for _, cookie := range cookies {
		if cookie.Name == SessionCookie && cookie.Value != "" {
			sum := sha256.Sum256([]byte(cookie.Value))
			return sum[:], true
		}
	}
	return nil, false
}

// Postgres keeps microseconds and rounds what it is sent: truncated, "now < expires_at" stays exact.
func Now(clock cptime.Clock) time.Time {
	return clock.Now().UTC().Truncate(time.Microsecond)
}

// Needs the sessions table as sessions, TokenHash as $1 and Now as $2: Session.ExpiryError for the commands.
const LiveSession = `sessions.token_hash = $1 AND sessions.expires_at > $2`
