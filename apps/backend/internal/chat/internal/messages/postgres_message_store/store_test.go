package postgres_message_store_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages/postgres_message_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func TestRunSuite(t *testing.T) {
	suite.Run(t, new(testSuite))
}

type testSuite struct {
	messages.StorageContractSuite

	db    *cppg.Postgres
	store *postgres_message_store.Store
}

func (s *testSuite) SetupSuite() {
	s.db = cppg.StartTestServer(s.T()).OpenSchema(s.T(), "chat", migrations.FS)
	s.store = postgres_message_store.New(s.db)
	s.NewStorage = func() messages.Storage { return s.store }
}

func (s *testSuite) SetupTest() {
	s.Require().NoError(s.db.Purge(s.T().Context()))
	s.StorageContractSuite.SetupTest()
}

var start = time.Date(2024, 1, 1, 12, 0, 0, 123_456_000, time.UTC)

func (s *testSuite) sent(text string, name string, at time.Time) messages.Message {
	return messages.Restored(s.T(), messages.Snapshot{
		ID: messages.MessageID("id-" + text), SentAt: at, Author: messages.AuthorOf(name, false, 0, 0, messages.Title{}),
		Country: "fr", Text: text,
	})
}

func (s *testSuite) keep(message messages.Message) {
	s.Require().NoError(s.store.Append(context.Background(), messages.NewRecord(message, "some-uuid", "203.0.113.7", "test-agent")))
}

func (s *testSuite) TestTheSenderIsStored() {
	s.keep(s.sent("hello", "Bob", start))

	var ip, userAgent, authorID string
	s.Require().NoError(s.db.QueryRowContext(context.Background(),
		`SELECT ip, user_agent, author_id FROM messages`).Scan(&ip, &userAgent, &authorID))

	s.Equal([]string{"203.0.113.7", "test-agent", "some-uuid"}, []string{ip, userAgent, authorID})
}

func (s *testSuite) TestOnlyAReplyKeepsTheMessageItRepliesTo() {
	ctx := context.Background()
	question := s.sent("hello", "Bob", start)
	s.keep(question)
	s.keep(s.sent("reply", "Ada", start.Add(time.Minute)).Answering(question))

	rows, err := s.db.QueryContext(ctx, `SELECT COALESCE(reply_to, 'none') FROM messages ORDER BY seq`)
	s.Require().NoError(err)
	defer func() { _ = rows.Close() }()

	var kept []string
	for rows.Next() {
		var replyTo string
		s.Require().NoError(rows.Scan(&replyTo))
		kept = append(kept, replyTo)
	}
	s.Require().NoError(rows.Err())
	s.Equal([]string{"none", "id-hello"}, kept, "the id alone: what it said is read when the reply is shown")
}

func (s *testSuite) TestTheBackfillPrefixesOnlyTheMessagesFromBeforeUsernames() {
	usernames := time.Date(2026, 9, 17, 14, 31, 30, 0, time.UTC)

	s.keep(s.sent("guest", "Bob", usernames.Add(-time.Minute)))
	s.keep(s.sent("player", "Bob_the_player", usernames.Add(time.Minute)))

	s.runMigration("20260917200000_guest_prefix_backfill.up.sql")
	s.Equal([]string{"guest_Bob", "Bob_the_player"}, s.names())

	s.runMigration("20260917200000_guest_prefix_backfill.down.sql")
	s.Equal([]string{"Bob", "Bob_the_player"}, s.names())
}

func (s *testSuite) runMigration(name string) {
	query, err := migrations.FS.ReadFile(name)
	s.Require().NoError(err)
	_, err = s.db.ExecContext(context.Background(), string(query))
	s.Require().NoError(err)
}

func (s *testSuite) names() []string {
	rows, err := s.db.QueryContext(context.Background(), `SELECT name FROM messages ORDER BY seq`)
	s.Require().NoError(err)
	defer func() { _ = rows.Close() }()

	var names []string
	for rows.Next() {
		var name string
		s.Require().NoError(rows.Scan(&name))
		names = append(names, name)
	}
	s.Require().NoError(rows.Err())
	return names
}
