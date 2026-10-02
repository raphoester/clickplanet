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

func gave(reactors ...reactions.Reactor) []reactions.Reactor {
	return reactors
}

func TestReactionsCountEachReactionInTheOrderItFirstAppeared(t *testing.T) {
	given := reactions.Reactions{}.With(clown, ada).With(laugh, bo).With(clown, bo)

	assert.Equal(t, []reactions.Count{
		{Reaction: clown, Count: 2, Mine: true, Reactors: gave(ada, bo)},
		{Reaction: laugh, Count: 1, Mine: false, Reactors: gave(bo)},
	}, given.Tally(ada))
}

func TestAReactorGivesAReactionOnce(t *testing.T) {
	given := reactions.Reactions{}.With(clown, ada).With(clown, ada)

	assert.Equal(t, []reactions.Count{
		{Reaction: clown, Count: 1, Mine: true, Reactors: gave(ada)},
	}, given.Tally(ada))
}

func TestAReactionNobodyGivesAnyMoreLosesItsPlace(t *testing.T) {
	given := reactions.Reactions{}.With(clown, ada).With(laugh, ada).Without(clown, ada).With(clown, bo)

	assert.Equal(t, []reactions.Count{
		{Reaction: laugh, Count: 1, Reactors: gave(ada)},
		{Reaction: clown, Count: 1, Reactors: gave(bo)},
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

	assert.Equal(t, []reactions.Count{
		{Reaction: clown, Count: 1, Reactors: gave(ada)},
	}, before.Tally(reactions.NoReactor))
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

func TestEveryAccountReactsAsItselfAndNoAccountIsNobody(t *testing.T) {
	account := cpsession.AccountID{15: 1}

	assert.Equal(t, reactions.Reactor("account:"+account.String()), reactions.ReactorOf(account))
	assert.Equal(t, reactions.NoReactor, reactions.ReactorOf(cpsession.NoAccount))
}

func TestAReactorSaysWhichAccountItIsAndAGuestTagSaysNone(t *testing.T) {
	account := cpsession.AccountID{15: 1}

	found, isAccount := reactions.AccountOf(reactions.ReactorOf(account))
	assert.True(t, isAccount)
	assert.Equal(t, account, found)

	for _, reactor := range []reactions.Reactor{bo, ada, reactions.NoReactor, "account:not-a-uuid"} {
		_, isAccount := reactions.AccountOf(reactor)
		assert.False(t, isAccount, string(reactor))
	}
}

func TestNamedShowsWhoEachAccountIsNow(t *testing.T) {
	one, two := cpsession.AccountID{15: 1}, cpsession.AccountID{15: 2}
	given := reactions.Reactions{}.With(clown, reactions.ReactorOf(one)).With(clown, reactions.ReactorOf(two))

	named := reactions.Named(given.Tally(reactions.NoReactor), map[messages.AccountID]messages.Author{
		one: {Name: "Ada"},
		two: {Name: "Bo"},
	})

	assert.Equal(t, []string{"Ada", "Bo"}, named[0].Names, "oldest first")
	assert.Equal(t, 2, named[0].Count)
}

func TestNamedCountsWhoItCannotNameWithoutNamingThem(t *testing.T) {
	one, two := cpsession.AccountID{15: 1}, cpsession.AccountID{15: 2}
	given := reactions.Reactions{}.
		With(clown, reactions.ReactorOf(one)).
		With(clown, reactions.ReactorOf(two)).
		With(clown, bo)

	named := reactions.Named(given.Tally(reactions.NoReactor), map[messages.AccountID]messages.Author{
		two: {Name: "Bo"},
	})

	assert.Equal(t, []string{"Bo"}, named[0].Names, "a deleted account, and a guest tag from before accounts")
	assert.Equal(t, 3, named[0].Count, "the count still says how many gave it")
}

func TestAccountsOfIsEveryoneUnderTheCountsOnce(t *testing.T) {
	one, two := cpsession.AccountID{15: 1}, cpsession.AccountID{15: 2}
	given := reactions.Reactions{}.
		With(clown, reactions.ReactorOf(one)).
		With(laugh, reactions.ReactorOf(one)).
		With(laugh, reactions.ReactorOf(two)).
		With(skull, bo)

	assert.Equal(t, []messages.AccountID{one, two},
		reactions.AccountsOf(given.Tally(reactions.NoReactor)),
		"one ask names somebody however often it reacted, and a guest tag is nobody to ask about")
}
