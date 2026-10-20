package messages_query_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	chatv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/chat/v1"
	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/chatv1controller/list_messages_handler/messages_query"
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
			ID: messages.MessageID(fmt.Sprint(i)), SentAt: now.Add(-time.Duration(count-i+1) * time.Minute),
			Account: account, Country: "fr", Text: fmt.Sprintf("message %d", i),
		})
	}
}

func (s *testSuite) removed(id string, at time.Time) {
	s.Require().NoError(postgres_removal_store.New(s.db).Save(s.T().Context(),
		removals.Snapshot{MessageID: messages.MessageID(id), RemovedAt: at}))
}

func (s *testSuite) query() *messages_query.PostgresQuery {
	return messages_query.NewPostgresQuery(s.db, stubAuthors{}, cptime.NewFixedClock(now), retention)
}

func (s *testSuite) page(cursor messages_query.Cursor, limit uint32) []*chatv1.KeptMessage {
	page, err := s.query().Messages(s.T().Context(), cursor, messages.NoAccount, limit)
	s.Require().NoError(err)
	return page.GetMessages()
}

func (s *testSuite) pageOf(author messages.AccountID, cursor messages_query.Cursor, limit uint32) []*chatv1.KeptMessage {
	page, err := s.query().Messages(s.T().Context(), cursor, author, limit)
	s.Require().NoError(err)
	return page.GetMessages()
}

func ids(kept []*chatv1.KeptMessage) []string {
	found := make([]string, 0, len(kept))
	for _, message := range kept {
		found = append(found, message.GetMessageId())
	}
	return found
}

func (s *testSuite) TestWithNoCursorThePageIsTheNewestOldestFirst() {
	s.conversation(5)

	s.Equal([]string{"3", "4", "5"}, ids(s.page(messages_query.Cursor{}, 3)))
}

func (s *testSuite) TestThePageBeforeAMessageIsTheMessagesJustBeforeIt() {
	s.conversation(9)

	page := s.page(messages_query.Cursor{Before: "5"}, 2)

	s.Equal([]string{"3", "4"}, ids(page))
	four := page[1]
	s.Equal(bob.String(), four.GetAccountId())
	s.Equal("Bob", four.GetAuthorName())
	s.Equal("message 4", four.GetText())
	s.Equal("fr", four.GetCountryId())
	s.Equal(now.Add(-6*time.Minute), four.GetSentAt().AsTime())
	s.Nil(four.GetRemovedAt(), "a message shown has no removal")
	s.Nil(four.GetReplyTo(), "a message that answers none has no quote")
}

func (s *testSuite) TestThePageAfterAMessageIsTheMessagesJustAfterIt() {
	s.conversation(9)

	s.Equal([]string{"6", "7", "8"}, ids(s.page(messages_query.Cursor{After: "5"}, 3)))
}

func (s *testSuite) TestAPageIsAsManyAsThereAre() {
	s.conversation(4)

	s.Equal([]string{"1"}, ids(s.page(messages_query.Cursor{Before: "2"}, 10)))
	s.Equal([]string{"4"}, ids(s.page(messages_query.Cursor{After: "3"}, 10)))
	s.Empty(s.page(messages_query.Cursor{After: "4"}, 10), "nothing was sent after the newest")
	s.Empty(s.page(messages_query.Cursor{Before: "1"}, 10), "nothing is kept before the oldest")
}

func (s *testSuite) TestAnAccountsPageHoldsItsMessagesAloneOldestFirst() {
	s.conversation(9)

	s.Equal([]string{"2", "4", "6", "8"}, ids(s.pageOf(bob, messages_query.Cursor{}, 10)))
	s.Equal([]string{"6", "8"}, ids(s.pageOf(bob, messages_query.Cursor{}, 2)), "its newest")
	s.Empty(s.pageOf(deleted, messages_query.Cursor{}, 10), "an account that said nothing")
}

func (s *testSuite) TestAnAccountsCursorsPageThroughItsMessagesAlone() {
	s.conversation(9)

	s.Equal([]string{"4", "6"}, ids(s.pageOf(bob, messages_query.Cursor{Before: "8"}, 2)))
	s.Equal([]string{"4", "6"}, ids(s.pageOf(bob, messages_query.Cursor{After: "2"}, 2)))
	s.Equal([]string{"2", "4"}, ids(s.pageOf(bob, messages_query.Cursor{Before: "5"}, 10)),
		"a cursor may be another author's message")

	var read []string
	page := s.pageOf(ada, messages_query.Cursor{}, 2)
	for len(page) > 0 {
		read = append(ids(page), read...)
		page = s.pageOf(ada, messages_query.Cursor{Before: page[0].GetMessageId()}, 2)
	}
	s.Equal([]string{"1", "3", "5", "7", "9"}, read, "paging back reads each of its messages once")
}

func (s *testSuite) TestPagingBackFromTheNewestReadsEveryMessageOnce() {
	s.conversation(7)

	var read []string
	page := s.page(messages_query.Cursor{}, 3)
	for len(page) > 0 {
		read = append(ids(page), read...)
		page = s.page(messages_query.Cursor{Before: page[0].GetMessageId()}, 3)
	}

	s.Equal([]string{"1", "2", "3", "4", "5", "6", "7"}, read)
}

func (s *testSuite) TestThePageIsFiftyUnlessAskedAndTwoHundredAtMost() {
	s.conversation(messages_query.MaxLimit + 3)

	s.Len(s.page(messages_query.Cursor{}, 0), messages_query.DefaultLimit)
	s.Len(s.page(messages_query.Cursor{}, 1000), messages_query.MaxLimit)
}

func (s *testSuite) TestNothingPastTheRetentionIsRead() {
	s.sent(messages.Snapshot{ID: "old", SentAt: now.Add(-retention - time.Second), Account: ada, Country: "fr", Text: "old"})
	s.sent(messages.Snapshot{ID: "kept", SentAt: now.Add(-time.Hour), Account: ada, Country: "fr", Text: "kept"})
	s.sent(messages.Snapshot{ID: "insult", SentAt: now.Add(-time.Minute), Account: bob, Country: "fr", Text: "you suck"})

	s.Equal([]string{"kept"}, ids(s.page(messages_query.Cursor{Before: "insult"}, 10)))
	s.Equal([]string{"kept", "insult"}, ids(s.page(messages_query.Cursor{}, 10)))
}

func (s *testSuite) TestACursorPastTheRetentionOrNeverSentIsNotFound() {
	s.sent(messages.Snapshot{ID: "old", SentAt: now.Add(-retention - time.Second), Account: bob, Country: "fr", Text: "old"})

	for _, cursor := range []messages_query.Cursor{{Before: "old"}, {After: "old"}, {Before: "never"}, {After: "never"}} {
		_, err := s.query().Messages(s.T().Context(), cursor, messages.NoAccount, 10)
		s.Require().ErrorIs(err, messages_query.ErrNoMessage, cursor)
	}
}

func (s *testSuite) TestTwoCursorsAreRefused() {
	s.conversation(3)

	_, err := s.query().Messages(s.T().Context(), messages_query.Cursor{Before: "3", After: "1"}, messages.NoAccount, 10)

	s.Require().ErrorIs(err, messages_query.ErrTwoCursors)
}

func (s *testSuite) TestARemovedMessageIsReadAsItWasSentWithWhenItWasRemoved() {
	s.conversation(3)
	s.removed("2", now.Add(-time.Second))

	page := s.page(messages_query.Cursor{}, 3)

	s.Equal("message 2", page[1].GetText(), "the operator reads what was removed")
	s.Equal(now.Add(-time.Second), page[1].GetRemovedAt().AsTime())
	s.Nil(page[0].GetRemovedAt())
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

	page, err := messages_query.NewPostgresQuery(s.db, stubAuthors{asked: asked}, cptime.NewFixedClock(now), retention).
		Messages(s.T().Context(), messages_query.Cursor{}, messages.NoAccount, 10)
	s.Require().NoError(err)

	kept := page.GetMessages()
	s.Equal("Old_Name", kept[0].GetAuthorName(), "the name it was sent under")
	s.Empty(kept[0].GetAccountId())
	s.Equal(messages_query.DeletedName, kept[1].GetAuthorName())
	s.Equal(deleted.String(), kept[1].GetAccountId(), "the operator tools still take its id")
	s.Equal("Bob", kept[2].GetAuthorName())
	s.Equal("Bob", kept[3].GetAuthorName())
	s.ElementsMatch([]cpsession.AccountID{deleted, bob}, *asked, "each account once, none for a message before accounts")
}

func (s *testSuite) TestAReplyCarriesWhatItAnswersAsItWasSent() {
	s.sent(messages.Snapshot{ID: "bait", SentAt: now.Add(-time.Hour), Account: ada, Country: "fr", Text: "come at me"})
	s.removed("bait", now.Add(-time.Second))
	s.sent(messages.Snapshot{
		ID: "insult", SentAt: now.Add(-time.Minute), Account: bob, Country: "fr", Text: "you suck",
		Quote: messages.Quote{MessageID: "bait"},
	})

	page := s.page(messages_query.Cursor{After: "bait"}, 10)

	quote := page[0].GetReplyTo()
	s.Require().NotNil(quote)
	s.Equal("bait", quote.GetMessageId())
	s.Equal("Ada", quote.GetAuthorName())
	s.Equal("come at me", quote.GetText(), "the operator reads what was removed")
	s.Equal(now.Add(-time.Second), quote.GetRemovedAt().AsTime())
}

func (s *testSuite) TestAReplyToAMessagePastTheRetentionHasNoQuote() {
	s.sent(messages.Snapshot{ID: "old", SentAt: now.Add(-retention - time.Second), Account: ada, Country: "fr", Text: "old"})
	s.sent(messages.Snapshot{
		ID: "insult", SentAt: now.Add(-time.Minute), Account: bob, Country: "fr", Text: "you suck",
		Quote: messages.Quote{MessageID: "old"},
	})

	s.Nil(s.page(messages_query.Cursor{}, 10)[0].GetReplyTo())
}

func (s *testSuite) TestNobodyToNameRefusesTheRead() {
	s.conversation(3)

	_, err := messages_query.NewPostgresQuery(s.db, stubAuthors{err: assert.AnError}, cptime.NewFixedClock(now), retention).
		Messages(s.T().Context(), messages_query.Cursor{}, messages.NoAccount, 10)

	s.Require().ErrorIs(err, assert.AnError)
}
