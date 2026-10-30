package mute_query_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/chatv1controller/get_mute_handler/mute_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages/postgres_message_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/mutes"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/mutes/postgres_mute_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

func TestRunSuite(t *testing.T) {
	suite.Run(t, new(testSuite))
}

var (
	now      = time.Date(2026, 10, 30, 20, 0, 0, 0, time.UTC)
	bully    = messages.AccountID{15: 1}
	newGuest = messages.AccountID{15: 2}
	home     = mutes.Scope("2a01:e0a:1:2::/64")
)

type testSuite struct {
	suite.Suite

	db *cppg.Postgres
}

func (s *testSuite) SetupSuite() {
	s.db = cppg.StartTestServer(s.T()).OpenSchema(s.T(), "chat", migrations.FS)
}

func (s *testSuite) SetupTest() {
	s.Require().NoError(s.db.Purge(s.T().Context()))
}

func (s *testSuite) posted(account messages.AccountID, ip string) {
	s.Require().NoError(postgres_message_store.New(s.db).Append(s.T().Context(), messages.NewRecord(
		messages.Restored(s.T(), messages.Snapshot{
			ID: messages.MessageID(account.String() + ip), SentAt: now.Add(-time.Hour), Account: account, Country: "fr", Text: "hi",
		}), "browser-1", ip, "test-agent")))
}

func (s *testSuite) muted(id byte, account messages.AccountID, scope mutes.Scope, duration time.Duration) mutes.Mute {
	mute := mutes.NewMute(mutes.MuteID{15: id}, mutes.CallerOf(account, scope), now.Add(-time.Minute), duration)
	s.Require().NoError(postgres_mute_store.New(s.db).Save(s.T().Context(), mute))
	return mute
}

func (s *testSuite) TestAMuteGivenToTheAccountSaysUntilWhen() {
	mute := s.muted(1, bully, mutes.NoScope, time.Hour)

	answer, err := mute_query.NewPostgresQuery(s.db, cptime.NewFixedClock(now)).Mute(s.T().Context(), bully)
	s.Require().NoError(err)

	s.Equal(mute.Until(), answer.GetMutedUntil().AsTime())
	s.Equal(bully.String(), answer.GetMutedAccountId())
}

func (s *testSuite) TestAMuteOnTheNetworkItLastPostedFromNamesTheAccountItWasGivenTo() {
	s.posted(newGuest, "2a01:e0a:1:2:aaaa::9")
	mute := s.muted(1, bully, home, time.Hour)

	answer, err := mute_query.NewPostgresQuery(s.db, cptime.NewFixedClock(now)).Mute(s.T().Context(), newGuest)
	s.Require().NoError(err)

	s.Equal(mute.Until(), answer.GetMutedUntil().AsTime())
	s.Equal(bully.String(), answer.GetMutedAccountId())
}

func (s *testSuite) TestTheMuteThatEndsLastIsRead() {
	s.posted(bully, "2a01:e0a:1:2::1")
	s.muted(1, bully, home, time.Hour)
	long := s.muted(2, newGuest, home, 24*time.Hour)

	answer, err := mute_query.NewPostgresQuery(s.db, cptime.NewFixedClock(now)).Mute(s.T().Context(), bully)
	s.Require().NoError(err)

	s.Equal(long.Until(), answer.GetMutedUntil().AsTime())
	s.Equal(newGuest.String(), answer.GetMutedAccountId())
}

func (s *testSuite) TestNoMuteThatHoldsTheAccountNowAnswersNothing() {
	s.muted(1, bully, mutes.NoScope, time.Hour)
	s.muted(2, newGuest, mutes.NoScope, 30*time.Second)

	for _, account := range []messages.AccountID{newGuest, {15: 3}} {
		answer, err := mute_query.NewPostgresQuery(s.db, cptime.NewFixedClock(now)).Mute(s.T().Context(), account)
		s.Require().NoError(err)

		s.Nil(answer.GetMutedUntil(), "an ended mute, or another account's with no network, holds nobody")
		s.Empty(answer.GetMutedAccountId())
	}
}
