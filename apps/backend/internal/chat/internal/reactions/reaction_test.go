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

func TestAReactionNobodyGivesAnyMoreLosesItsPlace(t *testing.T) {
	given := reactions.Reactions{}.With(clown, ada).With(laugh, ada).Without(clown, ada).With(clown, bo)

	assert.Equal(t, []reactions.Count{
		reactions.CountOf(laugh, 1, false, gave(ada)),
		reactions.CountOf(clown, 1, false, gave(bo)),
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
		reactions.CountOf(clown, 1, false, gave(ada)),
	}, before.Tally(reactions.NoReactor))
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

func TestACountIsTheAccountsThatGaveItOldestFirst(t *testing.T) {
	one, two := cpsession.AccountID{15: 1}, cpsession.AccountID{15: 2}
	given := reactions.Reactions{}.
		With(clown, reactions.ReactorOf(two)).
		With(clown, reactions.ReactorOf(one)).
		With(clown, bo)

	count := given.Tally(reactions.NoReactor)[0]

	assert.Equal(t, []messages.AccountID{two, one}, count.Accounts(), "a guest tag from before accounts is nobody")
	assert.Equal(t, 3, count.Total(), "the count still says how many gave it")
}
