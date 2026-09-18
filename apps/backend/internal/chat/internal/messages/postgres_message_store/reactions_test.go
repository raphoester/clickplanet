package postgres_message_store_test

import (
	"context"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
)

func reaction(message messages.MessageID, reaction messages.Reaction, reactor messages.Reactor, at time.Time) messages.ReactionChange {
	return messages.ReactionChange{MessageID: message, Reaction: reaction, Reactor: reactor, On: true, At: at}
}

func (s *testSuite) reactions(ids ...messages.MessageID) []messages.ReactionChange {
	given, err := s.store.Reactions(context.Background(), ids)
	s.Require().NoError(err)
	return given
}

func (s *testSuite) TestReactionsReadBackOldestFirstForTheMessagesAsked() {
	ctx := context.Background()
	clown := reaction("id-hello", 2, "guest:a1b2c3", start.Add(time.Second))
	laugh := reaction("id-hello", 1, "account:ada", start)
	other := reaction("id-planet", 1, "account:ada", start)
	for _, change := range []messages.ReactionChange{clown, laugh, other} {
		s.Require().NoError(s.store.InsertReaction(ctx, change))
	}

	s.Equal([]messages.ReactionChange{laugh, clown}, s.reactions("id-hello"))
	s.Empty(s.reactions())
}

func (s *testSuite) TestAReactionPutOnTwiceIsOneRow() {
	ctx := context.Background()
	first := reaction("id-hello", 2, "account:ada", start)
	s.Require().NoError(s.store.InsertReaction(ctx, first))
	s.Require().NoError(s.store.InsertReaction(ctx, reaction("id-hello", 2, "account:ada", start.Add(time.Hour))))

	s.Equal([]messages.ReactionChange{first}, s.reactions("id-hello"))
}

func (s *testSuite) TestDeleteReactionTakesOffOnlyThatOne() {
	ctx := context.Background()
	mine := reaction("id-hello", 2, "account:ada", start)
	theirs := reaction("id-hello", 2, "guest:a1b2c3", start)
	s.Require().NoError(s.store.InsertReaction(ctx, mine))
	s.Require().NoError(s.store.InsertReaction(ctx, theirs))

	s.Require().NoError(s.store.DeleteReaction(ctx, mine))
	s.Require().NoError(s.store.DeleteReaction(ctx, mine), "taking off what is not there is not an error")

	s.Equal([]messages.ReactionChange{theirs}, s.reactions("id-hello"))
}

func (s *testSuite) TestDeleteReactionsBeforeRemovesOnlyOlderReactions() {
	ctx := context.Background()
	recent := reaction("id-hello", 1, "account:ada", start.Add(48*time.Hour))
	s.Require().NoError(s.store.InsertReaction(ctx, reaction("id-hello", 2, "account:ada", start)))
	s.Require().NoError(s.store.InsertReaction(ctx, recent))

	deleted, err := s.store.DeleteReactionsBefore(ctx, start.Add(24*time.Hour))
	s.Require().NoError(err)

	s.Equal(int64(1), deleted)
	s.Equal([]messages.ReactionChange{recent}, s.reactions("id-hello"))
}
