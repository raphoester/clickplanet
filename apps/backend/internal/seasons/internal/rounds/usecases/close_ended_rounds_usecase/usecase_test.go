package close_ended_rounds_usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	seasonsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/seasons/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/calendar"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/rounds"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/rounds/inmemory_round_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/rounds/usecases/close_ended_rounds_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/rounds/usecases/close_round_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
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
	outbox    *cpatomic.FakeOutbox
	clock     *cptime.FixedClock
	useCase   *close_ended_rounds_usecase.UseCase
}

func (f fixture) on(store close_round_usecase.Rounds) *close_ended_rounds_usecase.UseCase {
	closer := close_round_usecase.New(cpatomic.FakeTransactioner{}, store, f.outbox)
	return close_ended_rounds_usecase.New(f.territory, f.store, closer, seasonZero, f.clock)
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
	sixteenth   = rounds.Round{EndsAt: utc(16, 21, 0)}
	seventeenth = rounds.Round{EndsAt: utc(17, 21, 0)}
	finale      = rounds.Round{EndsAt: utc(31, 23, 0), Finale: true}
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
	assert.True(t, proto.Equal(&seasonsv1.RoundClosed{
		Season:  0,
		Number:  2,
		EndedAt: timestamppb.New(sixteenth.EndsAt),
		Results: []*seasonsv1.RoundResult{
			{Country: "fr", Rank: 1, Points: 25},
			{Country: "de", Rank: 2, Points: 18},
		},
	}, published[1]), "got %v", published[1])
}

func TestClosingTheFinaleScoresItThreeTimesAndTellsTheSeasonsTable(t *testing.T) {
	f := newFixture(utc(31, 20, 59))
	f.execute(t)
	f.clock.Advance(time.Minute)
	f.execute(t)
	f.territory.held = map[rounds.Country]uint32{"de": 4, "fr": 1}

	f.clock.Advance(2 * time.Hour)
	report := f.execute(t)

	require.Len(t, report.Closed, 1)
	assert.Equal(t, []rounds.Result{
		{Country: "de", Rank: 1, Points: 75, Share: 0.04},
		{Country: "fr", Rank: 2, Points: 54, Share: 0.01},
	}, report.Closed[0].Results)
	assert.Zero(t, report.Next, "no round follows the last season's finale")
	published := f.outbox.Recorded()
	require.Len(t, published, 3)
	assert.True(t, proto.Equal(&seasonsv1.SeasonClosed{
		Season:  0,
		EndedAt: timestamppb.New(finale.EndsAt),
		Table: []*seasonsv1.SeasonPlace{
			{Country: "de", Rank: 1, Points: 93, RoundsWon: 1, FinalePoints: 75},
			{Country: "fr", Rank: 2, Points: 79, RoundsWon: 1, FinalePoints: 54},
		},
	}, published[2]), "got %v", published[2])
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

func TestAFinaleWhoseSeasonCouldNotBeReadClosesAtTheNextTryAndIsToldOnce(t *testing.T) {
	f := newFixture(utc(31, 22, 0))
	store := &unreadSeasons{Store: f.store}
	useCase := f.on(store)
	_, err := useCase.Execute(t.Context())
	require.NoError(t, err)

	store.err = errors.New("postgres is down")
	f.clock.Advance(time.Hour)
	_, err = useCase.Execute(t.Context())
	require.Error(t, err)
	assert.Empty(t, f.outbox.Recorded())
	assert.Empty(t, f.closedIn(t), "the finale is left to close")

	store.err = nil
	f.clock.Advance(close_ended_rounds_usecase.Retry)
	_, err = useCase.Execute(t.Context())
	require.NoError(t, err)
	f.clock.Advance(close_ended_rounds_usecase.Retry)
	_, err = useCase.Execute(t.Context())
	require.NoError(t, err)

	published := f.outbox.Recorded()
	require.Len(t, published, 2)
	assert.IsType(t, &seasonsv1.RoundClosed{}, published[0])
	assert.IsType(t, &seasonsv1.SeasonClosed{}, published[1])
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
	assert.Equal(t, seventeenth.EndsAt, report.Next, "the round that starts is opened anyway")
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
