package postgres_message_store_test

import (
	"context"
	"fmt"
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
	suite.Suite
	db    *cppg.Postgres
	store *postgres_message_store.Store
}

func (s *testSuite) SetupSuite() {
	s.db = cppg.StartTestServer(s.T()).OpenSchema(s.T(), "chat", migrations.FS)
	s.store = postgres_message_store.New(s.db)
}

func (s *testSuite) SetupTest() {
	s.Require().NoError(s.db.Purge(s.T().Context()))
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

func (s *testSuite) recent(since time.Time, limit int) []messages.Message {
	recent, err := s.store.Recent(context.Background(), since, limit)
	s.Require().NoError(err)
	return recent
}

func (s *testSuite) TestAnEmptyTableHasNoRecentMessages() {
	s.Empty(s.recent(start, 10))
}

func (s *testSuite) TestInsertThenRecent() {
	ctx := context.Background()
	s.Require().NoError(s.store.Insert(ctx, record("hello", start)))
	s.Require().NoError(s.store.Insert(ctx, record("planet", start.Add(time.Second))))

	s.Equal([]messages.Message{record("hello", start).Message, record("planet", start.Add(time.Second)).Message},
		s.recent(start, 10))
}

func (s *testSuite) TestAnAdminsMessageReadsBackAsOne() {
	admin := record("hello", start)
	admin.Message.AuthorAdmin = true
	s.Require().NoError(s.store.Insert(context.Background(), admin))
	s.Require().NoError(s.store.Insert(context.Background(), record("planet", start.Add(time.Second))))

	s.Equal([]messages.Message{admin.Message, record("planet", start.Add(time.Second)).Message}, s.recent(start, 10))
}

func (s *testSuite) TestTheSenderIsStored() {
	s.Require().NoError(s.store.Insert(context.Background(), record("hello", start)))

	var ip, userAgent, authorID string
	s.Require().NoError(s.db.QueryRowContext(context.Background(),
		`SELECT ip, user_agent, author_id FROM messages`).Scan(&ip, &userAgent, &authorID))

	s.Equal([]string{"203.0.113.7", "test-agent", "some-uuid"}, []string{ip, userAgent, authorID})
}

func (s *testSuite) TestRecentIsTheNewestWithinTheWindowOldestFirst() {
	ctx := context.Background()
	for i := range 5 {
		s.Require().NoError(s.store.Insert(ctx, record(fmt.Sprintf("msg-%d", i), start.Add(time.Duration(i)*time.Hour))))
	}

	s.Equal([]string{"msg-3", "msg-4"}, texts(s.recent(start, 2)))
	s.Equal([]string{"msg-2", "msg-3", "msg-4"}, texts(s.recent(start.Add(2*time.Hour), 10)))
}

func (s *testSuite) TestRecentKeepsTheOrderMessagesWereRecordedIn() {
	ctx := context.Background()
	s.Require().NoError(s.store.Insert(ctx, record("first", start.Add(time.Second))))
	s.Require().NoError(s.store.Insert(ctx, record("second", start)))

	s.Equal([]string{"first", "second"}, texts(s.recent(start, 10)))
}

func (s *testSuite) TestDeleteBeforeRemovesOnlyOlderMessages() {
	ctx := context.Background()
	s.Require().NoError(s.store.Insert(ctx, record("ancient", start)))
	s.Require().NoError(s.store.Insert(ctx, record("recent", start.Add(48*time.Hour))))

	deleted, err := s.store.DeleteBefore(ctx, start.Add(24*time.Hour))
	s.Require().NoError(err)

	s.Equal(int64(1), deleted)
	s.Equal([]string{"recent"}, texts(s.recent(start, 10)))
}

func (s *testSuite) TestTheBackfillPrefixesOnlyTheMessagesFromBeforeUsernames() {
	ctx := context.Background()
	usernames := time.Date(2026, 9, 17, 14, 31, 30, 0, time.UTC)

	guest := record("guest", usernames.Add(-time.Minute))
	player := record("player", usernames.Add(time.Minute))
	player.Message.AuthorName = "Bob_the_player"
	s.Require().NoError(s.store.Insert(ctx, guest))
	s.Require().NoError(s.store.Insert(ctx, player))

	s.runMigration("20260917200000_guest_prefix_backfill.up.sql")
	s.Equal([]string{"guest_Bob", "Bob_the_player"}, names(s.recent(start, 10)))

	s.runMigration("20260917200000_guest_prefix_backfill.down.sql")
	s.Equal([]string{"Bob", "Bob_the_player"}, names(s.recent(start, 10)))
}

func (s *testSuite) runMigration(name string) {
	query, err := migrations.FS.ReadFile(name)
	s.Require().NoError(err)
	_, err = s.db.ExecContext(context.Background(), string(query))
	s.Require().NoError(err)
}

func names(recent []messages.Message) []string {
	names := make([]string, 0, len(recent))
	for _, message := range recent {
		names = append(names, message.AuthorName)
	}
	return names
}

func texts(recent []messages.Message) []string {
	texts := make([]string, 0, len(recent))
	for _, message := range recent {
		texts = append(texts, message.Text)
	}
	return texts
}
