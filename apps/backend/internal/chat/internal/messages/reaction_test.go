package messages_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
)

const (
	laugh messages.Reaction = 1
	clown messages.Reaction = 2
	skull messages.Reaction = 3

	ada messages.Reactor = "account:ada"
	bo  messages.Reactor = "guest:91aa3d"
)

func TestReactionsCountEachReactionInTheOrderItFirstAppeared(t *testing.T) {
	reactions := messages.Reactions{}.With(clown, ada).With(laugh, bo).With(clown, bo)

	assert.Equal(t, []messages.Count{
		{Reaction: clown, Count: 2, Mine: true},
		{Reaction: laugh, Count: 1, Mine: false},
	}, reactions.Tally(ada))
}

func TestAReactorGivesAReactionOnce(t *testing.T) {
	reactions := messages.Reactions{}.With(clown, ada).With(clown, ada)

	assert.Equal(t, []messages.Count{{Reaction: clown, Count: 1, Mine: true}}, reactions.Tally(ada))
}

func TestAReactionNobodyGivesAnyMoreLosesItsPlace(t *testing.T) {
	reactions := messages.Reactions{}.With(clown, ada).With(laugh, ada).Without(clown, ada).With(clown, bo)

	assert.Equal(t, []messages.Count{
		{Reaction: laugh, Count: 1},
		{Reaction: clown, Count: 1},
	}, reactions.Tally(messages.NoReactor))
}

func TestTakingOffAReactionNotGivenChangesNothing(t *testing.T) {
	reactions := messages.Reactions{}.With(clown, ada)

	assert.Equal(t, reactions, reactions.Without(clown, bo))
	assert.Equal(t, reactions, reactions.Without(skull, ada))
}

func TestWithAndWithoutLeaveTheReceiverAsItWas(t *testing.T) {
	before := messages.Reactions{}.With(clown, ada)

	_ = before.With(clown, bo)
	_ = before.With(laugh, bo)
	_ = before.Without(clown, ada)

	assert.Equal(t, []messages.Count{{Reaction: clown, Count: 1}}, before.Tally(messages.NoReactor))
}

func TestNoReactorOwnsNothing(t *testing.T) {
	assert.False(t, messages.Reactions{}.With(clown, ada).Tally(messages.NoReactor)[0].Mine)
}

func TestAppliedPutsOnAndTakesOff(t *testing.T) {
	on := messages.Reactions{}.Applied(messages.ReactionChange{Reaction: skull, Reactor: ada, On: true})
	off := on.Applied(messages.ReactionChange{Reaction: skull, Reactor: ada, On: false})

	assert.True(t, on.Given(skull, ada))
	assert.False(t, off.Given(skull, ada))
	assert.Empty(t, off.Tally(ada))
}

func TestAPlayerReactsAsItsAccountAndEveryoneElseAsItsTag(t *testing.T) {
	account := cpsession.AccountID{15: 1}
	player := messages.Author{Username: "ada", Tag: "a1b2c3"}
	guest := messages.Author{Tag: "a1b2c3"}

	assert.Equal(t, messages.Reactor("account:"+account.String()), messages.ReactorOf(account, player))
	assert.Equal(t, messages.Reactor("guest:a1b2c3"), messages.ReactorOf(account, guest), "an account with no username")
	assert.Equal(t, messages.Reactor("guest:a1b2c3"), messages.ReactorOf(cpsession.NoAccount, player), "no token")
}
