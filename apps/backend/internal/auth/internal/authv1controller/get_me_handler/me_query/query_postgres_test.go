package me_query_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"google.golang.org/protobuf/proto"

	authv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts/postgres_account_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/authv1controller/get_me_handler/me_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

func TestRunSuite(t *testing.T) {
	suite.Run(t, new(testSuite))
}

type testSuite struct {
	suite.Suite

	db    *cppg.Postgres
	store *postgres_account_store.Store
}

var (
	start    = time.Date(2026, 9, 15, 12, 0, 0, 0, time.UTC)
	lifetime = accounts.Lifetime{}.WithDefaults()
	ada      = accounts.AccountID{15: 1}
)

func (s *testSuite) SetupSuite() {
	s.db = cppg.StartTestServer(s.T()).OpenSchema(s.T(), "auth", migrations.FS)
	s.store = postgres_account_store.New(s.db)
}

func (s *testSuite) SetupTest() {
	s.Require().NoError(s.db.Purge(s.T().Context()))
}

func (s *testSuite) guest(account accounts.AccountID, token string) *accounts.Session {
	session := accounts.GuestSession(account, accounts.TokenOf(token), lifetime, start)
	s.Require().NoError(s.store.CreateGuest(s.T().Context(), session))
	return session
}

func (s *testSuite) link(account accounts.AccountID, provider string, subject string, token string, at time.Time) {
	identity := accounts.NewIdentity(provider, accounts.ClaimOf(subject, "", false), account, at)
	session := accounts.LinkedSession(account, accounts.TokenOf(token), lifetime, at)
	s.Require().NoError(s.store.SaveSignIn(s.T().Context(), accounts.NewSignIn(session).WithIdentity(identity)))
}

func (s *testSuite) me(cookieHeader string, at time.Time) (*authv1.GetMeResponse, error) {
	return me_query.NewPostgresQuery(s.db, cptime.NewFixedClock(at)).Me(s.T().Context(), cookieHeader) //nolint:wrapcheck // the tests read the sentinel.
}

func (s *testSuite) TestAGuestIsItsAccountWithNoProvider() {
	s.guest(ada, "token-1")

	me, err := s.me("theme=dark; cp_sid=token-1; lang=fr", start)

	s.Require().NoError(err)
	s.True(proto.Equal(&authv1.GetMeResponse{AccountId: ada.String(), Kind: authv1.AccountKind_ACCOUNT_KIND_GUEST}, me), "got %v", me)
}

func (s *testSuite) TestALinkedAccountListsItsProvidersOldestLinkFirstThenByName() {
	s.guest(ada, "token-1")
	s.link(ada, "google", "g", "token-2", start.Add(time.Hour))
	s.link(ada, "email", "ada@example.com", "token-3", start.Add(2*time.Hour))
	s.link(ada, "discord", "d", "token-4", start.Add(time.Hour))

	me, err := s.me("cp_sid=token-1", start)

	s.Require().NoError(err)
	s.True(proto.Equal(&authv1.GetMeResponse{
		AccountId: ada.String(), Kind: authv1.AccountKind_ACCOUNT_KIND_LINKED,
		Providers: []authv1.Provider{authv1.Provider_PROVIDER_DISCORD, authv1.Provider_PROVIDER_GOOGLE, authv1.Provider_PROVIDER_EMAIL},
	}, me), "got %v", me)
}

func (s *testSuite) TestASessionLivesUntilItsExpiryAndNotAtIt() {
	session := s.guest(ada, "token-1")

	_, err := s.me("cp_sid=token-1", session.ExpiresAt().Add(-time.Nanosecond))
	s.Require().NoError(err)

	_, err = s.me("cp_sid=token-1", session.ExpiresAt())
	s.Require().ErrorIs(err, me_query.ErrNoAccount)
}

func (s *testSuite) TestNoLiveSessionIsNoAccount() {
	s.guest(ada, "token-1")

	for name, cookie := range map[string]string{
		"no cookie":    "",
		"other cookie": "theme=dark",
		"empty value":  "cp_sid=",
		"garbage":      ";;;=",
		"unknown":      "cp_sid=made-up",
	} {
		_, err := s.me(cookie, start)

		s.ErrorIs(err, me_query.ErrNoAccount, name)
	}
}

func (s *testSuite) TestADeletedAccountIsNoAccount() {
	s.guest(ada, "token-1")
	s.Require().NoError(s.store.DeleteAccount(s.T().Context(), ada))

	_, err := s.me("cp_sid=token-1", start)

	s.ErrorIs(err, me_query.ErrNoAccount)
}

func (s *testSuite) TestAProviderTheProtoDoesNotNameIsAnError() {
	s.guest(ada, "token-1")
	s.link(ada, "github", "gh", "token-2", start)

	_, err := s.me("cp_sid=token-1", start)

	s.Require().ErrorIs(err, me_query.ErrUnknownProvider)
	s.NotErrorIs(err, me_query.ErrNoAccount)
}

func (s *testSuite) TestAFailedReadIsNotNoAccount() {
	s.guest(ada, "token-1")
	ctx, cancel := context.WithCancel(s.T().Context())
	cancel()

	_, err := me_query.NewPostgresQuery(s.db, cptime.NewFixedClock(start)).Me(ctx, "cp_sid=token-1")

	s.Require().Error(err)
	s.NotErrorIs(err, me_query.ErrNoAccount)
}
