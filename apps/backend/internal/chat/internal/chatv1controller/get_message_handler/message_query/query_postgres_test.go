package message_query_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	chatv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/chat/v1"
	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/chatv1controller/get_message_handler/message_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages/postgres_message_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/removals"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/removals/postgres_removal_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

func TestRunSuite(t *testing.T) {
	suite.Run(t, new(testSuite))
}

const retention = 24 * time.Hour

var (
	now     = time.Date(2026, 10, 18, 12, 0, 0, 0, time.UTC)
	ada     = messages.AccountID{15: 1}
	bob     = messages.AccountID{15: 2}
	deleted = messages.AccountID{15: 3}
)

type stubAuthors struct {
	err   error
	asked *[]cpsession.AccountID
}

func (a stubAuthors) Authors(
	_ context.Context, accounts []cpsession.AccountID,
) (map[cpsession.AccountID]*playerv1.Author, error) {
	if a.asked != nil {
		*a.asked = append(*a.asked, accounts...)
	}
	if a.err != nil {
		return nil, a.err
	}
	known := map[cpsession.AccountID]*playerv1.Author{
		ada: {AccountId: ada.String(), Name: "Ada"},
		bob: {AccountId: bob.String(), Name: "Bob"},
	}
	found := map[cpsession.AccountID]*playerv1.Author{}
	for _, account := range accounts {
		if author, ok := known[account]; ok {
			found[account] = author
		}
	}
	return found, nil
}

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

// A message a minute, from Ada and Bob in turn, the last one a minute ago: "1" to "<count>".
func (s *testSuite) conversation(count int) {
	for i := 1; i <= count; i++ {
		account := ada
		if i%2 == 0 {
			account = bob
		}
		s.sent(messages.Snapshot{
			ID: messages.MessageID(fmt.Sprint(i)), SentAt: now.Add(-time.Duration(count-i+1) * time.Minute), Account: account,
			Country: "fr", Text: fmt.Sprintf("message %d", i),
		})
	}
}

func (s *testSuite) removed(id string, at time.Time) {
	s.Require().NoError(postgres_removal_store.New(s.db).Save(s.T().Context(),
		removals.Snapshot{MessageID: messages.MessageID(id), RemovedAt: at}))
}

func (s *testSuite) query() *message_query.PostgresQuery {
	return message_query.NewPostgresQuery(s.db, stubAuthors{}, cptime.NewFixedClock(now), retention)
}

func ids(kept []*chatv1.KeptMessage) []string {
	found := make([]string, 0, len(kept))
	for _, message := range kept {
		found = append(found, message.GetMessageId())
	}
	return found
}

func (s *testSuite) TestAMessageIsReadAsItWasSent() {
	s.sent(messages.Snapshot{ID: "insult", SentAt: now.Add(-time.Hour), Account: bob, Country: "fr", Text: "you all suck"})

	message, err := s.query().Message(s.T().Context(), "insult", 0, 0)
	s.Require().NoError(err)

	s.Equal("insult", message.GetMessageId())
	s.Equal(bob.String(), message.GetAccountId())
	s.Equal("Bob", message.GetAuthorName())
	s.Equal("you all suck", message.GetText())
	s.Equal("fr", message.GetCountryId())
	s.Equal(now.Add(-time.Hour), message.GetSentAt().AsTime())
	s.Nil(message.GetRemovedAt(), "a message shown has no removal")
	s.Nil(message.GetReplyTo(), "a message that answers none has no quote")
}

func (s *testSuite) TestARemovedMessageIsStillReadWithWhenItWasRemoved() {
	s.sent(messages.Snapshot{ID: "insult", SentAt: now.Add(-time.Hour), Account: bob, Country: "fr", Text: "you all suck"})
	s.removed("insult", now.Add(-time.Minute))

	message, err := s.query().Message(s.T().Context(), "insult", 0, 0)
	s.Require().NoError(err)

	s.Equal("you all suck", message.GetText(), "the operator reads what was removed")
	s.Equal(now.Add(-time.Minute), message.GetRemovedAt().AsTime())
}

func (s *testSuite) TestAMessageFromBeforeAccountsNamesNoAccount() {
	s.sent(messages.Snapshot{
		ID: "legacy", SentAt: now.Add(-time.Hour), Author: messages.AuthorOf("Bob", false, 0, 0, messages.Title{}),
		Country: "fr", Text: "old",
	})

	message, err := s.query().Message(s.T().Context(), "legacy", 0, 0)
	s.Require().NoError(err)

	s.Empty(message.GetAccountId())
	s.Equal("Bob", message.GetAuthorName(), "the name it was sent under")
}

func (s *testSuite) TestAMessagePastTheRetentionOrNeverSentIsNotFound() {
	s.sent(messages.Snapshot{ID: "old", SentAt: now.Add(-retention - time.Second), Account: bob, Country: "fr", Text: "old"})

	for _, id := range []string{"old", "never"} {
		_, err := s.query().Message(s.T().Context(), id, 10, 10)
		s.Require().ErrorIs(err, message_query.ErrNoMessage, id)
	}
}

func (s *testSuite) TestNothingAroundItUnlessAsked() {
	s.conversation(5)

	message, err := s.query().Message(s.T().Context(), "3", 0, 0)
	s.Require().NoError(err)

	s.Empty(message.GetBefore())
	s.Empty(message.GetAfter())
}

func (s *testSuite) TestTheMessagesJustBeforeAndJustAfterItAreReadOldestFirst() {
	s.conversation(9)

	message, err := s.query().Message(s.T().Context(), "5", 2, 3)
	s.Require().NoError(err)

	s.Equal("5", message.GetMessageId())
	s.Equal([]string{"3", "4"}, ids(message.GetBefore()))
	s.Equal([]string{"6", "7", "8"}, ids(message.GetAfter()))

	four := message.GetBefore()[1]
	s.Equal(bob.String(), four.GetAccountId())
	s.Equal("Bob", four.GetAuthorName())
	s.Equal("message 4", four.GetText())
	s.Equal("fr", four.GetCountryId())
	s.Equal(now.Add(-6*time.Minute), four.GetSentAt().AsTime())
}

func (s *testSuite) TestAroundItIsAsManyAsThereAre() {
	s.conversation(4)

	message, err := s.query().Message(s.T().Context(), "2", 10, 10)
	s.Require().NoError(err)

	s.Equal([]string{"1"}, ids(message.GetBefore()))
	s.Equal([]string{"3", "4"}, ids(message.GetAfter()))
}

func (s *testSuite) TestNothingPastTheRetentionIsReadBeforeIt() {
	s.sent(messages.Snapshot{ID: "old", SentAt: now.Add(-retention - time.Second), Account: ada, Country: "fr", Text: "old"})
	s.sent(messages.Snapshot{ID: "kept", SentAt: now.Add(-time.Hour), Account: ada, Country: "fr", Text: "kept"})
	s.sent(messages.Snapshot{ID: "insult", SentAt: now.Add(-time.Minute), Account: bob, Country: "fr", Text: "you suck"})

	message, err := s.query().Message(s.T().Context(), "insult", 10, 0)
	s.Require().NoError(err)

	s.Equal([]string{"kept"}, ids(message.GetBefore()))
}

func (s *testSuite) TestAroundItIsCapped() {
	s.conversation(2*message_query.MostAround + 3)

	message, err := s.query().Message(s.T().Context(), fmt.Sprint(message_query.MostAround+2), 1000, 1000)
	s.Require().NoError(err)

	s.Len(message.GetBefore(), message_query.MostAround)
	s.Len(message.GetAfter(), message_query.MostAround)
}

func (s *testSuite) TestAMessageRemovedAroundItIsReadAsItWasSent() {
	s.conversation(3)
	s.removed("1", now.Add(-time.Second))

	message, err := s.query().Message(s.T().Context(), "2", 1, 1)
	s.Require().NoError(err)

	s.Equal("message 1", message.GetBefore()[0].GetText())
	s.Equal(now.Add(-time.Second), message.GetBefore()[0].GetRemovedAt().AsTime())
	s.Nil(message.GetAfter()[0].GetRemovedAt())
}

func (s *testSuite) TestEveryAuthorIsNamedAsNowAndAskedForOnce() {
	asked := new([]cpsession.AccountID)
	s.sent(messages.Snapshot{
		ID: "legacy", SentAt: now.Add(-4 * time.Minute), Author: messages.AuthorOf("Old_Name", false, 0, 0, messages.Title{}),
		Country: "fr", Text: "old",
	})
	s.sent(messages.Snapshot{ID: "gone", SentAt: now.Add(-3 * time.Minute), Account: deleted, Country: "fr", Text: "bye"})
	s.sent(messages.Snapshot{ID: "insult", SentAt: now.Add(-2 * time.Minute), Account: bob, Country: "fr", Text: "you suck"})
	s.sent(messages.Snapshot{ID: "again", SentAt: now.Add(-time.Minute), Account: bob, Country: "fr", Text: "again"})

	message, err := message_query.NewPostgresQuery(s.db, stubAuthors{asked: asked}, cptime.NewFixedClock(now), retention).
		Message(s.T().Context(), "insult", 10, 10)
	s.Require().NoError(err)

	s.Equal("Old_Name", message.GetBefore()[0].GetAuthorName())
	s.Equal(message_query.DeletedName, message.GetBefore()[1].GetAuthorName())
	s.Equal(deleted.String(), message.GetBefore()[1].GetAccountId(), "the operator tools still take its id")
	s.Equal("Bob", message.GetAuthorName())
	s.Equal("Bob", message.GetAfter()[0].GetAuthorName())
	s.ElementsMatch([]cpsession.AccountID{deleted, bob}, *asked, "each account once, none for a message before accounts")
}

func (s *testSuite) TestAReplyCarriesWhatItAnswersAsItWasSent() {
	s.sent(messages.Snapshot{ID: "bait", SentAt: now.Add(-time.Hour), Account: ada, Country: "fr", Text: "come at me"})
	s.removed("bait", now.Add(-time.Second))
	s.sent(messages.Snapshot{
		ID: "insult", SentAt: now.Add(-time.Minute), Account: bob, Country: "fr", Text: "you suck",
		Quote: messages.Quote{MessageID: "bait"},
	})

	message, err := s.query().Message(s.T().Context(), "insult", 1, 0)
	s.Require().NoError(err)

	quote := message.GetReplyTo()
	s.Require().NotNil(quote)
	s.Equal("bait", quote.GetMessageId())
	s.Equal("Ada", quote.GetAuthorName())
	s.Equal("come at me", quote.GetText(), "the operator reads what was removed")
	s.Equal(now.Add(-time.Second), quote.GetRemovedAt().AsTime())
	s.Nil(message.GetBefore()[0].GetReplyTo())
}

func (s *testSuite) TestAReplyToAMessagePastTheRetentionHasNoQuote() {
	s.sent(messages.Snapshot{ID: "old", SentAt: now.Add(-retention - time.Second), Account: ada, Country: "fr", Text: "old"})
	s.sent(messages.Snapshot{
		ID: "insult", SentAt: now.Add(-time.Minute), Account: bob, Country: "fr", Text: "you suck",
		Quote: messages.Quote{MessageID: "old"},
	})

	message, err := s.query().Message(s.T().Context(), "insult", 0, 0)
	s.Require().NoError(err)

	s.Nil(message.GetReplyTo())
}

func (s *testSuite) TestNobodyToNameRefusesTheRead() {
	s.conversation(3)

	_, err := message_query.NewPostgresQuery(s.db, stubAuthors{err: assert.AnError}, cptime.NewFixedClock(now), retention).
		Message(s.T().Context(), "2", 1, 1)

	s.Require().ErrorIs(err, assert.AnError)
}
