package tell_day_usecase_test

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	gamemasterv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/gamemaster/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/moments"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/moments/inmemory_moment_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/moments/outbox_moment_events"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/moments/usecases/tell_day_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/schedule/inmemory_entry_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/season"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/season/inmemory_season_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

const day = 24 * time.Hour

var (
	dayOneEnds = time.Date(2026, 10, 8, 21, 0, 0, 0, time.UTC)
	zero       = season.Snapshot{Number: 0, Length: 23*day + 23*time.Hour}
)

type harness struct {
	t       *testing.T
	clock   *cptime.FixedClock
	seasons *inmemory_season_store.Store
	store   *inmemory_moment_store.Store
	outbox  *cpatomic.FakeOutbox
	useCase *tell_day_usecase.UseCase
}

// Season 0 runs from its origin to 31 Oct 23:00; its day 24 ends at 21:00, and no day runs after it.
func newHarness(t *testing.T, at time.Time) *harness {
	t.Helper()

	h := &harness{
		t: t, clock: cptime.NewFixedClock(at), seasons: inmemory_season_store.New(), store: inmemory_moment_store.New(),
		outbox: &cpatomic.FakeOutbox{},
	}
	require.NoError(t, h.seasons.Save(t.Context(), zero.Restore()))
	planner := moments.NewPlanner(h.seasons, inmemory_entry_store.New(), h.store, season.Cutoff(21*time.Hour))
	h.useCase = tell_day_usecase.New(planner, h.store, outbox_moment_events.New(h.outbox), cpatomic.FakeTransactioner{}, h.clock)
	return h
}

func (h *harness) execute() []moments.DayStarted {
	h.t.Helper()
	told, err := h.useCase.Execute(h.t.Context())
	require.NoError(h.t, err)
	return told
}

func started(number uint32, endsAt time.Time) *gamemasterv1.DayStarted {
	ends := dayOneEnds.Add(time.Duration(number-1) * day)
	return &gamemasterv1.DayStarted{
		Season: 0, Day: number, RoundId: fmt.Sprintf("season-0/day-%d", number),
		BookId: "season-0", StartsAt: timestamppb.New(ends.Add(-day)), EndsAt: timestamppb.New(endsAt),
	}
}

func (h *harness) assertRecorded(want ...*gamemasterv1.DayStarted) {
	h.t.Helper()
	recorded := h.outbox.Recorded()
	require.Len(h.t, recorded, len(want))
	for n := range want {
		assert.True(h.t, proto.Equal(want[n], recorded[n]), "%v", recorded[n])
	}
}

func TestABootTellsTheDayThatRunsOnce(t *testing.T) {
	h := newHarness(t, dayOneEnds.Add(20*day+12*time.Hour))

	assert.Len(t, h.execute(), 1)
	assert.Empty(t, h.execute(), "told once")

	h.assertRecorded(started(22, dayOneEnds.Add(21*day)))
}

func TestTheNextDayIsToldAtTheCutoff(t *testing.T) {
	h := newHarness(t, dayOneEnds.Add(21*day-time.Second))
	h.execute()

	h.clock.Advance(time.Second)
	h.execute()

	h.assertRecorded(started(22, dayOneEnds.Add(21*day)), started(23, dayOneEnds.Add(22*day)))
}

func TestNoDayIsToldAfterTheSeasonsLastCutoff(t *testing.T) {
	h := newHarness(t, dayOneEnds.Add(23*day+time.Hour))

	assert.Empty(t, h.execute())
	h.clock.Advance(2 * day)
	assert.Empty(t, h.execute(), "nor between two seasons")

	h.assertRecorded()
}

func TestADayCutByAMovedSeasonEndIsToldAgainWithItsNewEnd(t *testing.T) {
	h := newHarness(t, dayOneEnds.Add(23*day-9*time.Hour))
	h.execute()
	require.NoError(t, h.store.KeepOpened(t.Context(), "season-0/day-24"))

	moved := zero
	moved.Length = 23*day + 20*time.Hour
	require.NoError(t, h.seasons.Save(t.Context(), moved.Restore()))
	h.execute()

	h.assertRecorded(started(24, dayOneEnds.Add(23*day)), started(24, dayOneEnds.Add(23*day-time.Hour)))
}

func TestADayThatCannotBeKeptIsAnErrorAndToldNot(t *testing.T) {
	h := newHarness(t, dayOneEnds.Add(20*day+12*time.Hour))
	down := errors.New("postgres is down")
	h.store.FailWith(down)

	_, err := h.useCase.Execute(t.Context())

	require.ErrorIs(t, err, down)
	h.assertRecorded()
}
