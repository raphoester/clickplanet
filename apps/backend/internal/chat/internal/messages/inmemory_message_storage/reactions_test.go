package inmemory_message_storage_test

import (
	"context"
	"errors"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages/inmemory_message_storage"
)

const (
	clown messages.Reaction = 2
	laugh messages.Reaction = 1

	ada messages.Reactor = "account:ada"
	bo  messages.Reactor = "guest:91aa3d"
)

func (s *testSuite) change(message string, reaction messages.Reaction, reactor messages.Reactor, on bool) messages.ReactionChange {
	return messages.ReactionChange{
		MessageID: messages.MessageID(message),
		Reaction:  reaction,
		Reactor:   reactor,
		On:        on,
		At:        s.clock.Now(),
	}
}

func (s *testSuite) react(storage *inmemory_message_storage.Storage, change messages.ReactionChange) messages.Reactions {
	reactions, err := storage.React(context.Background(), change)
	s.Require().NoError(err)
	return reactions
}

func (s *testSuite) TestAReactionShowsUpInHistoryMarkedForItsReactor() {
	storage := s.newStorage(inmemory_message_storage.Config{})
	s.Require().NoError(storage.Append(context.Background(), s.record("hello")))

	s.react(storage, s.change("hello", clown, ada, true))
	s.react(storage, s.change("hello", clown, bo, true))

	s.Equal([]messages.Count{{Reaction: clown, Count: 2, Mine: true}},
		storage.History(context.Background(), ada)[0].Reactions)
	s.Equal([]messages.Count{{Reaction: clown, Count: 2, Mine: false}},
		storage.History(context.Background(), messages.NoReactor)[0].Reactions)
}

func (s *testSuite) TestAReactionIsRecordedAndTakenOff() {
	storage := s.newStorage(inmemory_message_storage.Config{})
	s.Require().NoError(storage.Append(context.Background(), s.record("hello")))

	s.react(storage, s.change("hello", clown, ada, true))
	s.Len(s.persistence.StoredReactions(), 1)

	s.react(storage, s.change("hello", clown, ada, false))
	s.Empty(s.persistence.StoredReactions())
	s.Empty(storage.History(context.Background(), ada)[0].Reactions)
}

func (s *testSuite) TestSubscribersHearEveryReactionAfterItsMessage() {
	storage := s.newStorage(inmemory_message_storage.Config{})
	feed, err := storage.Subscribe(s.T().Context())
	s.Require().NoError(err)

	s.Require().NoError(storage.Append(context.Background(), s.record("hello")))
	s.react(storage, s.change("hello", clown, ada, true))
	s.react(storage, s.change("hello", laugh, bo, true))

	s.Require().Eventually(func() bool { return len(feed) == 3 }, 2*time.Second, time.Millisecond)
	s.Equal("hello", (<-feed).Message.Text)
	s.Equal(&messages.Tally{MessageID: "hello", Counts: []messages.Count{{Reaction: clown, Count: 1}}}, (<-feed).Reactions)
	s.Equal(&messages.Tally{MessageID: "hello", Counts: []messages.Count{
		{Reaction: clown, Count: 1},
		{Reaction: laugh, Count: 1},
	}}, (<-feed).Reactions, "the stream tallies for nobody")
}

func (s *testSuite) TestAChangeThatChangesNothingIsNeitherRecordedNorPublished() {
	storage := s.newStorage(inmemory_message_storage.Config{})
	s.Require().NoError(storage.Append(context.Background(), s.record("hello")))
	s.react(storage, s.change("hello", clown, ada, true))

	feed, err := storage.Subscribe(s.T().Context())
	s.Require().NoError(err)
	s.persistence.FailWith(errors.New("a write would fail"))

	s.True(s.react(storage, s.change("hello", clown, ada, true)).Given(clown, ada))
	s.False(s.react(storage, s.change("hello", laugh, ada, false)).Given(laugh, ada))
	s.Empty(feed)
}

func (s *testSuite) TestAReactionToAMessageNotInHistoryIsRefused() {
	storage := s.newStorage(inmemory_message_storage.Config{HistorySize: 1})
	s.Require().NoError(storage.Append(context.Background(), s.record("gone")))
	s.Require().NoError(storage.Append(context.Background(), s.record("hello")))

	for _, message := range []string{"gone", "never-sent"} {
		_, err := storage.React(context.Background(), s.change(message, clown, ada, true))
		s.Require().ErrorIs(err, messages.ErrUnknownMessage)
	}
	s.Empty(s.persistence.StoredReactions())
}

func (s *testSuite) TestAReactionThatCannotBeStoredIsNotPublished() {
	storage := s.newStorage(inmemory_message_storage.Config{})
	s.Require().NoError(storage.Append(context.Background(), s.record("hello")))
	feed, err := storage.Subscribe(s.T().Context())
	s.Require().NoError(err)

	s.persistence.FailWith(errors.New("postgres is down"))
	_, err = storage.React(context.Background(), s.change("hello", clown, ada, true))
	s.Require().Error(err)

	s.Empty(feed)
	s.Empty(storage.History(context.Background(), ada)[0].Reactions)
}

func (s *testSuite) TestAMessageLeavingHistoryTakesItsReactionsWithIt() {
	storage := s.newStorage(inmemory_message_storage.Config{HistorySize: 1})
	s.Require().NoError(storage.Append(context.Background(), s.record("hello")))
	s.react(storage, s.change("hello", clown, ada, true))

	s.Require().NoError(storage.Append(context.Background(), s.record("planet")))
	s.Require().NoError(storage.Append(context.Background(), s.record("hello")))

	s.Empty(storage.History(context.Background(), ada)[0].Reactions, "a message sent again under the same id starts bare")
}

func (s *testSuite) TestLoadReplaysTheReactionsOfTheMessagesInHistory() {
	s.persistence = inmemory_message_storage.NewMemoryPersistence(s.record("hello"), s.record("planet")).
		WithReactions(
			s.change("hello", laugh, bo, true),
			s.change("hello", clown, ada, true),
			s.change("planet", clown, bo, true),
		)

	history := s.newStorage(inmemory_message_storage.Config{}).History(context.Background(), ada)

	s.Equal([]messages.Count{{Reaction: laugh, Count: 1}, {Reaction: clown, Count: 1, Mine: true}}, history[0].Reactions)
	s.Equal([]messages.Count{{Reaction: clown, Count: 1}}, history[1].Reactions)
}

func (s *testSuite) TestRunDeletesReactionsPastRetention() {
	storage := s.newStorage(inmemory_message_storage.Config{Retention: 24 * time.Hour, PruneInterval: time.Millisecond})
	s.Require().NoError(storage.Append(context.Background(), s.record("hello")))
	s.react(storage, s.change("hello", clown, ada, true))
	s.clock.Advance(48 * time.Hour)
	s.react(storage, s.change("hello", laugh, ada, true))

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		storage.Run(ctx)
	}()

	s.Require().Eventually(func() bool { return len(s.persistence.StoredReactions()) == 1 }, 5*time.Second, time.Millisecond)
	cancel()
	<-done

	s.Equal(laugh, s.persistence.StoredReactions()[0].Reaction)
}
