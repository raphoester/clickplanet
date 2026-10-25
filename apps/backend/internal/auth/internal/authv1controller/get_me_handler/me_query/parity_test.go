package me_query_test

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	authv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/authv1controller/authprovider"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/authv1controller/get_me_handler/me_query"
)

func (s *testSuite) sentBack(session *accounts.Session, token string) string {
	cookie, err := http.ParseSetCookie(session.Cookie(accounts.TokenOf(token), start))
	s.Require().NoError(err)
	return cookie.Name + "=" + cookie.Value
}

func (s *testSuite) kept(token string) *accounts.Session {
	session, err := s.store.Session(s.T().Context(), accounts.TokenOf(token).Hash())
	s.Require().NoError(err)
	return session
}

func (s *testSuite) callersAccount(cookieHeader string, at time.Time) (*accounts.Session, *accounts.Account, error) {
	session, err := accounts.Caller(s.T().Context(), s.store, cookieHeader, at)
	if err != nil {
		return nil, nil, err //nolint:wrapcheck // the test reads the sentinel.
	}
	account, err := s.store.Account(s.T().Context(), session.Account())
	return session, account, err //nolint:wrapcheck // the test reads the sentinel.
}

func (s *testSuite) TestGetMeFindsTheCallerTheCommandsFind() {
	odd := start.Add(123_456_789 * time.Nanosecond)
	guest := accounts.GuestSession(ada, accounts.TokenOf("token-1"), lifetime, odd)
	s.Require().NoError(s.store.CreateGuest(s.T().Context(), guest))
	bob := accounts.AccountID{15: 2}
	s.guest(bob, "token-2")
	s.link(bob, "google", "g", "token-3", start.Add(time.Hour))
	s.link(bob, "discord", "d", "token-4", start.Add(time.Hour))
	cara := accounts.AccountID{15: 3}
	s.guest(cara, "token-5")
	s.Require().NoError(s.store.DeleteAccount(s.T().Context(), cara))

	headers := []string{
		s.sentBack(guest, "token-1"), "theme=dark; cp_sid=token-2", "cp_sid=token-3", "cp_sid=token-5",
		"", "cp_sid=", "cp_sid=made-up", ";;;=", "cp_sid=token-1; cp_sid=token-2", "cp_sid=; cp_sid=token-2", "CP_SID=token-1",
	}
	expiry, linkedExpiry := s.kept("token-1").ExpiresAt(), s.kept("token-3").ExpiresAt()
	times := []time.Time{
		start, guest.ExpiresAt(), expiry.Add(-time.Nanosecond), expiry, expiry.Add(time.Nanosecond),
		linkedExpiry.Add(-time.Nanosecond), linkedExpiry,
	}

	answered := 0
	for _, header := range headers {
		for _, at := range times {
			name := fmt.Sprintf("%q at %s", header, at.Format(time.RFC3339Nano))
			session, account, err := s.callersAccount(header, at)
			me, queryErr := s.me(header, at)

			if errors.Is(err, accounts.ErrNoAccount) || errors.Is(err, accounts.ErrAccountNotFound) {
				s.Require().ErrorIs(queryErr, me_query.ErrNoAccount, name)
				continue
			}
			s.Require().NoError(err, name)
			s.Require().NoError(queryErr, name)
			answered++

			var providers []authv1.Provider
			for _, provider := range account.Providers() {
				providers = append(providers, authprovider.ProtoOf(provider))
			}
			s.Equal(account.ID().String(), me.GetAccountId(), name)
			s.Equal(providers, me.GetProviders(), name)
			s.Equal(session.Linked(), me.GetKind() == authv1.AccountKind_ACCOUNT_KIND_LINKED, name)
		}
	}
	s.Positive(answered, "some callers have an account")
}

func (s *testSuite) TestEveryProviderTheProtoNamesIsAnsweredAsTheOneStartSignInNamed() {
	for value := range authv1.Provider_name {
		provider := authv1.Provider(value)
		if provider == authv1.Provider_PROVIDER_UNSPECIFIED {
			continue
		}
		s.Require().NoError(s.db.Purge(s.T().Context()))
		s.guest(ada, "token-1")
		s.link(ada, authprovider.NameOf(provider), "user", "token-2", start)

		me, err := s.me("cp_sid=token-1", start)

		s.Require().NoError(err, provider)
		s.Equal([]authv1.Provider{provider}, me.GetProviders(), provider)
		s.Equal(authv1.AccountKind_ACCOUNT_KIND_LINKED, me.GetKind(), provider)
	}
}
