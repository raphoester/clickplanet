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
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

func TestRunSuite(t *testing.T) {
	suite.Run(t, new(testSuite))
}

type fakePublisher struct {
	updates []feed.Update
}

func (f *fakePublisher) Publish(update feed.Update) {
	f.updates = append(f.updates, update)
}

type failingBoard struct {
	*inmemory_reaction_storage.Storage
}

func (failingBoard) Save(context.Context, reactions.Change) error {
	return errors.New("postgres is down")
}

var (
	ada    = cpsession.AccountID{15: 1}
	bob    = cpsession.AccountID{15: 2}
	clown  = reactions.Reaction(2)
	laugh  = reactions.Reaction(1)
	now    = time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC)
	window = messages.NewWindow(1, 24*time.Hour)
	flags  = messages.FakeFlags{ada: "fr", bob: "de"}
)

type testSuite struct {
	suite.Suite

	messages  *inmemory_message_storage.Storage
	board     *inmemory_reaction_storage.Storage
	publisher *fakePublisher
}

func (s *testSuite) SetupTest() {
	s.messages = inmemory_message_storage.New()
	s.board = inmemory_reaction_storage.New()
	s.publisher = &fakePublisher{}
	s.sent("hello")
}

func (s *testSuite) sent(id messages.MessageID) {
	s.sentIn(id, messages.World)
}

func (s *testSuite) sentIn(id messages.MessageID, room messages.Room) {
	s.Require().NoError(s.messages.Append(context.Background(),
		messages.NewRecord(s.message(id, room), "", "")))
}

func (s *testSuite) message(id messages.MessageID, room messages.Room) messages.Message {
	return messages.Restored(s.T(), messages.Snapshot{ID: id, SentAt: now.Add(-time.Hour), Room: room})
}

func (s *testSuite) reactWith(board react_usecase.Board, in react_usecase.In) (react_usecase.Out, error) {
	clock := cptime.NewFixedClock(now)
	shown := messages.NewShowing(s.messages, messages.NewRooms(flags), clock, window)
	return react_usecase.New(shown, board, s.publisher, clock).Execute(context.Background(), in)
}

func (s *testSuite) react(account messages.AccountID, reaction reactions.Reaction, on bool) []reactions.Count {
	out, err := s.reactWith(s.board, react_usecase.In{Account: account, MessageID: "hello", Reaction: reaction, On: on})
	s.Require().NoError(err)
	return out.Counts
}

func (s *testSuite) TestAnAccountReactsAsItselfAndIsAnsweredWithItsOwn() {
	s.Equal([]reactions.Count{
		reactions.CountOf(clown, 1, true, []reactions.Reactor{reactions.ReactorOf(ada)}),
	}, s.react(ada, clown, true))

	given, err := s.board.Reactions(context.Background(), []messages.MessageID{"hello"})
	s.Require().NoError(err)
	s.True(given["hello"].Given(clown, reactions.ReactorOf(ada)))
}

func (s *testSuite) TestTwoAccountsAreTwoReactors() {
	s.react(ada, clown, true)

	s.Equal([]reactions.Count{
		reactions.CountOf(clown, 2, true, []reactions.Reactor{reactions.ReactorOf(ada), reactions.ReactorOf(bob)}),
	}, s.react(bob, clown, true))
}

func (s *testSuite) TestNoAccountIsRefusedAndNothingIsSaved() {
	_, err := s.reactWith(s.board, react_usecase.In{Account: cpsession.NoAccount, MessageID: "hello", Reaction: clown, On: true})

	s.Require().ErrorIs(err, messages.ErrNoAccount)
	s.Empty(s.publisher.updates)
}

func (s *testSuite) TestEveryChangeIsPublishedAsTheWholeTallyForNobodyVersioned() {
	s.react(ada, clown, true)
	s.react(bob, laugh, true)
	s.react(ada, clown, false)

	adaGave := []reactions.Reactor{reactions.ReactorOf(ada)}
	bobGave := []reactions.Reactor{reactions.ReactorOf(bob)}

	hello := s.message("hello", messages.World)
	s.Equal([]feed.Update{
		feed.ReactionsChanged(hello, reactions.TallyFor("hello", []reactions.Count{
			reactions.CountOf(clown, 1, false, adaGave),
		}, 1)),
		feed.ReactionsChanged(hello, reactions.TallyFor("hello", []reactions.Count{
			reactions.CountOf(clown, 1, false, adaGave),
			reactions.CountOf(laugh, 1, false, bobGave),
		}, 2)),
		feed.ReactionsChanged(hello, reactions.TallyFor("hello", []reactions.Count{
			reactions.CountOf(laugh, 1, false, bobGave),
		}, 3)),
	}, s.publisher.updates)
}

func (s *testSuite) TestTheAnswerCarriesTheVersionItWasReadAt() {
	s.react(ada, clown, true)

	out, err := s.reactWith(s.board, react_usecase.In{Account: ada, MessageID: "hello", Reaction: clown, On: true})

	s.Require().NoError(err)
	s.Equal(react_usecase.Out{Counts: []reactions.Count{
		reactions.CountOf(clown, 1, true, []reactions.Reactor{reactions.ReactorOf(ada)}),
	}, Version: 1}, out, "a change that changes nothing answers what is there")
}

func (s *testSuite) TestAChangeThatChangesNothingIsNeitherSavedNorPublished() {
	s.react(ada, clown, true)
	s.publisher.updates = nil

	mine := []reactions.Count{
		reactions.CountOf(clown, 1, true, []reactions.Reactor{reactions.ReactorOf(ada)}),
	}
	s.Equal(mine, s.react(ada, clown, true))
	s.Equal(mine, s.react(ada, laugh, false))
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

func (s *testSuite) TestAReactionThatCannotBeSavedIsNotPublished() {
	_, err := s.reactWith(failingBoard{s.board}, react_usecase.In{Account: ada, MessageID: "hello", Reaction: clown, On: true})

	s.Require().Error(err)
	s.Empty(s.publisher.updates)
}

func (s *testSuite) TestACountSaysWhichAccountsGaveItOldestFirst() {
	s.react(ada, clown, true)

	counts := s.react(bob, clown, true)

	s.Equal([]messages.AccountID{ada, bob}, counts[0].Accounts(), "the page names them")
	s.Equal(2, counts[0].Total())
}

func (s *testSuite) TestOnlyTheAccountsThatPlayForAFlagReactInItsRoom() {
	s.sentIn("plan", messages.FlagRoom("fr"))

	_, err := s.reactWith(s.board, react_usecase.In{Account: bob, MessageID: "plan", Reaction: clown, On: true})
	s.Require().ErrorIs(err, messages.ErrNotInRoom)
	s.Empty(s.publisher.updates)

	out, err := s.reactWith(s.board, react_usecase.In{Account: ada, MessageID: "plan", Reaction: clown, On: true})
	s.Require().NoError(err)
	s.Equal(1, out.Counts[0].Total())
	s.Require().Len(s.publisher.updates, 1)
	s.True(s.publisher.updates[0].For(messages.FlagRoom("fr")), "the tally goes to the room of the message")
	s.False(s.publisher.updates[0].For(messages.World))
}
