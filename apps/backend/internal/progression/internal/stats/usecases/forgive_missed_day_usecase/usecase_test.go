package forgive_missed_day_usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	progressionv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/progression/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/stats"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/stats/inmemory_stats_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/stats/usecases/forgive_missed_day_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/stats/usecases/record_stats_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpredis"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var (
	ada   = accounts.AccountID{15: 1}
	noon  = time.Date(2026, 11, 17, 12, 0, 0, 0, time.UTC)
	today = stats.DayOf(noon)
)

func daysAgo(n int) stats.Day { return stats.DayOf(noon.AddDate(0, 0, -n)) }

type fakeLedger struct {
	days  map[accounts.AccountID][]stats.Day
	err   error
	asked [][]accounts.AccountID
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
	return answer, nil
}

type fixture struct {
	stats    *inmemory_stats_store.Store
	ledger   *fakeLedger
	events   *cpredis.RecordedEvents
	useCase  *forgive_missed_day_usecase.UseCase
	position *uint64
}

func setUp() fixture {
	store, ledger, events := inmemory_stats_store.New(), &fakeLedger{days: map[accounts.AccountID][]stats.Day{}}, cpredis.NewRecordedEvents()
	return fixture{
		stats:    store,
		ledger:   ledger,
		events:   events,
		useCase:  forgive_missed_day_usecase.New(store, ledger, record_stats_usecase.New(store, events), cptime.NewFixedClock(noon)),
		position: new(uint64),
	}
}

// A take counted live, and the ledger that holds it.
func (f fixture) playedDaysAgo(t *testing.T, ago ...int) {
	t.Helper()

	for _, n := range ago {
		require.NoError(t, f.stats.Record(t.Context(), ada, stats.TakeAt(*f.position, 0, noon.AddDate(0, 0, -n))))
		*f.position++
		f.ledger.days[ada] = append(f.ledger.days[ada], daysAgo(n))
	}
}

func (f fixture) kept(t *testing.T) stats.Stats {
	t.Helper()

	counted, err := f.stats.Stats(t.Context(), ada)
	require.NoError(t, err)
	return counted
}

func TestAForgivenGapJoinsTheRunsAndSaysTheStatsChanged(t *testing.T) {
	f := setUp()
	f.playedDaysAgo(t, 7, 6, 5, 4, 3, 1, 0)

	forgiven, err := f.useCase.Execute(t.Context(), forgive_missed_day_usecase.In{Account: ada, Day: daysAgo(2)})

	require.NoError(t, err)
	assert.Equal(t, stats.StreakOf(2, today), forgiven.Before.Streak())
	assert.Equal(t, stats.StreakOf(8, today), forgiven.After.Streak())
	assert.Equal(t, uint32(8), forgiven.After.StreakBest())
	assert.Equal(t, uint64(7), forgiven.After.DaysPlayed(), "a forgiven day is no day played")
	assert.Equal(t, forgiven.After, f.kept(t))
	assert.Equal(t, [][]accounts.AccountID{{ada}}, f.ledger.asked, "planet is asked the one account")
	require.Len(t, f.events.Published(), 1)
	assert.True(t, proto.Equal(&progressionv1.StatsChanged{AccountId: ada.String()}, f.events.Published()[0]),
		"titles, levels and badges follow")
}

func TestYesterdayForgivenIsTheLastDay(t *testing.T) {
	f := setUp()
	f.playedDaysAgo(t, 4, 3, 2)

	forgiven, err := f.useCase.Execute(t.Context(), forgive_missed_day_usecase.In{Account: ada, Day: daysAgo(1)})

	require.NoError(t, err)
	assert.Equal(t, stats.StreakOf(4, daysAgo(1)), f.kept(t).Streak())
	assert.Equal(t, f.kept(t), forgiven.After)
}

func TestADryRunAnswersWhatWouldChangeAndWritesNothing(t *testing.T) {
	f := setUp()
	f.playedDaysAgo(t, 4, 3, 1, 0)
	before := f.kept(t)

	forgiven, err := f.useCase.Execute(t.Context(), forgive_missed_day_usecase.In{Account: ada, Day: daysAgo(2), DryRun: true})

	require.NoError(t, err)
	assert.Equal(t, stats.StreakOf(5, today), forgiven.After.Streak())
	assert.Equal(t, before, f.kept(t))
	assert.Empty(t, f.events.Published())
}

func TestARefusedDayWritesNothing(t *testing.T) {
	for name, refused := range map[string]struct {
		day stats.Day
		err error
	}{
		"today, by the clock":   {day: today, err: stats.ErrNotPast},
		"a day played":          {day: daysAgo(3), err: stats.ErrDayPlayed},
		"a day that joins none": {day: daysAgo(6), err: stats.ErrJoinsNothing},
	} {
		t.Run(name, func(t *testing.T) {
			f := setUp()
			f.playedDaysAgo(t, 4, 3, 1, 0)
			before := f.kept(t)

			forgiven, err := f.useCase.Execute(t.Context(), forgive_missed_day_usecase.In{Account: ada, Day: refused.day})

			require.ErrorIs(t, err, refused.err)
			assert.Equal(t, before, forgiven.Before)
			assert.Equal(t, before, f.kept(t))
			assert.Empty(t, f.events.Published())
		})
	}
}

func TestADayForgivenTwiceIsRefusedTheSecondTime(t *testing.T) {
	f := setUp()
	f.playedDaysAgo(t, 4, 3, 1, 0)
	_, err := f.useCase.Execute(t.Context(), forgive_missed_day_usecase.In{Account: ada, Day: daysAgo(2)})
	require.NoError(t, err)

	_, err = f.useCase.Execute(t.Context(), forgive_missed_day_usecase.In{Account: ada, Day: daysAgo(2)})

	require.ErrorIs(t, err, stats.ErrDayForgiven)
	assert.Equal(t, []stats.Day{daysAgo(2)}, f.kept(t).ForgivenDays())
	assert.Len(t, f.events.Published(), 1)
}

func TestAnAccountWithNoStatsIsRefusedAndPlanetNotAsked(t *testing.T) {
	f := setUp()

	_, err := f.useCase.Execute(t.Context(), forgive_missed_day_usecase.In{Account: ada, Day: daysAgo(1)})

	require.ErrorIs(t, err, stats.ErrNoStats)
	assert.Empty(t, f.ledger.asked)
}

func TestAFailureToAskPlanetWritesNothing(t *testing.T) {
	f := setUp()
	f.playedDaysAgo(t, 4, 3, 1, 0)
	before := f.kept(t)
	f.ledger.err = errors.New("planet is down")

	_, err := f.useCase.Execute(t.Context(), forgive_missed_day_usecase.In{Account: ada, Day: daysAgo(2)})

	require.ErrorIs(t, err, f.ledger.err)
	assert.Equal(t, before, f.kept(t))
	assert.Empty(t, f.events.Published())
}
