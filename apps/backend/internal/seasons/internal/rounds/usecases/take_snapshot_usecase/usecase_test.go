package take_snapshot_usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/calendar"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/rounds"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/rounds/inmemory_round_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/rounds/usecases/take_snapshot_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

func utc(day, hour, minute int) time.Time {
	return time.Date(2026, 10, day, hour, minute, 0, 0, time.UTC)
}

var seasonZero = calendar.New(calendar.Config{List: []calendar.Entry{
	{Number: 0, EndsAt: utc(31, 23, 0), Finale: 2 * time.Hour},
}})

type stubTerritory struct {
	held  map[rounds.Country]uint32
	err   error
	asked int
}

func (s *stubTerritory) Snapshot(context.Context) (rounds.Snapshot, error) {
	s.asked++
	return rounds.Snapshot{Tiles: 100, Held: s.held}, s.err
}

type fixture struct {
	territory *stubTerritory
	store     *inmemory_round_store.Store
	clock     *cptime.FixedClock
	useCase   *take_snapshot_usecase.UseCase
}

func newFixture(now time.Time) fixture {
	territory := &stubTerritory{held: map[rounds.Country]uint32{"fr": 3, "de": 1}}
	store := inmemory_round_store.New()
	clock := cptime.NewFixedClock(now)
	return fixture{
		territory: territory,
		store:     store,
		clock:     clock,
		useCase:   take_snapshot_usecase.New(territory, store, seasonZero, clock),
	}
}

func (f fixture) held(t *testing.T, round rounds.Round) map[rounds.Country]uint64 {
	t.Helper()
	held, err := f.store.Held(t.Context(), round)
	require.NoError(t, err)
	return held
}

var (
	sixteenth = rounds.Round{EndsAt: utc(16, 21, 0)}
	finale    = rounds.Round{EndsAt: utc(31, 23, 0), Finale: true}
)

func TestASnapshotAddsWhatEachCountryHoldsToTheRoundInProgress(t *testing.T) {
	f := newFixture(utc(16, 12, 0))

	_, err := f.useCase.Execute(t.Context())
	require.NoError(t, err)
	f.clock.Advance(time.Minute)
	_, err = f.useCase.Execute(t.Context())
	require.NoError(t, err)

	assert.Equal(t, map[rounds.Country]uint64{"fr": 6, "de": 2}, f.held(t, sixteenth))
}

func TestTheFirstSnapshotAfterARoundEndsClosesItWithItsResults(t *testing.T) {
	f := newFixture(utc(16, 20, 59))
	_, err := f.useCase.Execute(t.Context())
	require.NoError(t, err)

	f.clock.Advance(time.Minute)
	closed, err := f.useCase.Execute(t.Context())

	require.NoError(t, err)
	assert.Equal(t, []rounds.Round{sixteenth}, closed)
	assert.Equal(t, []rounds.Result{
		{Country: "fr", Rank: 1, Points: 25},
		{Country: "de", Rank: 2, Points: 18},
	}, f.store.Results(sixteenth))
	assert.Equal(t, map[rounds.Country]uint64{"fr": 3, "de": 1}, f.held(t, rounds.Round{EndsAt: utc(17, 21, 0)}),
		"the snapshot at the end counts for the round that starts")
}

func TestAfterTheLastSeasonNothingIsCountedAndTheFinaleStillCloses(t *testing.T) {
	f := newFixture(utc(31, 22, 0))
	_, err := f.useCase.Execute(t.Context())
	require.NoError(t, err)

	f.clock.Advance(2 * time.Hour)
	closed, err := f.useCase.Execute(t.Context())

	require.NoError(t, err)
	assert.Equal(t, []rounds.Round{finale}, closed)
	assert.Equal(t, []rounds.Result{
		{Country: "fr", Rank: 1, Points: 75},
		{Country: "de", Rank: 2, Points: 54},
	}, f.store.Results(finale))
	assert.Equal(t, 1, f.territory.asked)
}

func TestATerritoryThatCannotBeReadIsAnErrorAndTheEndedRoundsStillClose(t *testing.T) {
	f := newFixture(utc(16, 20, 59))
	_, err := f.useCase.Execute(t.Context())
	require.NoError(t, err)
	unreachable := errors.New("planet is down")
	f.territory.err = unreachable

	f.clock.Advance(time.Minute)
	closed, err := f.useCase.Execute(t.Context())

	require.ErrorIs(t, err, unreachable)
	assert.Equal(t, []rounds.Round{sixteenth}, closed)
	assert.Empty(t, f.held(t, rounds.Round{EndsAt: utc(17, 21, 0)}))
}

func TestAStoreThatFailsIsAnError(t *testing.T) {
	f := newFixture(utc(16, 12, 0))
	down := errors.New("postgres is down")
	f.store.FailWith(down)

	_, err := f.useCase.Execute(t.Context())

	require.ErrorIs(t, err, down)
}
