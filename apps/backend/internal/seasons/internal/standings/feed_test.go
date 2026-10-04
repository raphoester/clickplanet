package standings_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/standings"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
)

var (
	ada  = standings.AccountID{15: 1}
	noon = time.Date(2026, 10, 16, 12, 0, 0, 0, time.UTC)
)

func entry(position standings.Position, account standings.AccountID, country standings.Country, reverted bool) standings.Entry {
	return standings.EntryOf(position, standings.Take{Account: account, Country: country, At: noon}, reverted)
}

func TestOnlyATakeOfATileByAnAccountThatWasNotRevertedCounts(t *testing.T) {
	assert.True(t, entry(0, ada, "fr", false).Countable())
	assert.False(t, entry(0, cpsession.NoAccount, "fr", false).Countable(), "no account, or a deleted one")
	assert.False(t, entry(0, ada, "", false).Countable(), "a clear of native land takes no tile")
	assert.False(t, entry(0, ada, "fr", true).Countable(), "reverted before it was read")
}

func TestABatchCountsItsTakesThatCountInPositionOrderAndReadsOnPastAll(t *testing.T) {
	batch, err := standings.BatchOf(3, 9, []standings.Entry{
		entry(3, ada, "fr", false),
		entry(4, ada, "", false),
		entry(6, ada, "de", false),
	})

	require.NoError(t, err)
	assert.Equal(t, []standings.Take{
		{Account: ada, Country: "fr", At: noon},
		{Account: ada, Country: "de", At: noon},
	}, batch.Takes())
	assert.Equal(t, standings.Position(9), batch.Next(), "past the entries of other kinds read after the last take")
	assert.False(t, batch.Empty())

	nothing, err := standings.BatchOf(9, 9, nil)
	require.NoError(t, err)
	assert.True(t, nothing.Empty())
}

func TestABatchOutOfOrderOrOutsideWhatWasReadIsRefused(t *testing.T) {
	for name, c := range map[string]struct {
		from, next standings.Position
		entries    []standings.Entry
	}{
		"before where it was asked":   {from: 5, next: 6, entries: []standings.Entry{entry(4, ada, "fr", false)}},
		"twice at one position":       {from: 0, next: 3, entries: []standings.Entry{entry(2, ada, "fr", false), entry(2, ada, "fr", false)}},
		"at or past what was read":    {from: 0, next: 2, entries: []standings.Entry{entry(2, ada, "fr", false)}},
		"read up to before the start": {from: 4, next: 3},
	} {
		_, err := standings.BatchOf(c.from, c.next, c.entries)
		assert.ErrorIs(t, err, standings.ErrDisordered, name)
	}
}
