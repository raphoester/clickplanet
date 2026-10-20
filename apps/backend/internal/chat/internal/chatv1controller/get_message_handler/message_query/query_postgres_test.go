package message_query_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/chatv1controller/get_message_handler/message_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages/postgres_message_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/removals"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/removals/postgres_removal_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

func TestRunSuite(t *testing.T) {
	suite.Run(t, new(testSuite))
}

const retention = 24 * time.Hour

var (
	now = time.Date(2026, 10, 18, 12, 0, 0, 0, time.UTC)
	bob = messages.AccountID{15: 2}
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

func (s *testSuite) sent(snapshot messages.Snapshot) {
	s.Require().NoError(postgres_message_store.New(s.db).Append(s.T().Context(),
		messages.NewRecord(messages.Restored(s.T(), snapshot), "browser-1", "203.0.113.7", "test-agent")))
}

func (s *testSuite) query() *message_query.PostgresQuery {
	return message_query.NewPostgresQuery(s.db, cptime.NewFixedClock(now), retention)
}

func (s *testSuite) TestAMessageIsReadAsItWasSent() {
	s.sent(messages.Snapshot{ID: "insult", SentAt: now.Add(-time.Hour), Account: bob, Country: "fr", Text: "you all suck"})

	message, err := s.query().Message(s.T().Context(), "insult")
	s.Require().NoError(err)

	s.Equal("insult", message.GetMessageId())
	s.Equal(bob.String(), message.GetAccountId())
	s.Equal("you all suck", message.GetText())
	s.Equal("fr", message.GetCountryId())
	s.Equal(now.Add(-time.Hour), message.GetSentAt().AsTime())
	s.Nil(message.GetRemovedAt(), "a message shown has no removal")
}

func (s *testSuite) TestARemovedMessageIsStillReadWithWhenItWasRemoved() {
	s.sent(messages.Snapshot{ID: "insult", SentAt: now.Add(-time.Hour), Account: bob, Country: "fr", Text: "you all suck"})
	s.Require().NoError(postgres_removal_store.New(s.db).Save(s.T().Context(),
		removals.Snapshot{MessageID: "insult", RemovedAt: now.Add(-time.Minute)}))

	message, err := s.query().Message(s.T().Context(), "insult")
	s.Require().NoError(err)

	s.Equal("you all suck", message.GetText(), "the operator reads what was removed")
	s.Equal(now.Add(-time.Minute), message.GetRemovedAt().AsTime())
}

func (s *testSuite) TestAMessageFromBeforeAccountsNamesNoAccount() {
	s.sent(messages.Snapshot{
		ID: "legacy", SentAt: now.Add(-time.Hour), Author: messages.AuthorOf("Bob", false, 0, 0, messages.Title{}),
		Country: "fr", Text: "old",
	})

	message, err := s.query().Message(s.T().Context(), "legacy")
	s.Require().NoError(err)

	s.Empty(message.GetAccountId())
}

func (s *testSuite) TestAMessagePastTheRetentionOrNeverSentIsNotFound() {
	s.sent(messages.Snapshot{ID: "old", SentAt: now.Add(-retention - time.Second), Account: bob, Country: "fr", Text: "old"})

	for _, id := range []string{"old", "never"} {
		_, err := s.query().Message(s.T().Context(), id)
		s.Require().ErrorIs(err, message_query.ErrNoMessage, id)
	}
}
