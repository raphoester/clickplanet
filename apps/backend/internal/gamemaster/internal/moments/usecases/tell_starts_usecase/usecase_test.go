package tell_starts_usecase_test

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	gamemasterv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/gamemaster/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/moments"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/moments/inmemory_moment_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/moments/outbox_moment_events"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/moments/usecases/tell_starts_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/schedule"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/schedule/inmemory_entry_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/season"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/season/inmemory_season_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/sieges/inmemory_siege_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

const day = 24 * time.Hour

var (
	dayOneEnds      = time.Date(2026, 10, 8, 21, 0, 0, 0, time.UTC)
	seasonZeroEnds  = time.Date(2026, 10, 31, 23, 0, 0, 0, time.UTC)
	finaleStarts    = time.Date(2026, 10, 31, 21, 0, 0, 0, time.UTC)
	seasonOneStarts = time.Date(2026, 11, 1, 15, 0, 0, 0, time.UTC)
	battleStarts    = time.Date(2026, 10, 27, 18, 0, 0, 0, time.UTC)
	battleEnds      = time.Date(2026, 10, 27, 21, 0, 0, 0, time.UTC)
	zero            = season.Snapshot{Number: 0, Length: 23*day + 23*time.Hour}
	one             = season.Snapshot{Number: 1, Break: 16 * time.Hour, Length: 30 * day}
	finale          = season.Finale{Length: 2 * time.Hour, Set: "rush", Description: "Triple points.", Times: 3}
	rush            = schedule.EntryID(uuid.MustParse("0192f5a0-7c00-7000-8000-000000000031"))
)

type harness struct {
	t       *testing.T
	clock   *cptime.FixedClock
	seasons *inmemory_season_store.Store
	entries *inmemory_entry_store.Store
	store   *inmemory_moment_store.Store
	outbox  *cpatomic.FakeOutbox
	useCase *tell_starts_usecase.UseCase
}

// Season 0 runs from its origin to 31 Oct 23:00; its day 24 ends at 21:00, and no day runs after it.
func newHarness(t *testing.T, at time.Time) *harness {
	t.Helper()

	h := &harness{
		t: t, clock: cptime.NewFixedClock(at), seasons: inmemory_season_store.New(), entries: inmemory_entry_store.New(),
		store: inmemory_moment_store.New(), outbox: &cpatomic.FakeOutbox{},
	}
	h.save(zero)
	h.boot()
	return h
}

func (h *harness) boot() {
	planner := moments.NewPlanner(h.seasons, h.entries, h.store, inmemory_siege_store.New(), season.Cutoff(21*time.Hour))
	h.useCase = tell_starts_usecase.New(planner, h.store, outbox_moment_events.New(h.outbox), cpatomic.FakeTransactioner{}, h.clock)
}

func (h *harness) save(snapshot season.Snapshot) {
	h.t.Helper()
	require.NoError(h.t, h.seasons.Save(h.t.Context(), snapshot.Restore()))
}

// The battle rush × 3 runs on day 20, from 18:00 to the cutoff.
func (h *harness) addBattle() {
	h.t.Helper()
	require.NoError(h.t, h.entries.Save(h.t.Context(), schedule.EntrySnapshot{
		ID: rush, Set: "rush", Kind: schedule.Battle, Description: "Triple points.", StartsAt: battleStarts,
		EndsAt: battleEnds, Priority: 1, Times: 3, MadeAt: battleStarts.Add(-day),
	}.Restore()))
}

func (h *harness) execute() []moments.Moment {
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

func assertTold[T proto.Message](h *harness, want ...T) {
	h.t.Helper()
	var told []T
	for _, event := range h.outbox.Recorded() {
		if event, ok := event.(T); ok {
			told = append(told, event)
		}
	}
	require.Len(h.t, told, len(want))
	for n := range want {
		assert.True(h.t, proto.Equal(want[n], told[n]), "%v", told[n])
	}
}

func TestABootTellsTheDayThatRunsOnce(t *testing.T) {
	h := newHarness(t, dayOneEnds.Add(20*day+12*time.Hour))

	assert.Len(t, h.execute(), 2, "the season and the day")
	assert.Empty(t, h.execute(), "told once")

	assertTold(h, started(22, dayOneEnds.Add(21*day)))
}

func TestTheNextDayIsToldAtTheCutoff(t *testing.T) {
	h := newHarness(t, dayOneEnds.Add(21*day-time.Second))
	h.execute()

	h.clock.Advance(time.Second)
	h.execute()

	assertTold(h, started(22, dayOneEnds.Add(21*day)), started(23, dayOneEnds.Add(22*day)))
}

func TestNoDayIsToldAfterTheSeasonsLastCutoff(t *testing.T) {
	h := newHarness(t, dayOneEnds.Add(23*day+time.Hour))

	h.execute()
	h.clock.Advance(2 * day)
	assert.Empty(t, h.execute(), "nor between two seasons")

	assertTold[*gamemasterv1.DayStarted](h)
}

func TestADayCutByAMovedSeasonEndIsToldAgainWithItsNewEnd(t *testing.T) {
	h := newHarness(t, dayOneEnds.Add(23*day-9*time.Hour))
	h.execute()
	require.NoError(t, h.store.KeepOpened(t.Context(), "season-0/day-24"))

	moved := zero
	moved.Length = 23*day + 20*time.Hour
	h.save(moved)
	h.execute()

	assertTold(h, started(24, dayOneEnds.Add(23*day)), started(24, dayOneEnds.Add(23*day-time.Hour)))
	assertTold(h, &gamemasterv1.SeasonStarted{
		Season: 0, BookId: "season-0", StartsAt: timestamppb.New(season.Origin), EndsAt: timestamppb.New(seasonZeroEnds),
	})
}

func TestASeasonIsToldOnceAsItStarts(t *testing.T) {
	h := newHarness(t, seasonOneStarts.Add(-time.Second))
	h.save(one)

	assert.Empty(t, h.execute(), "nothing runs in the break")
	h.clock.Advance(time.Second)
	assert.Len(t, h.execute(), 2, "the season and its first day")
	assert.Empty(t, h.execute(), "told once")
	h.boot()
	assert.Empty(t, h.execute(), "nor by a second boot")

	assertTold(h, &gamemasterv1.SeasonStarted{
		Season: 1, BookId: "season-1", StartsAt: timestamppb.New(seasonOneStarts),
		EndsAt: timestamppb.New(seasonOneStarts.Add(30 * day)),
	})
	assertTold(h, &gamemasterv1.DayStarted{
		Season: 1, Day: 1, RoundId: "season-1/day-1", BookId: "season-1", StartsAt: timestamppb.New(seasonOneStarts),
		EndsAt: timestamppb.New(seasonOneStarts.Add(6 * time.Hour)),
	})
}

func TestABattleIsToldOnceAsItStarts(t *testing.T) {
	h := newHarness(t, battleStarts.Add(-time.Second))
	h.addBattle()
	h.execute()

	h.clock.Advance(time.Second)
	assert.Len(t, h.execute(), 1)
	assert.Empty(t, h.execute(), "told once")
	h.boot()
	assert.Empty(t, h.execute(), "nor by a second boot")

	assertTold(h, &gamemasterv1.BattleStarted{
		Season: 0, Number: 1, EntryId: rush.String(), Times: 3, RoundId: rush.String(),
		StartsAt: timestamppb.New(battleStarts), EndsAt: timestamppb.New(battleEnds),
	})
}

func TestTheFinaleIsToldAsTheBattleThatEndsTheSeason(t *testing.T) {
	h := newHarness(t, finaleStarts)
	withFinale := zero
	withFinale.Finale = finale
	h.save(withFinale)
	h.addBattle()

	h.execute()
	h.clock.Advance(time.Hour)
	assert.Empty(t, h.execute(), "told once")

	assertTold(h, &gamemasterv1.BattleStarted{
		Season: 0, Number: 2, Times: 3, EndsSeason: true, RoundId: "season-0/finale",
		StartsAt: timestamppb.New(finaleStarts), EndsAt: timestamppb.New(seasonZeroEnds),
	})
	assertTold[*gamemasterv1.DayStarted](h)
}

func TestABootAfterABattleEndedTellsItNot(t *testing.T) {
	h := newHarness(t, battleEnds)
	h.addBattle()

	h.execute()

	assertTold[*gamemasterv1.BattleStarted](h)
	assertTold(h, started(21, dayOneEnds.Add(20*day)))
}

func TestABootBetweenTwoSeasonsTellsNothingThatEnded(t *testing.T) {
	h := newHarness(t, seasonOneStarts.Add(-time.Hour))
	withFinale := zero
	withFinale.Finale = finale
	h.save(withFinale)
	h.save(one)
	h.addBattle()

	assert.Empty(t, h.execute())
	assert.Empty(t, h.outbox.Recorded(), "season 0, its days, its battle and its finale ended")
}

func TestWhatCannotBeKeptIsAnErrorAndNothingIsTold(t *testing.T) {
	h := newHarness(t, dayOneEnds.Add(20*day+12*time.Hour))
	down := errors.New("postgres is down")
	h.store.FailWith(down)

	_, err := h.useCase.Execute(t.Context())

	require.ErrorIs(t, err, down)
	assert.Empty(t, h.outbox.Recorded())
}
