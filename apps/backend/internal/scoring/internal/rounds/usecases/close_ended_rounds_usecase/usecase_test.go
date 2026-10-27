package close_ended_rounds_usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	scoringv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/scoring/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/calendar"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/rounds"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/rounds/inmemory_round_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/rounds/usecases/close_ended_rounds_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/rounds/usecases/close_round_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/rounds/usecases/close_season_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

func utc(day, hour, minute int) time.Time {
	return time.Date(2026, 10, day, hour, minute, 0, 0, time.UTC)
}

var seasonZero = calendar.New(calendar.Config{DailyCutoff: 21 * time.Hour}, calendar.Window{Number: 0, StartsAt: time.Date(2026, 10, 3, 0, 0, 0, 0, time.UTC), EndsAt: utc(31, 23, 0)})

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
	outbox    *cpatomic.FakeOutbox
	clock     *cptime.FixedClock
	useCase   *close_ended_rounds_usecase.UseCase
}

type store interface {
	close_round_usecase.Rounds
	close_season_usecase.Seasons
}

func (f fixture) on(store store) *close_ended_rounds_usecase.UseCase {
	closer := close_round_usecase.New(cpatomic.FakeTransactioner{}, store, f.outbox)
	ender := close_season_usecase.New(cpatomic.FakeTransactioner{}, store, f.outbox)
	return close_ended_rounds_usecase.New(f.territory, f.store, closer, ender, seasonZero, f.clock)
}

func newFixture(now time.Time) fixture {
	f := fixture{
		territory: &stubTerritory{held: map[rounds.Country]uint32{"fr": 3, "de": 1}},
		store:     inmemory_round_store.New(),
		outbox:    &cpatomic.FakeOutbox{},
		clock:     cptime.NewFixedClock(now),
	}
	f.useCase = f.on(f.store)
	return f
}

func (f fixture) execute(t *testing.T) close_ended_rounds_usecase.Report {
	t.Helper()
	report, err := f.useCase.Execute(t.Context())
	require.NoError(t, err)
	return report
}

func (f fixture) closedIn(t *testing.T) []rounds.Closed {
	t.Helper()
	closed, err := f.store.ClosedIn(t.Context(), 0)
	require.NoError(t, err)
	return closed
}

var (
	sixteenth   = rounds.Round{EndsAt: utc(16, 21, 0), Times: 1}
	seventeenth = rounds.Round{EndsAt: utc(17, 21, 0), Times: 1}
	battle      = rounds.Round{
		EndsAt: utc(31, 23, 0), Finale: true, Own: true, Entry: rounds.EntryID(uuid.MustParse("0192f5a0-7c00-7000-8000-000000000031")),
		StartsAt: utc(31, 21, 0), Times: 3,
	}
)

func TestARoundIsScoredFromWhatEachCountryHoldsAtItsEnd(t *testing.T) {
	f := newFixture(utc(16, 12, 0))
	f.execute(t)
	f.territory.held = map[rounds.Country]uint32{"de": 4, "fr": 1}

	f.clock.Advance(9 * time.Hour)
	report := f.execute(t)

	results := []rounds.Result{
		{Country: "de", Rank: 1, Points: 25, Share: 0.04},
		{Country: "fr", Rank: 2, Points: 18, Share: 0.01},
	}
	assert.Equal(t, []rounds.Closed{{Round: sixteenth, Number: 1, Results: results}}, report.Closed)
	assert.Equal(t, utc(16, 21, 0), report.ReadAt)
	assert.Equal(t, []rounds.Closed{{Round: sixteenth, Number: 1, Results: results}}, f.closedIn(t))
	assert.Equal(t, 1, f.territory.asked, "the map is read at the end, and only then")
}

func TestARoundIsClosedOnce(t *testing.T) {
	f := newFixture(utc(16, 20, 59))
	f.execute(t)
	f.clock.Advance(time.Minute)
	f.execute(t)

	f.clock.Advance(time.Second)
	report := f.execute(t)

	assert.Empty(t, report.Closed)
	assert.Len(t, f.closedIn(t), 1)
	assert.Len(t, f.outbox.Recorded(), 1)
}

func TestARoundThatEndedWhileTheServerWasDownIsClosedAtBootFromTheMapAsReadThen(t *testing.T) {
	f := newFixture(utc(16, 12, 0))
	f.execute(t)
	f.clock.Advance(21 * time.Hour)
	f.territory.held = map[rounds.Country]uint32{"it": 7}
	booted := f.on(f.store)

	report, err := booted.Execute(t.Context())

	require.NoError(t, err)
	assert.Equal(t, []rounds.Closed{{Round: sixteenth, Number: 1, Results: []rounds.Result{
		{Country: "it", Rank: 1, Points: 25, Share: 0.07},
	}}}, report.Closed)
	assert.Equal(t, utc(17, 9, 0), report.ReadAt)
	assert.Equal(t, seventeenth.EndsAt, report.Next)
}

func TestClosingARoundTellsTheOtherModulesItsResults(t *testing.T) {
	f := newFixture(utc(15, 20, 59))
	f.execute(t)
	f.clock.Advance(time.Minute)
	f.execute(t)

	f.clock.Advance(24 * time.Hour)
	f.execute(t)

	published := f.outbox.Recorded()
	require.Len(t, published, 2)
	assert.True(t, proto.Equal(&scoringv1.RoundClosed{
		Season:  0,
		Number:  2,
		EndedAt: timestamppb.New(sixteenth.EndsAt),
		Results: []*scoringv1.RoundResult{
			{Country: "fr", Rank: 1, Points: 25},
			{Country: "de", Rank: 2, Points: 18},
		},
		Times: 1,
	}, published[1]), "got %v", published[1])
}

func TestARoundOfItsOwnClosesAtItsEndAndTheSeasonEndsWithItsTable(t *testing.T) {
	f := newFixture(utc(31, 20, 59))
	f.execute(t)
	f.clock.Advance(time.Minute)
	f.execute(t)
	require.NoError(t, f.store.Open(t.Context(), battle))
	f.territory.held = map[rounds.Country]uint32{"de": 4, "fr": 1}

	f.clock.Advance(2 * time.Hour)
	report := f.execute(t)

	require.Len(t, report.Closed, 1)
	assert.Equal(t, battle, report.Closed[0].Round)
	assert.Equal(t, []rounds.Result{
		{Country: "de", Rank: 1, Points: 75, Share: 0.04},
		{Country: "fr", Rank: 2, Points: 54, Share: 0.01},
	}, report.Closed[0].Results)
	require.Len(t, report.Ended, 1)
	assert.Zero(t, report.Next, "nothing is left to end after the last season")
	published := f.outbox.Recorded()
	require.Len(t, published, 3)
	assert.True(t, proto.Equal(&scoringv1.RoundClosed{
		Season:  0,
		Number:  1,
		Finale:  true,
		EndedAt: timestamppb.New(battle.EndsAt),
		Results: []*scoringv1.RoundResult{{Country: "de", Rank: 1, Points: 75}, {Country: "fr", Rank: 2, Points: 54}},
		Own:     true,
		Times:   3,
		EntryId: "0192f5a0-7c00-7000-8000-000000000031",
	}, published[1]), "got %v", published[1])
	assert.True(t, proto.Equal(&scoringv1.SeasonClosed{
		Season:  0,
		EndedAt: timestamppb.New(battle.EndsAt),
		Table: []*scoringv1.SeasonPlace{
			{Country: "de", Rank: 1, Points: 93, RoundsWon: 1, FinalePoints: 75},
			{Country: "fr", Rank: 2, Points: 79, RoundsWon: 1, FinalePoints: 54},
		},
	}, published[2]), "got %v", published[2])
}

func TestASeasonWithNoRoundAtItsEndEndsAtItsEnd(t *testing.T) {
	f := newFixture(utc(31, 20, 59))
	f.execute(t)

	f.clock.Advance(time.Minute)
	report := f.execute(t)
	assert.Equal(t, battle.EndsAt, report.Next, "no day runs to the season's end: the runner wakes for it")
	assert.Empty(t, report.Ended)

	f.clock.Advance(2 * time.Hour)
	report = f.execute(t)

	require.Len(t, report.Ended, 1)
	assert.Equal(t, rounds.SeasonEnd{
		Season: 0, EndedAt: battle.EndsAt,
		Table: []rounds.Placed{
			{Rank: 1, Score: rounds.Score{Country: "fr", Points: 25, RoundsWon: 1}},
			{Rank: 2, Score: rounds.Score{Country: "de", Points: 18}},
		},
	}, report.Ended[0])
	published := f.outbox.Recorded()
	require.Len(t, published, 2)
	assert.IsType(t, &scoringv1.SeasonClosed{}, published[1])
}

func TestASeasonEndsOnce(t *testing.T) {
	f := newFixture(utc(31, 22, 0))
	f.clock.Advance(time.Hour)
	f.execute(t)

	f.clock.Advance(time.Hour)
	report := f.execute(t)

	assert.Empty(t, report.Ended)
	assert.Len(t, f.outbox.Recorded(), 1)
}

func TestTheRunnerWakesAtTheFirstEndOfARoundOrOfTheSeason(t *testing.T) {
	f := newFixture(utc(20, 12, 0))
	assert.Equal(t, utc(20, 21, 0), f.execute(t).Next, "the day")

	require.NoError(t, f.store.Open(t.Context(), rounds.Round{EndsAt: utc(20, 19, 0), Own: true, StartsAt: utc(20, 17, 0), Times: 2}))
	assert.Equal(t, utc(20, 19, 0), f.execute(t).Next, "a round of its own that ends first")
}

type unreadSeasons struct {
	*inmemory_round_store.Store

	err error
}

func (u *unreadSeasons) ClosedIn(ctx context.Context, season calendar.Number) ([]rounds.Closed, error) {
	if u.err != nil {
		return nil, u.err
	}
	return u.Store.ClosedIn(ctx, season) //nolint:wrapcheck // the fake fails or is the store.
}

func TestASeasonWhoseRoundsCouldNotBeReadEndsAtTheNextTryAndIsToldOnce(t *testing.T) {
	f := newFixture(utc(31, 22, 0))
	store := &unreadSeasons{Store: f.store}
	useCase := f.on(store)
	require.NoError(t, f.store.Open(t.Context(), battle))

	store.err = errors.New("postgres is down")
	f.clock.Advance(time.Hour)
	_, err := useCase.Execute(t.Context())
	require.Error(t, err)
	require.Len(t, f.outbox.Recorded(), 1, "the round closed")
	ended, err := f.store.Ended(t.Context(), 0)
	require.NoError(t, err)
	assert.False(t, ended, "the season is left to end")

	store.err = nil
	f.clock.Advance(close_ended_rounds_usecase.Retry)
	_, err = useCase.Execute(t.Context())
	require.NoError(t, err)
	f.clock.Advance(close_ended_rounds_usecase.Retry)
	_, err = useCase.Execute(t.Context())
	require.NoError(t, err)

	published := f.outbox.Recorded()
	require.Len(t, published, 2)
	assert.IsType(t, &scoringv1.RoundClosed{}, published[0])
	assert.IsType(t, &scoringv1.SeasonClosed{}, published[1])
}

func TestAMapThatCannotBeReadIsAnErrorAndTheRoundIsLeftToClose(t *testing.T) {
	f := newFixture(utc(16, 20, 59))
	f.execute(t)
	unreachable := errors.New("planet is down")
	f.territory.err = unreachable

	f.clock.Advance(time.Minute)
	report, err := f.useCase.Execute(t.Context())

	require.ErrorIs(t, err, unreachable)
	assert.Empty(t, report.Closed)
	assert.Equal(t, sixteenth.EndsAt, report.Next, "the round left to close ends first")
	assert.Empty(t, f.closedIn(t))

	f.territory.err = nil
	f.clock.Advance(close_ended_rounds_usecase.Retry)
	assert.Len(t, f.execute(t).Closed, 1)
}

func TestARoundThatFailedToCloseTellsNothing(t *testing.T) {
	f := newFixture(utc(16, 20, 59))
	f.execute(t)
	f.store.FailWith(errors.New("postgres is down"))

	f.clock.Advance(time.Minute)
	_, err := f.useCase.Execute(t.Context())

	require.Error(t, err)
	assert.Empty(t, f.outbox.Recorded())
}

func TestAStoreThatFailsIsAnError(t *testing.T) {
	f := newFixture(utc(16, 12, 0))
	down := errors.New("postgres is down")
	f.store.FailWith(down)

	_, err := f.useCase.Execute(t.Context())

	require.ErrorIs(t, err, down)
}
