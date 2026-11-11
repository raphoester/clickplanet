package ledger_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger"
)

func TestTheRecentHistoryIsTheDayUpToTheLastHalfHourOnTheUTCClock(t *testing.T) {
	nepal := time.FixedZone("NPT", 5*3600+45*60)

	recent := ledger.RecentAt(time.Date(2026, 11, 11, 18, 32, 13, 0, nepal))

	assert.Equal(t, ledger.Recent{
		Since: time.Date(2026, 11, 10, 12, 30, 0, 0, time.UTC),
		Until: time.Date(2026, 11, 11, 12, 30, 0, 0, time.UTC),
		Step:  30 * time.Minute,
	}, recent, "12:47:13 UTC, whatever the zone")
	assert.Equal(t, time.Date(2026, 11, 11, 13, 0, 0, 0, time.UTC), recent.Next())
	assert.Equal(t, time.Date(2026, 11, 11, 13, 0, 0, 0, time.UTC),
		ledger.RecentAt(time.Date(2026, 11, 11, 13, 0, 0, 0, time.UTC)).Until, "a step's own time starts it")
}

func TestAResetInTheRecentHistoryStartsItAtTheFirstStepAtOrAfterIt(t *testing.T) {
	recent := ledger.RecentAt(time.Date(2026, 11, 11, 12, 47, 0, 0, time.UTC))

	assert.Equal(t, recent, recent.Wiped(recent.Since.Add(-time.Minute)))
	assert.Equal(t, recent, recent.Wiped(recent.Since))
	assert.Equal(t, time.Date(2026, 11, 11, 3, 30, 0, 0, time.UTC),
		recent.Wiped(time.Date(2026, 11, 11, 3, 10, 0, 0, time.UTC)).Since)
	assert.Equal(t, time.Date(2026, 11, 11, 3, 30, 0, 0, time.UTC),
		recent.Wiped(time.Date(2026, 11, 11, 3, 30, 0, 0, time.UTC)).Since)
	assert.False(t, recent.Wiped(recent.Until).Empty(), "the map at the end is the blank one")
	assert.True(t, recent.Wiped(recent.Until.Add(time.Second)).Empty(), "the map at the end is gone")
	assert.Equal(t, recent.Until, recent.Wiped(time.Date(2026, 11, 11, 3, 10, 0, 0, time.UTC)).Until)
}

func TestTheLedgerKeepsTheRecentHistoryInPostgresFromItsHead(t *testing.T) {
	require.NoError(t, ledger.Config{}.Validate())
	require.NoError(t, ledger.Config{Retention: 72 * time.Hour}.Validate())
	require.ErrorIs(t, ledger.Config{Retention: 24 * time.Hour}.Validate(), ledger.ErrShortRetention)
}
