package ledger_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger"
)

var (
	historySince = time.Date(2026, 10, 8, 0, 0, 0, 0, time.UTC)
	historyUntil = historySince.Add(90 * time.Minute)
)

func record(t *testing.T, history *ledger.History, events ...ledger.Event) {
	t.Helper()
	for _, event := range events {
		entry, err := event.Entry()
		require.NoError(t, err)
		require.False(t, history.Done(entry))
		require.NoError(t, history.Record(entry))
	}
}

func take(tile uint32, country string, at time.Time) ledger.Taking {
	return ledger.Taking{Tile: tile, Scope: "203.0.113.7", Country: country, At: at}
}

func TestAHistoryOpensOnEveryTileHeldThenKeepsWhatChangedAtEachStep(t *testing.T) {
	history, err := ledger.NewHistory(historySince, historyUntil, time.Hour)
	require.NoError(t, err)

	record(t, history,
		take(1, "fr", historySince.Add(-48*time.Hour)),
		take(2, "de", historySince.Add(-time.Hour)),
		take(1, "de", historySince.Add(10*time.Minute)),
		take(1, "fr", historySince.Add(20*time.Minute)),
		take(3, "jp", historySince.Add(time.Hour)),
		ledger.Bombing{Scope: "203.0.113.7", At: historySince.Add(80 * time.Minute), Blast: clicks.Blast{
			Tile: 2, CountryID: "fr", Cleared: []uint32{2}, Owners: []string{"de"},
		}},
	)

	assert.Equal(t, []ledger.Frame{
		{At: historySince, Owners: map[uint32]string{1: "fr", 2: "de"}},
		{At: historySince.Add(time.Hour), Owners: map[uint32]string{}},
		{At: historyUntil, Owners: map[uint32]string{2: "", 3: "jp"}},
	}, history.Frames(), "tile 1 went back to France within the hour; a take at a step's time is in the next frame")
}

func TestAHistoryCountsEachTilesChangesAndShieldHitsDuringTheWindowOnly(t *testing.T) {
	history, err := ledger.NewHistory(historySince, historyUntil, time.Hour)
	require.NoError(t, err)

	record(t, history,
		take(1, "fr", historySince.Add(-time.Minute)),
		take(1, "de", historySince.Add(time.Minute)),
		take(1, "fr", historySince.Add(2*time.Minute)),
		ledger.Striking{Tile: 4, Scope: "203.0.113.7", Country: "fr", Owner: "de", Shields: 1, At: historySince.Add(3 * time.Minute)},
	)

	assert.Equal(t, map[uint32]uint32{1: 2, 4: 1}, history.Hits())
}

func TestAWipeAtTheWindowsStartOpensItOnAnEmptyMap(t *testing.T) {
	history, err := ledger.NewHistory(historySince, historyUntil, time.Hour)
	require.NoError(t, err)

	record(t, history, take(1, "fr", historySince.Add(-time.Hour)))
	history.Wipe(historySince)
	record(t, history, take(2, "de", historySince.Add(time.Minute)))

	assert.Equal(t, []ledger.Frame{
		{At: historySince, Owners: map[uint32]string{}},
		{At: historySince.Add(time.Hour), Owners: map[uint32]string{2: "de"}},
		{At: historyUntil, Owners: map[uint32]string{}},
	}, history.Frames())
}

func TestAHistoryIsDoneWithTheEntriesFromItsEnd(t *testing.T) {
	history, err := ledger.NewHistory(historySince, historyUntil, time.Hour)
	require.NoError(t, err)

	entry, err := take(1, "fr", historyUntil).Entry()
	require.NoError(t, err)
	assert.True(t, history.Done(entry))
}

func TestAHistoryStartsBeforeItEndsAndMovesByAStep(t *testing.T) {
	_, err := ledger.NewHistory(historyUntil, historySince, time.Hour)
	require.ErrorIs(t, err, ledger.ErrInvalidHistory)
	_, err = ledger.NewHistory(historySince, historyUntil, 0)
	require.ErrorIs(t, err, ledger.ErrInvalidHistory)
}
