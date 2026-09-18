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

func record(text string, at time.Time) messages.Record {
	return messages.Record{
		Message: messages.Message{
			ID:         messages.MessageID("id-" + text),
			SentAt:     at,
			AuthorName: "Bob",
			AuthorTag:  "a1b2c3",
			CountryID:  "fr",
			Text:       text,
		},
		AuthorID:  "some-uuid",
		IP:        "203.0.113.7",
		UserAgent: "test-agent",
	}
}

func (s *testSuite) TestTheSenderIsStored() {
	s.Require().NoError(s.store.Append(context.Background(), record("hello", start)))

	var ip, userAgent, authorID string
	s.Require().NoError(s.db.QueryRowContext(context.Background(),
		`SELECT ip, user_agent, author_id FROM messages`).Scan(&ip, &userAgent, &authorID))

	s.Equal([]string{"203.0.113.7", "test-agent", "some-uuid"}, []string{ip, userAgent, authorID})
}

func (s *testSuite) TestTheBackfillPrefixesOnlyTheMessagesFromBeforeUsernames() {
	ctx := context.Background()
	usernames := time.Date(2026, 9, 17, 14, 31, 30, 0, time.UTC)

	guest := record("guest", usernames.Add(-time.Minute))
	player := record("player", usernames.Add(time.Minute))
	player.Message.AuthorName = "Bob_the_player"
	s.Require().NoError(s.store.Append(ctx, guest))
	s.Require().NoError(s.store.Append(ctx, player))

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
	recent, err := s.store.Recent(context.Background(), start, 10)
	s.Require().NoError(err)

	names := make([]string, 0, len(recent))
	for _, message := range recent {
		names = append(names, message.AuthorName)
	}
	return names
}
