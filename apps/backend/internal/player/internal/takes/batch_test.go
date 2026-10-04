package takes_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/takes"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
)

var (
	ada    = players.AccountID{15: 1}
	bob    = players.AccountID{15: 2}
	monday = time.Date(2026, 10, 5, 12, 0, 0, 0, time.UTC)
)

func batch(t *testing.T, from takes.Position, list ...takes.Take) takes.Batch {
	t.Helper()

	next := from
	if len(list) > 0 {
		next = list[len(list)-1].Position() + 1
	}
	batch, err := takes.BatchOf(from, next, list)
	require.NoError(t, err)
	return batch
}

func TestOnlyATakeOfATileByAnAccountThatWasNotRevertedCounts(t *testing.T) {
	assert.True(t, takes.TakeOf(0, ada, "fr", monday, false).Countable())
	assert.False(t, takes.TakeOf(0, cpsession.NoAccount, "fr", monday, false).Countable(), "no account, or a deleted one")
	assert.False(t, takes.TakeOf(0, ada, "", monday, false).Countable(), "a clear of native land takes no tile")
	assert.False(t, takes.TakeOf(0, ada, "fr", monday, true).Countable(), "reverted before it was read")
}

func TestABatchReadsOnFromWhatItRead(t *testing.T) {
	assert.Equal(t, takes.Position(9), batch(t, 3, takes.TakeOf(4, ada, "fr", monday, false), takes.TakeOf(8, ada, "", monday, false)).Next(),
		"a take that does not count still moves the position")
	assert.Equal(t, takes.Position(3), batch(t, 3).Next(), "nothing read stays where it was asked")
	assert.True(t, batch(t, 3).Empty())
}

func TestEntriesOfAKindThatIsNotATakeAreReadPast(t *testing.T) {
	onlyOthers, err := takes.BatchOf(3, 6, nil)
	require.NoError(t, err)
	assert.False(t, onlyOthers.Empty(), "something was read")
	assert.Zero(t, onlyOthers.Len())
	assert.Equal(t, takes.Position(6), onlyOthers.Next())

	afterTheLastTake, err := takes.BatchOf(3, 9, []takes.Take{takes.TakeOf(4, ada, "fr", monday, false)})
	require.NoError(t, err)
	assert.Equal(t, takes.Position(9), afterTheLastTake.Next())
}

func TestABatchOutOfOrderOrOutsideWhatWasReadIsRefused(t *testing.T) {
	for name, c := range map[string]struct {
		from, next takes.Position
		list       []takes.Take
	}{
		"before where it was asked":   {from: 5, next: 6, list: []takes.Take{takes.TakeOf(4, ada, "fr", monday, false)}},
		"twice at one position":       {from: 0, next: 3, list: []takes.Take{takes.TakeOf(2, ada, "fr", monday, false), takes.TakeOf(2, ada, "fr", monday, false)}},
		"at or past what was read":    {from: 0, next: 2, list: []takes.Take{takes.TakeOf(2, ada, "fr", monday, false)}},
		"read up to before the start": {from: 4, next: 3},
	} {
		_, err := takes.BatchOf(c.from, c.next, c.list)
		assert.ErrorIs(t, err, takes.ErrDisordered, name)
	}
}

func TestTheAccountsAreThoseThatCountOnceEachInAccountOrder(t *testing.T) {
	accounts := batch(t, 0,
		takes.TakeOf(0, bob, "fr", monday, false),
		takes.TakeOf(1, ada, "fr", monday, false),
		takes.TakeOf(2, bob, "fr", monday, false),
		takes.TakeOf(3, players.AccountID{15: 3}, "fr", monday, true),
	).Accounts()

	assert.Equal(t, []players.AccountID{ada, bob}, accounts)
}

func TestATallyAppliesTheDomainsRuleInPositionOrderOnTheStatsKept(t *testing.T) {
	kept := players.NewStats(ada).WithTake(monday.Add(-24 * time.Hour)).WithMessage()

	tallied := batch(t, 0,
		takes.TakeOf(0, ada, "fr", monday, false),
		takes.TakeOf(1, bob, "de", monday, false),
		takes.TakeOf(2, ada, "", monday, false),
		takes.TakeOf(3, ada, "fr", monday.Add(24*time.Hour), false),
		takes.TakeOf(4, bob, "de", monday, true),
	).Tallied(map[players.AccountID]players.Stats{ada: kept})

	assert.Equal(t, []players.Stats{
		kept.WithTake(monday).WithTake(monday.Add(24 * time.Hour)),
		players.NewStats(bob).WithTake(monday),
	}, tallied)
	assert.Equal(t, uint32(3), tallied[0].Streak().Days())
}

func TestABatchWithNothingThatCountsTalliesNothing(t *testing.T) {
	assert.Empty(t, batch(t, 0, takes.TakeOf(0, cpsession.NoAccount, "fr", monday, false)).Tallied(map[players.AccountID]players.Stats{}))
}
