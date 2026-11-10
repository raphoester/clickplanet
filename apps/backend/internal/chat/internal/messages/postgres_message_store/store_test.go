package postgres_message_store_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages/postgres_message_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/removals"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/removals/postgres_removal_store"
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

func (s *testSuite) sent(text string, at time.Time) messages.Message {
	return messages.Restored(s.T(), messages.Snapshot{ID: messages.MessageID("id-" + text), SentAt: at, Country: "fr", Text: text})
}

func (s *testSuite) keep(message messages.Message) {
	s.Require().NoError(s.store.Append(context.Background(), messages.NewRecord(message, "203.0.113.7", "test-agent")))
}

func (s *testSuite) TestTheSenderIsStored() {
	s.keep(s.sent("hello", start))

	var ip, userAgent string
	s.Require().NoError(s.db.On(context.Background()).QueryRowContext(context.Background(),
		`SELECT ip, user_agent FROM messages`).Scan(&ip, &userAgent))

	s.Equal([]string{"203.0.113.7", "test-agent"}, []string{ip, userAgent})
}

func (s *testSuite) TestOnlyAReplyKeepsTheMessageItRepliesTo() {
	ctx := context.Background()
	question := s.sent("hello", start)
	s.keep(question)
	reply, err := s.sent("reply", start.Add(time.Minute)).Answering(question)
	s.Require().NoError(err)
	s.keep(reply)

	rows, err := s.db.On(ctx).QueryContext(ctx, `SELECT COALESCE(reply_to, 'none') FROM messages ORDER BY seq`)
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

func (s *testSuite) removed(message messages.Message) {
	snapshot := message.TakeSnapshot()
	s.Require().NoError(postgres_removal_store.New(s.db).Save(context.Background(),
		removals.Snapshot{MessageID: snapshot.ID, RemovedAt: snapshot.SentAt.Add(time.Minute)}))
}

func (s *testSuite) TestARemovedMessageIsNeitherShownNorReadAndIsStillKept() {
	ctx := context.Background()
	insult := s.sent("insult", start)
	s.keep(insult)
	s.keep(s.sent("hello", start.Add(time.Second)))
	s.removed(insult)

	_, err := s.store.Message(ctx, "id-insult", start, 10)
	s.Require().ErrorIs(err, messages.ErrNotShown, "nobody answers or reacts to a removed message")

	kept, err := s.store.KeptMessage(ctx, "id-insult", start)
	s.Require().NoError(err, "the row stays, as evidence")
	s.Equal(insult.TakeSnapshot().Text, kept.TakeSnapshot().Text)
	s.Equal(insult.TakeSnapshot().Account, kept.TakeSnapshot().Account)
}

func (s *testSuite) TestAKeptGifIsReadWithItsAddress() {
	gif := messages.Gif{URL: "https://static.klipy.com/ii/935d7ab9/14/af/SE72470w.webp", Width: 220, Height: 124}
	sent := messages.Restored(s.T(), messages.Snapshot{
		ID: "id-gif", SentAt: start, Account: messages.AccountID{15: 1}, Room: messages.World, Country: "fr", Gif: gif,
	})
	s.keep(sent)

	kept, err := s.store.KeptMessage(context.Background(), "id-gif", start)
	s.Require().NoError(err)
	s.Equal(gif, kept.TakeSnapshot().Gif)
	s.Empty(kept.TakeSnapshot().Text)
}

func (s *testSuite) TestARemovedMessageKeepsItsPlaceInTheWindow() {
	older := s.sent("older", start)
	insult := s.sent("insult", start.Add(time.Second))
	s.keep(older)
	s.keep(insult)
	s.removed(insult)

	_, err := s.store.Message(context.Background(), "id-older", start, 1)
	s.Require().ErrorIs(err, messages.ErrNotShown, "a window of one holds the removed message, as the history does")
}

func (s *testSuite) TestAMessageIsKeptOnlyInsideTheRetention() {
	ctx := context.Background()
	s.keep(s.sent("old", start))

	for id, since := range map[messages.MessageID]time.Time{
		"id-old":     start.Add(time.Second),
		"id-unknown": start,
	} {
		_, err := s.store.KeptMessage(ctx, id, since)
		s.Require().ErrorIs(err, messages.ErrNotKept, id)
	}
}
