package history_query_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
	"google.golang.org/protobuf/proto"

	chatv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/chat/v1"
	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/announcements"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/announcements/postgres_announcement_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/chatv1controller/get_history_handler/history_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages/postgres_message_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/reactions"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/reactions/postgres_reaction_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

func TestRunSuite(t *testing.T) {
	suite.Run(t, new(testSuite))
}

type fakeAuthors struct {
	named map[messages.AccountID]*playerv1.Author
	asked [][]messages.AccountID
	err   error
}

func (f *fakeAuthors) Authors(
	_ context.Context,
	accounts []messages.AccountID,
) (map[messages.AccountID]*playerv1.Author, error) {
	f.asked = append(f.asked, accounts)
	if f.err != nil {
		return nil, f.err
	}

	found := make(map[messages.AccountID]*playerv1.Author, len(accounts))
	for _, account := range accounts {
		if author, known := f.named[account]; known {
			found[account] = author
		}
	}
	return found, nil
}

type testSuite struct {
	suite.Suite

	db            *cppg.Postgres
	messages      *postgres_message_store.Store
	reactions     *postgres_reaction_store.Store
	announcements *postgres_announcement_store.Store
	authors       *fakeAuthors
}

var (
	now    = time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC)
	window = messages.Window{Size: 2, Retention: 24 * time.Hour}

	ada = messages.AccountID{15: 1}
	bob = messages.AccountID{15: 2}

	laugh = reactions.Reaction(chatv1.Reaction_REACTION_LAUGH)
	clown = reactions.Reaction(chatv1.Reaction_REACTION_CLOWN)

	expired = announcements.AnnouncementID{15: 1}
	old     = announcements.AnnouncementID{15: 2}
	middle  = announcements.AnnouncementID{15: 3}
	latest  = announcements.AnnouncementID{15: 4}
)

func (s *testSuite) SetupSuite() {
	s.db = cppg.StartTestServer(s.T()).OpenSchema(s.T(), "chat", migrations.FS)
	s.messages = postgres_message_store.New(s.db)
	s.reactions = postgres_reaction_store.New(s.db)
	s.announcements = postgres_announcement_store.New(s.db)
}

func (s *testSuite) SetupTest() {
	s.Require().NoError(s.db.Purge(s.T().Context()))
	s.authors = &fakeAuthors{named: map[messages.AccountID]*playerv1.Author{
		ada: {AccountId: ada.String(), Name: "Ada"},
		bob: {AccountId: bob.String(), Name: "Bob"},
	}}
}

func (s *testSuite) query() *history_query.PostgresQuery {
	return history_query.NewPostgresQuery(s.db, s.authors, cptime.NewFixedClock(now), window)
}

func (s *testSuite) history(viewer messages.AccountID) *chatv1.GetHistoryResponse {
	answer, err := s.query().History(s.T().Context(), viewer)
	s.Require().NoError(err)
	return answer
}

func (s *testSuite) sentAt(text string, account messages.AccountID, at time.Time) {
	s.Require().NoError(s.messages.Append(s.T().Context(), messages.NewRecord(messages.Message{
		ID: messages.MessageID(text), SentAt: at, Account: account, CountryID: "fr", Text: text,
	}, "browser-1", "203.0.113.7", "test-agent")))
}

func (s *testSuite) sent(text string, ago time.Duration) {
	s.sentAt(text, ada, now.Add(-ago))
}

func (s *testSuite) reacted(text string, reaction reactions.Reaction, reactor reactions.Reactor, on bool, at time.Time) {
	s.Require().NoError(s.reactions.Save(s.T().Context(), reactions.Change{
		MessageID: messages.MessageID(text), Reaction: reaction, Reactor: reactor, On: on, At: at,
	}))
}

func (s *testSuite) announced(id announcements.AnnouncementID, kind announcements.Kind, at time.Time) {
	s.Require().NoError(s.announcements.Append(s.T().Context(), announcements.Announcement{
		ID: id, Kind: kind, At: at, Payload: json.RawMessage(`{"country":"fr","ground":"de","tile":42,"cleared":3}`),
	}))
}

func texts(answer *chatv1.GetHistoryResponse) []string {
	texts := make([]string, 0, len(answer.GetMessages()))
	for _, message := range answer.GetMessages() {
		texts = append(texts, message.GetText())
	}
	return texts
}

func ids(answer *chatv1.GetHistoryResponse) []string {
	ids := make([]string, 0, len(answer.GetAnnouncements()))
	for _, announcement := range answer.GetAnnouncements() {
		ids = append(ids, announcement.GetId())
	}
	return ids
}

func (s *testSuite) TestAnEmptyChatIsAnEmptyHistory() {
	answer := s.history(ada)

	s.Empty(answer.GetMessages())
	s.Empty(answer.GetAnnouncements())
}

func (s *testSuite) TestTheHistoryIsTheNewestMessagesOldestFirst() {
	s.sent("old", 3*time.Hour)
	s.sent("middle", 2*time.Hour)
	s.sent("new", time.Hour)

	s.Equal([]string{"middle", "new"}, texts(s.history(ada)))
}

func (s *testSuite) TestAMessageOlderThanTheRetentionIsLeftOut() {
	s.sent("expired", 25*time.Hour)
	s.sent("kept", time.Hour)

	s.Equal([]string{"kept"}, texts(s.history(ada)))
}

func (s *testSuite) TestTheOrderIsTheOrderMessagesWereKeptIn() {
	s.sent("first", time.Hour)
	s.sent("second", 2*time.Hour)

	s.Equal([]string{"first", "second"}, texts(s.history(ada)), "the client sorts by time")
}

func (s *testSuite) TestAMessageCarriesWhatItsRowHolds() {
	at := now.Add(-time.Hour).Add(123_456 * time.Microsecond)
	s.sentAt("hello", ada, at)

	message := s.history(ada).GetMessages()[0]

	s.Equal("hello", message.GetId())
	s.Equal(at.UnixMilli(), message.GetSentAtUnixMs(), "milliseconds, cut rather than rounded")
	s.Equal("fr", message.GetCountryId())
	s.Equal("hello", message.GetText())
}

func (s *testSuite) TestEachMessageIsNamedByWhoItsAccountIsNow() {
	title := &playerv1.Title{
		Id: "settler", Name: "Settler", Rank: &playerv1.Rank{TrackId: "conquest", TrackName: "Conquest", Number: 1, Count: 5},
	}
	s.authors.named[ada] = &playerv1.Author{
		AccountId: ada.String(), Name: "Ada Lovelace", Admin: true, Color: playerv1.NameColor_NAME_COLOR_TEAL, Streak: 12,
		WornTitle: title,
	}
	s.sent("one", 2*time.Hour)
	s.sent("two", time.Hour)

	for _, message := range s.history(ada).GetMessages() {
		s.Equal("Ada Lovelace", message.GetAuthorName(), message.GetText())
		s.True(message.GetAuthorAdmin(), message.GetText())
		s.Equal(playerv1.NameColor_NAME_COLOR_TEAL, message.GetAuthorColor(), message.GetText())
		s.Equal(uint32(12), message.GetAuthorStreak(), message.GetText())
		s.True(proto.Equal(title, message.GetAuthorTitle()), message.GetText())
	}
}

func (s *testSuite) TestOneAccountIsAskedAboutOnceHoweverMuchItSaid() {
	s.sent("one", 2*time.Hour)
	s.sent("two", time.Hour)

	s.history(ada)

	s.Equal([][]messages.AccountID{{ada}}, s.authors.asked, "a page costs one ask, not one per message")
}

func (s *testSuite) TestADeletedAccountIsNoLongerNamed() {
	delete(s.authors.named, ada)
	s.sent("hello", time.Hour)

	message := s.history(ada).GetMessages()[0]

	s.Equal(history_query.DeletedName, message.GetAuthorName(), "showing the old name would undo the deletion")
	s.False(message.GetAuthorAdmin())
	s.Equal(playerv1.NameColor_NAME_COLOR_UNSPECIFIED, message.GetAuthorColor())
	s.Zero(message.GetAuthorStreak())
	s.Nil(message.GetAuthorTitle())
	s.Equal("hello", message.GetText(), "what it said stays: a thread keeps its shape")
}

func (s *testSuite) TestAMessageFromBeforeAccountsKeepsTheNameItCarries() {
	s.Require().NoError(s.messages.Append(s.T().Context(), messages.NewRecord(messages.Message{
		ID: "legacy", SentAt: now.Add(-time.Hour), CountryID: "fr", Text: "legacy",
		AuthorName: "guest_Bob", AuthorAdmin: true,
	}, "browser-1", "203.0.113.7", "test-agent")))

	message := s.history(ada).GetMessages()[0]

	s.Equal("guest_Bob", message.GetAuthorName(), "it has no account to read a name from")
	s.True(message.GetAuthorAdmin())
	s.Empty(slices.Concat(s.authors.asked...), "and nobody is asked about it")
}

func (s *testSuite) TestOneAskNamesTheSendersAndThePeopleUnderTheirReactions() {
	s.sent("hello", time.Hour)
	s.reacted("hello", clown, reactions.ReactorOf(bob), true, now.Add(-time.Minute))
	s.reacted("hello", laugh, reactions.ReactorOf(ada), true, now)

	message := s.history(ada).GetMessages()[0]

	s.Equal("Ada", message.GetAuthorName())
	s.Equal([]string{"Bob"}, message.GetReactions()[0].GetReactors())
	s.Equal([][]messages.AccountID{{ada, bob}}, s.authors.asked, "the sender and the reactor in one, each once")
}

func (s *testSuite) TestEachMessageCarriesItsReactionsMarkedForTheViewer() {
	s.sent("hello", time.Hour)
	s.reacted("hello", clown, reactions.ReactorOf(ada), true, now)

	for viewer, mine := range map[messages.AccountID]bool{ada: true, bob: false, messages.NoAccount: false} {
		message := s.history(viewer).GetMessages()[0]

		s.Require().Len(message.GetReactions(), 1, viewer.String())
		count := message.GetReactions()[0]
		s.Equal(chatv1.Reaction_REACTION_CLOWN, count.GetReaction(), viewer.String())
		s.Equal(uint32(1), count.GetCount(), viewer.String())
		s.Equal(mine, count.GetMine(), viewer.String())
		s.Equal([]string{"Ada"}, count.GetReactors(), viewer.String())
		s.Equal(uint64(1), message.GetReactionsVersion(), viewer.String())
	}
}

func (s *testSuite) TestReactionsKeepThePlaceTheyFirstAppearedInAndSoDoThePeopleUnderThem() {
	s.sent("hello", time.Hour)
	s.reacted("hello", clown, reactions.ReactorOf(bob), true, now.Add(-3*time.Minute))
	s.reacted("hello", laugh, reactions.ReactorOf(ada), true, now.Add(-2*time.Minute))
	s.reacted("hello", clown, reactions.ReactorOf(ada), true, now.Add(-time.Minute))

	counts := s.history(ada).GetMessages()[0].GetReactions()

	s.Require().Len(counts, 2)
	s.Equal(chatv1.Reaction_REACTION_CLOWN, counts[0].GetReaction())
	s.Equal([]string{"Bob", "Ada"}, counts[0].GetReactors())
	s.Equal(chatv1.Reaction_REACTION_LAUGH, counts[1].GetReaction())
	s.Equal([]string{"Ada"}, counts[1].GetReactors())
}

func (s *testSuite) TestAMessageNobodyReactedToHasNoReactionsAtVersionZero() {
	s.sent("hello", time.Hour)

	message := s.history(ada).GetMessages()[0]

	s.Empty(message.GetReactions())
	s.Zero(message.GetReactionsVersion())
}

func (s *testSuite) TestAMessageWhoseReactionsWereAllTakenOffKeepsItsVersion() {
	s.sent("hello", time.Hour)
	s.reacted("hello", clown, reactions.ReactorOf(ada), true, now)
	s.reacted("hello", clown, reactions.ReactorOf(ada), false, now)

	message := s.history(ada).GetMessages()[0]

	s.Empty(message.GetReactions())
	s.Equal(uint64(2), message.GetReactionsVersion(), "a client holding version 1 must learn of version 2")
}

func (s *testSuite) TestSomebodyNobodyCanNameIsCountedWithoutBeingNamed() {
	gone := messages.AccountID{15: 9}
	s.sent("hello", time.Hour)
	s.reacted("hello", clown, reactions.ReactorOf(ada), true, now.Add(-2*time.Minute))
	s.reacted("hello", clown, reactions.ReactorOf(gone), true, now.Add(-time.Minute))
	s.reacted("hello", clown, "guest:91aa3d", true, now)

	count := s.history(ada).GetMessages()[0].GetReactions()[0]

	s.Equal(uint32(3), count.GetCount())
	s.Equal([]string{"Ada"}, count.GetReactors(), "a list of reactions is not the place to say somebody is gone")
	s.Equal([][]messages.AccountID{{ada, gone}}, s.authors.asked, "a guest tag is nobody, so nobody is asked about it")
}

func (s *testSuite) TestACountNamesOnlyItsFirstReactors() {
	s.sent("hello", time.Hour)
	reactors := history_query.NamedReactors + 5
	names := make([]string, 0, reactors)
	for i := range reactors {
		account := messages.AccountID{14: 1, 15: byte(i)}
		name := fmt.Sprintf("Player%02d", i)
		s.authors.named[account] = &playerv1.Author{AccountId: account.String(), Name: name}
		names = append(names, name)
		s.reacted("hello", clown, reactions.ReactorOf(account), true, now.Add(time.Duration(i-reactors)*time.Second))
	}

	count := s.history(ada).GetMessages()[0].GetReactions()[0]

	s.Equal(uint32(reactors), count.GetCount()) //nolint:gosec // a small test count.
	s.Equal(names[:history_query.NamedReactors], count.GetReactors())
}

func (s *testSuite) TestTheHistoryCarriesTheNewestAnnouncementsOldestFirst() {
	s.sent("hello", time.Hour)
	ago := map[announcements.AnnouncementID]time.Duration{expired: 40, old: 20, middle: 10, latest: 4}
	for _, id := range []announcements.AnnouncementID{latest, expired, middle, old} {
		s.announced(id, announcements.KindBomb, now.Add(-ago[id]*time.Hour))
	}

	answer := s.history(ada)

	s.Equal([]string{uuidOf(middle), uuidOf(latest)}, ids(answer))
	s.Equal([]string{"hello"}, texts(answer), "announcements do not take the messages' places")
}

func (s *testSuite) TestAnAnnouncementCarriesWhatItsRowHolds() {
	at := now.Add(-time.Hour).Add(123_456 * time.Microsecond)
	s.announced(latest, announcements.KindBomb, at)

	announcement := s.history(ada).GetAnnouncements()[0]

	s.Equal(uuidOf(latest), announcement.GetId())
	s.Equal(at.UnixMilli(), announcement.GetAnnouncedAtUnixMs())
	s.Equal("bomb", announcement.GetKind())
	s.JSONEq(`{"country":"fr","ground":"de","tile":42,"cleared":3}`, announcement.GetPayload())
}

func (s *testSuite) TestAnnouncementsBeginWhereAFullWindowOfMessagesDoes() {
	s.sent("old", 3*time.Hour)
	s.sent("middle", 2*time.Hour)
	s.sent("new", time.Hour)
	s.announced(old, announcements.KindBomb, now.Add(-150*time.Minute))
	s.announced(middle, announcements.KindBomb, now.Add(-90*time.Minute))

	answer := s.history(ada)

	s.Equal([]string{"middle", "new"}, texts(answer))
	s.Equal([]string{uuidOf(middle)}, ids(answer),
		"one from before the oldest message shown would sit on top of the chat, among messages left out")
}

func (s *testSuite) TestAReactionTheProtoDoesNotNameIsAnError() {
	s.sent("hello", time.Hour)
	s.reacted("hello", reactions.Reaction(99), reactions.ReactorOf(ada), true, now)

	_, err := s.query().History(s.T().Context(), ada)

	s.ErrorIs(err, history_query.ErrUnknownReaction)
}

func (s *testSuite) TestAnAnnouncementOfAKindNobodyKnowsIsAnError() {
	s.announced(latest, announcements.Kind("meteor"), now.Add(-time.Hour))

	_, err := s.query().History(s.T().Context(), ada)

	s.ErrorIs(err, history_query.ErrUnknownKind)
}

func (s *testSuite) TestAHistoryNobodyCanBeNamedInIsARefusal() {
	s.sent("hello", time.Hour)
	s.authors.err = errors.New("the player module is down")

	_, err := s.query().History(s.T().Context(), ada)

	s.ErrorIs(err, s.authors.err, "a chat of anonymous messages is worse than none")
}

func uuidOf(id announcements.AnnouncementID) string {
	return uuid.UUID(id).String()
}
