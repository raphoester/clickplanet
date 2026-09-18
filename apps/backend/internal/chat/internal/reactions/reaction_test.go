package reactions_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/reactions"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
)

const (
	laugh reactions.Reaction = 1
	clown reactions.Reaction = 2
	skull reactions.Reaction = 3

	ada reactions.Reactor = "account:ada"
	bo  reactions.Reactor = "guest:91aa3d"
)

func TestReactionsCountEachReactionInTheOrderItFirstAppeared(t *testing.T) {
	given := reactions.Reactions{}.With(clown, ada).With(laugh, bo).With(clown, bo)

	assert.Equal(t, []reactions.Count{
		{Reaction: clown, Count: 2, Mine: true},
		{Reaction: laugh, Count: 1, Mine: false},
	}, given.Tally(ada))
}

func TestAReactorGivesAReactionOnce(t *testing.T) {
	given := reactions.Reactions{}.With(clown, ada).With(clown, ada)

	assert.Equal(t, []reactions.Count{{Reaction: clown, Count: 1, Mine: true}}, given.Tally(ada))
}

func TestAReactionNobodyGivesAnyMoreLosesItsPlace(t *testing.T) {
	given := reactions.Reactions{}.With(clown, ada).With(laugh, ada).Without(clown, ada).With(clown, bo)

	assert.Equal(t, []reactions.Count{
		{Reaction: laugh, Count: 1},
		{Reaction: clown, Count: 1},
	}, given.Tally(reactions.NoReactor))
}

func TestTakingOffAReactionNotGivenChangesNothing(t *testing.T) {
	given := reactions.Reactions{}.With(clown, ada)

	assert.Equal(t, given, given.Without(clown, bo))
	assert.Equal(t, given, given.Without(skull, ada))
}

func TestWithAndWithoutLeaveTheReceiverAsItWas(t *testing.T) {
	before := reactions.Reactions{}.With(clown, ada)

	_ = before.With(clown, bo)
	_ = before.With(laugh, bo)
	_ = before.Without(clown, ada)

	assert.Equal(t, []reactions.Count{{Reaction: clown, Count: 1}}, before.Tally(reactions.NoReactor))
}

func TestNoReactorOwnsNothing(t *testing.T) {
	assert.False(t, reactions.Reactions{}.With(clown, ada).Tally(reactions.NoReactor)[0].Mine)
}

func TestAppliedPutsOnAndTakesOff(t *testing.T) {
	on := reactions.Reactions{}.Applied(reactions.Change{Reaction: skull, Reactor: ada, On: true})
	off := on.Applied(reactions.Change{Reaction: skull, Reactor: ada, On: false})

	assert.True(t, on.Given(skull, ada))
	assert.False(t, off.Given(skull, ada))
	assert.Empty(t, off.Tally(ada))
}

func TestAPlayerReactsAsItsAccountAndEveryoneElseAsItsTag(t *testing.T) {
	account := cpsession.AccountID{15: 1}
	player := messages.Author{Username: "ada", Tag: "a1b2c3"}
	guest := messages.Author{Tag: "a1b2c3"}

	assert.Equal(t, reactions.Reactor("account:"+account.String()), reactions.ReactorOf(account, player))
	assert.Equal(t, reactions.Reactor("guest:a1b2c3"), reactions.ReactorOf(account, guest), "an account with no username")
	assert.Equal(t, reactions.Reactor("guest:a1b2c3"), reactions.ReactorOf(cpsession.NoAccount, player), "no token")
}
