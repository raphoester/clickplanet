package sessionheader

import (
	"net/http"
	"regexp"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpconnect"
)

// A page framed on another site keeps no cookie of the API's (Safari drops even a partitioned one): it sends this.
const Name = cpconnect.AccountSessionHeader

const None = "none"

var token = regexp.MustCompile(`^[A-Za-z0-9_-]{1,128}$`)

func Carried(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		carried := r.Header.Get(Name)
		if carried == "" {
			next.ServeHTTP(w, r)
			return
		}

		r = r.Clone(r.Context())
		r.Header.Del("Cookie")
		if carried != None && token.MatchString(carried) {
			r.Header.Set("Cookie", accounts.CookieName+"="+carried)
		}
		next.ServeHTTP(&carrier{ResponseWriter: w}, r)
	})
}

type carrier struct {
	http.ResponseWriter
	wrote bool
}

func (c *carrier) WriteHeader(status int) {
	if !c.wrote {
		c.wrote = true
		carry(c.Header())
	}
	c.ResponseWriter.WriteHeader(status)
}

func (c *carrier) Write(body []byte) (int, error) {
	if !c.wrote {
		c.WriteHeader(http.StatusOK)
	}
	return c.ResponseWriter.Write(body) //nolint:wrapcheck // the writer's own error.
}

func (c *carrier) Unwrap() http.ResponseWriter {
	return c.ResponseWriter
}

func carry(header http.Header) {
	var kept []string
	for _, line := range header.Values("Set-Cookie") {
		cookie, err := http.ParseSetCookie(line)
		if err != nil || cookie.Name != accounts.CookieName {
			kept = append(kept, line)
			continue
		}
		if cookie.Value == "" || cookie.MaxAge < 0 {
			header.Set(Name, None)
		} else {
			header.Set(Name, cookie.Value)
		}
	}
	header.Del("Set-Cookie")
	for _, line := range kept {
		header.Add("Set-Cookie", line)
	}
}
