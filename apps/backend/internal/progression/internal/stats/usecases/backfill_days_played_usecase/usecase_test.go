package backfill_days_played_usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/stats"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/stats/inmemory_stats_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/stats/usecases/backfill_days_played_usecase"
)

var firstDay = time.Date(2026, 9, 20, 12, 0, 0, 0, time.UTC)

func onDay(n int) time.Time { return firstDay.AddDate(0, 0, n) }

func account(i int) accounts.AccountID {
	return accounts.AccountID{14: byte(i >> 8), 15: byte(i)}
}

type fakeLedger struct {
	days  map[accounts.AccountID][]stats.Day
	err   error
	asked [][]accounts.AccountID
	// Runs once planet answered, before the use case takes the lock: a take the relay delivers meanwhile.
	meanwhile func()
}

func (f *fakeLedger) PlayDays(_ context.Context, asked []accounts.AccountID) (map[accounts.AccountID][]stats.Day, error) {
	f.asked = append(f.asked, asked)
	if f.err != nil {
		return nil, f.err
	}
	answer := map[accounts.AccountID][]stats.Day{}
	for _, account := range asked {
		if days, ok := f.days[account]; ok {
			answer[account] = days
		}
	}
	if f.meanwhile != nil {
		f.meanwhile()
		f.meanwhile = nil
	}
	return answer, nil
}

type fixture struct {
	stats    *inmemory_stats_store.Store
	ledger   *fakeLedger
	position *uint64
}

func setUp() fixture {
	return fixture{
		stats:    inmemory_stats_store.New(),
		ledger:   &fakeLedger{days: map[accounts.AccountID][]stats.Day{}},
		position: new(uint64),
	}
}

// A take the live count saw, and the ledger holds.
func (f fixture) take(t *testing.T, i, day int) {
	t.Helper()

	f.liveTake(t, i, day)
	f.ledger.days[account(i)] = append(f.ledger.days[account(i)], stats.DayOf(onDay(day)))
}

// A take the ledger holds and the live count never saw: before the column, or before the ledger had accounts.
func (f fixture) ledgerTake(i, day int) {
	f.ledger.days[account(i)] = append(f.ledger.days[account(i)], stats.DayOf(onDay(day)))
}

func (f fixture) liveTake(t *testing.T, i, day int) {
	t.Helper()

	require.NoError(t, f.stats.Record(t.Context(), account(i), stats.TakeAt(*f.position, 0, onDay(day))))
	*f.position++
}

// The stats as they were when the column was added: counted with no day played.
func (f fixture) before(t *testing.T, i int) {
	t.Helper()

	require.NoError(t, f.stats.Record(t.Context(), account(i), func(s stats.Stats) stats.Stats {
		snapshot := s.TakeSnapshot()
		snapshot.DaysPlayed = 0
		return snapshot.Restore()
	}))
}

func (f fixture) played(t *testing.T, i int) uint64 {
	t.Helper()

	counted, err := f.stats.Stats(t.Context(), account(i))
	require.NoError(t, err)
	return counted.DaysPlayed()
}

func (f fixture) run(t *testing.T, dryRun bool) backfill_days_played_usecase.Backfilled {
	t.Helper()

	backfilled, err := backfill_days_played_usecase.New(f.stats, f.ledger).Execute(t.Context(), dryRun)
	require.NoError(t, err)
	return backfilled
}

func TestEachAccountCountsTheDaysTheLedgerHoldsATakeOn(t *testing.T) {
	f := setUp()
	f.ledgerTake(1, 0)
	f.ledgerTake(1, 3)
	f.take(t, 1, 5)
	f.before(t, 1)
	f.take(t, 2, 0)
	f.before(t, 2)

	backfilled := f.run(t, false)

	assert.Equal(t, backfill_days_played_usecase.Backfilled{Accounts: 2, Changed: 2}, backfilled)
	assert.Equal(t, uint64(3), f.played(t, 1))
	assert.Equal(t, uint64(1), f.played(t, 2))
}

func TestASecondRunChangesNothing(t *testing.T) {
	f := setUp()
	f.ledgerTake(1, 0)
	f.take(t, 1, 2)
	f.before(t, 1)

	f.run(t, false)
	again := f.run(t, false)

	assert.Equal(t, backfill_days_played_usecase.Backfilled{Accounts: 1, Changed: 0}, again)
	assert.Equal(t, uint64(2), f.played(t, 1))
}

func TestADryRunCountsWhatWouldChangeAndWritesNothing(t *testing.T) {
	f := setUp()
	f.ledgerTake(1, 0)
	f.take(t, 1, 2)
	f.before(t, 1)

	backfilled := f.run(t, true)

	assert.Equal(t, backfill_days_played_usecase.Backfilled{Accounts: 1, Changed: 1}, backfilled)
	assert.Zero(t, f.played(t, 1))
}

func TestATakeOnANewDayBetweenThePlanetReadAndTheLockStillCounts(t *testing.T) {
	f := setUp()
	f.ledgerTake(1, 0)
	f.take(t, 1, 2)
	f.before(t, 1)
	f.ledger.meanwhile = func() { f.liveTake(t, 1, 4) }

	f.run(t, false)

	assert.Equal(t, uint64(3), f.played(t, 1), "days 0 and 2 from the ledger, day 4 from the live count, once each")
	assert.Equal(t, backfill_days_played_usecase.Backfilled{Accounts: 1}, f.run(t, false))
}

func TestTheBestStreakIsTheFloor(t *testing.T) {
	f := setUp()
	for day := range 5 {
		f.liveTake(t, 1, day)
	}
	f.before(t, 1)

	f.run(t, false)

	assert.Equal(t, uint64(5), f.played(t, 1), "the ledger holds none of them")
}

func TestEveryPageIsAskedAndCounted(t *testing.T) {
	f := setUp()
	// From 1: account(0) is the zero id, the cursor the first page starts after, and no account.
	for i := 1; i <= 501; i++ {
		f.take(t, i, 0)
		f.before(t, i)
	}

	backfilled := f.run(t, false)

	assert.Equal(t, backfill_days_played_usecase.Backfilled{Accounts: 501, Changed: 501}, backfilled)
	require.Len(t, f.ledger.asked, 2)
	assert.Len(t, f.ledger.asked[0], 500)
	assert.Equal(t, []accounts.AccountID{account(501)}, f.ledger.asked[1])
	assert.Equal(t, uint64(1), f.played(t, 501))
}

func TestNoStatsAsksNothing(t *testing.T) {
	f := setUp()

	assert.Equal(t, backfill_days_played_usecase.Backfilled{}, f.run(t, false))
	assert.Empty(t, f.ledger.asked)
}

func TestAFailureToAskPlanetWritesNothing(t *testing.T) {
	f := setUp()
	f.take(t, 1, 0)
	f.before(t, 1)
	f.ledger.err = errors.New("planet is down")

	_, err := backfill_days_played_usecase.New(f.stats, f.ledger).Execute(t.Context(), false)

	require.ErrorIs(t, err, f.ledger.err)
	assert.Zero(t, f.played(t, 1))
}
