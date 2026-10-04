package caller_query_test

import (
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts"
)

// The query writes three rules of the domain again: the cookie's name, the token's hash and the expiry.
func (s *testSuite) TestTheQueryNamesTheCallerTheDomainNames() {
	s.Require().NoError(s.store.CreateGuest(s.T().Context(),
		accounts.GuestSession(accounts.AccountID{15: 2}, accounts.TokenOf("token-2"), lifetime, start.Add(time.Hour))))

	for _, at := range []time.Duration{0, 89 * 24 * time.Hour, 90 * 24 * time.Hour, 90*24*time.Hour + time.Hour, 91 * 24 * time.Hour} {
		now := start.Add(at)
		for _, header := range []string{
			"cp_sid=token-1", "cp_sid=token-2", "a=b; cp_sid=token-2", "cp_sid=made-up", "cp_sid=", "", "theme=dark", "cp_sid",
		} {
			want := ""
			session, err := accounts.Caller(s.T().Context(), s.store, header, now)
			if err == nil {
				want = session.Account.String()
			} else {
				s.Require().ErrorIs(err, accounts.ErrNoAccount, "%q at %s", header, now)
			}

			s.Equal(want, s.callerAt(now, header), "%q at %s", header, now)
		}
	}
}
