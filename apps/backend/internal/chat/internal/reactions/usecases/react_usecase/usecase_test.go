package react_usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/feed"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages/inmemory_message_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/reactions"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/reactions/inmemory_reaction_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/reactions/usecases/react_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

func TestRunSuite(t *testing.T) {
	suite.Run(t, new(testSuite))
}

type fakeAuthors struct {
	author messages.Author
	ips    []string
	err    error
}

func (f *fakeAuthors) Author(_ context.Context, _ messages.AccountID, ip string) (messages.Author, error) {
	f.ips = append(f.ips, ip)
	return f.author, f.err
}

type fakePublisher struct {
	updates []feed.Update
}

func (f *fakePublisher) Publish(update feed.Update) {
	f.updates = append(f.updates, update)
}

// failingBoard is a board whose saves fail.
type failingBoard struct {
	*inmemory_reaction_storage.Storage
}

func (failingBoard) Save(context.Context, reactions.Change) error {
	return errors.New("postgres is down")
}

var (
	ada    = cpsession.AccountID{15: 1}
	clown  = reactions.Reaction(2)
	laugh  = reactions.Reaction(1)
	now    = time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC)
	player = messages.Author{Username: "ada", Tag: "a1b2c3"}
	window = messages.Window{Size: 1, Retention: 24 * time.Hour}
)

type testSuite struct {
	suite.Suite

	messages  *inmemory_message_storage.Storage
	board     *inmemory_reaction_storage.Storage
	authors   *fakeAuthors
	publisher *fakePublisher
}

func (s *testSuite) SetupTest() {
	s.messages = inmemory_message_storage.New()
	s.board = inmemory_reaction_storage.New()
	s.authors = &fakeAuthors{author: player}
	s.publisher = &fakePublisher{}
	s.sent("hello")
}

func (s *testSuite) sent(id messages.MessageID) {
	s.Require().NoError(s.messages.Append(context.Background(), messages.Record{
		Message: messages.Message{ID: id, SentAt: now.Add(-time.Hour)},
	}))
}

func (s *testSuite) reactWith(board react_usecase.Board, in react_usecase.In) ([]reactions.Count, error) {
	ctx := cpctx.AddIPToContext(context.Background(), "1.2.3.4")
	return react_usecase.New(s.messages, board, s.authors, s.publisher, cptime.NewFixedClock(now), window).Execute(ctx, in)
}

func (s *testSuite) react(account messages.AccountID, reaction reactions.Reaction, on bool) []reactions.Count {
	counts, err := s.reactWith(s.board, react_usecase.In{Account: account, MessageID: "hello", Reaction: reaction, On: on})
	s.Require().NoError(err)
	return counts
}

func (s *testSuite) TestAPlayerReactsAsItsAccountAndIsAnsweredWithItsOwn() {
	s.Equal([]reactions.Count{{Reaction: clown, Count: 1, Mine: true}}, s.react(ada, clown, true))

	given, err := s.board.Reactions(context.Background(), []messages.MessageID{"hello"})
	s.Require().NoError(err)
	s.True(given["hello"].Given(clown, reactions.ReactorOf(ada, player)))
	s.Equal([]string{"1.2.3.4"}, s.authors.ips, "a guest is the tag of the address the reaction came from")
}

func (s *testSuite) TestAPlayerAndAGuestOnOneAddressAreTwoReactors() {
	s.react(ada, clown, true)

	s.Equal([]reactions.Count{{Reaction: clown, Count: 2, Mine: true}}, s.react(cpsession.NoAccount, clown, true))
}

func (s *testSuite) TestEveryChangeIsPublishedAsTheWholeTallyForNobody() {
	s.react(ada, clown, true)
	s.react(cpsession.NoAccount, laugh, true)
	s.react(ada, clown, false)

	s.Equal([]feed.Update{
		{Reactions: &reactions.Tally{MessageID: "hello", Counts: []reactions.Count{{Reaction: clown, Count: 1}}}},
		{Reactions: &reactions.Tally{MessageID: "hello", Counts: []reactions.Count{
			{Reaction: clown, Count: 1}, {Reaction: laugh, Count: 1},
		}}},
		{Reactions: &reactions.Tally{MessageID: "hello", Counts: []reactions.Count{{Reaction: laugh, Count: 1}}}},
	}, s.publisher.updates)
}

func (s *testSuite) TestAChangeThatChangesNothingIsNeitherSavedNorPublished() {
	s.react(ada, clown, true)
	s.publisher.updates = nil

	s.Equal([]reactions.Count{{Reaction: clown, Count: 1, Mine: true}}, s.react(ada, clown, true))
	s.Equal([]reactions.Count{{Reaction: clown, Count: 1, Mine: true}}, s.react(ada, laugh, false))
	s.Empty(s.publisher.updates)
}

func (s *testSuite) TestAMessageTheChatNoLongerShowsIsRefused() {
	s.sent("newer")

	for _, id := range []messages.MessageID{"hello", "never-sent"} {
		_, err := s.reactWith(s.board, react_usecase.In{Account: ada, MessageID: id, Reaction: clown, On: true})
		s.Require().ErrorIs(err, reactions.ErrUnknownMessage)
	}
	s.Empty(s.publisher.updates)
}

func (s *testSuite) TestAReactorThePlayerModuleCannotNameIsRefused() {
	s.authors.err = errors.New("player module is down")

	_, err := s.reactWith(s.board, react_usecase.In{Account: ada, MessageID: "hello", Reaction: clown, On: true})

	s.Require().ErrorIs(err, messages.ErrAuthorUnavailable)
	s.Empty(s.publisher.updates)
}

func (s *testSuite) TestAReactionThatCannotBeSavedIsNotPublished() {
	_, err := s.reactWith(failingBoard{s.board}, react_usecase.In{Account: ada, MessageID: "hello", Reaction: clown, On: true})

	s.Require().Error(err)
	s.Empty(s.publisher.updates)
}
