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

// fakeAuthors is the player module, which knows what each account is called. An account it is not told about
// is one nobody can name any more: deleted.
type fakeAuthors struct {
	named map[messages.AccountID]messages.Author
	asked int
	err   error
}

func (f *fakeAuthors) Authors(
	_ context.Context,
	accounts []messages.AccountID,
) (map[messages.AccountID]messages.Author, error) {
	f.asked++
	if f.err != nil {
		return nil, f.err
	}

	found := make(map[messages.AccountID]messages.Author, len(accounts))
	for _, account := range accounts {
		if author, known := f.named[account]; known {
			found[account] = author
		}
	}
	return found, nil
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
	bob    = cpsession.AccountID{15: 2}
	clown  = reactions.Reaction(2)
	laugh  = reactions.Reaction(1)
	now    = time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC)
	window = messages.Window{Size: 1, Retention: 24 * time.Hour}
)

type testSuite struct {
	suite.Suite

	messages  *inmemory_message_storage.Storage
	board     *inmemory_reaction_storage.Storage
	publisher *fakePublisher
	authors   *fakeAuthors
}

func (s *testSuite) SetupTest() {
	s.messages = inmemory_message_storage.New()
	s.board = inmemory_reaction_storage.New()
	s.publisher = &fakePublisher{}
	s.authors = &fakeAuthors{named: map[messages.AccountID]messages.Author{
		ada: {Name: "Ada"},
		bob: {Name: "Bob"},
	}}
	s.sent("hello")
}

func (s *testSuite) sent(id messages.MessageID) {
	s.Require().NoError(s.messages.Append(context.Background(), messages.Record{
		Message: messages.Message{ID: id, SentAt: now.Add(-time.Hour)},
	}))
}

func (s *testSuite) reactWith(board react_usecase.Board, in react_usecase.In) (react_usecase.Out, error) {
	useCase := react_usecase.New(s.messages, board, s.publisher, s.authors, cptime.NewFixedClock(now), window)
	return useCase.Execute(context.Background(), in)
}

func (s *testSuite) react(account messages.AccountID, reaction reactions.Reaction, on bool) []reactions.Count {
	out, err := s.reactWith(s.board, react_usecase.In{Account: account, MessageID: "hello", Reaction: reaction, On: on})
	s.Require().NoError(err)
	return out.Counts
}

func (s *testSuite) TestAnAccountReactsAsItselfAndIsAnsweredWithItsOwn() {
	s.Equal([]reactions.Count{{
		Reaction: clown, Count: 1, Mine: true,
		Reactors: []reactions.Reactor{reactions.ReactorOf(ada)}, Names: []string{"Ada"},
	}}, s.react(ada, clown, true))

	given, err := s.board.Reactions(context.Background(), []messages.MessageID{"hello"})
	s.Require().NoError(err)
	s.True(given["hello"].Given(clown, reactions.ReactorOf(ada)))
}

func (s *testSuite) TestTwoAccountsAreTwoReactors() {
	s.react(ada, clown, true)

	s.Equal([]reactions.Count{{
		Reaction: clown, Count: 2, Mine: true,
		Reactors: []reactions.Reactor{reactions.ReactorOf(ada), reactions.ReactorOf(bob)},
		Names:    []string{"Ada", "Bob"},
	}}, s.react(bob, clown, true))
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

	s.Equal([]feed.Update{
		{Reactions: &reactions.Tally{MessageID: "hello", Version: 1, Counts: []reactions.Count{
			{Reaction: clown, Count: 1, Reactors: adaGave, Names: []string{"Ada"}},
		}}},
		{Reactions: &reactions.Tally{MessageID: "hello", Version: 2, Counts: []reactions.Count{
			{Reaction: clown, Count: 1, Reactors: adaGave, Names: []string{"Ada"}},
			{Reaction: laugh, Count: 1, Reactors: bobGave, Names: []string{"Bob"}},
		}}},
		{Reactions: &reactions.Tally{MessageID: "hello", Version: 3, Counts: []reactions.Count{
			{Reaction: laugh, Count: 1, Reactors: bobGave, Names: []string{"Bob"}},
		}}},
	}, s.publisher.updates)
}

func (s *testSuite) TestTheAnswerCarriesTheVersionItWasReadAt() {
	s.react(ada, clown, true)

	out, err := s.reactWith(s.board, react_usecase.In{Account: ada, MessageID: "hello", Reaction: clown, On: true})

	s.Require().NoError(err)
	s.Equal(react_usecase.Out{Counts: []reactions.Count{{
		Reaction: clown, Count: 1, Mine: true,
		Reactors: []reactions.Reactor{reactions.ReactorOf(ada)}, Names: []string{"Ada"},
	}}, Version: 1}, out, "a change that changes nothing answers what is there")
}

func (s *testSuite) TestAChangeThatChangesNothingIsNeitherSavedNorPublished() {
	s.react(ada, clown, true)
	s.publisher.updates = nil

	mine := []reactions.Count{{
		Reaction: clown, Count: 1, Mine: true,
		Reactors: []reactions.Reactor{reactions.ReactorOf(ada)}, Names: []string{"Ada"},
	}}
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

func (s *testSuite) TestEveryoneUnderAMessageIsNamedInOneAsk() {
	s.react(ada, clown, true)
	s.authors.asked = 0

	counts := s.react(bob, clown, true)

	s.Equal([]string{"Ada", "Bob"}, counts[0].Names, "oldest first")
	s.Equal(1, s.authors.asked, "the answer and the frame that goes out share it")
}

func (s *testSuite) TestARenameShowsUnderEveryReactionItsPlayerEverGave() {
	s.react(ada, clown, true)
	s.react(ada, laugh, true)
	s.authors.named[ada] = messages.Author{Name: "Ada Lovelace"}

	counts := s.react(bob, clown, true)

	s.Equal([]string{"Ada Lovelace", "Bob"}, counts[0].Names)
}

func (s *testSuite) TestADeletedAccountIsCountedWithoutBeingNamed() {
	s.react(ada, clown, true)
	delete(s.authors.named, ada)

	counts := s.react(bob, clown, true)

	s.Equal(2, counts[0].Count)
	s.Equal([]string{"Bob"}, counts[0].Names, "the count says how many, the names say who is still there")
}

func (s *testSuite) TestAReactionNobodyCanBeNamedUnderIsARefusal() {
	s.authors.err = errors.New("the player module is down")

	_, err := s.reactWith(s.board, react_usecase.In{Account: ada, MessageID: "hello", Reaction: clown, On: true})

	s.Require().Error(err)
}
