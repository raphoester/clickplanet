package edit_season_usecase_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/moments"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/schedule"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/schedule/inmemory_entry_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/season"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/season/inmemory_season_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/season/usecases/edit_season_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var (
	zero = season.Snapshot{Number: 0, Length: time.Date(2026, 10, 31, 23, 0, 0, 0, time.UTC).Sub(season.Origin)}
	now  = time.Date(2026, 10, 28, 12, 0, 0, 0, time.UTC)
)

var rush = season.Finale{Length: 2 * time.Hour, Set: "rush", Description: "Triple points.", Times: 3}

func editing(seasons *inmemory_season_store.Store, entries *inmemory_entry_store.Store, at time.Time) *edit_season_usecase.UseCase {
	keepers := []schedule.Keeper{
		schedule.FakeKeeper{Module: "planet", Sets: []schedule.SetName{"default", "rush"}},
		schedule.FakeKeeper{Module: "the antibot", Sets: []schedule.SetName{"default"}},
	}
	return edit_season_usecase.New(seasons, entries, keepers, season.Cutoff(21*time.Hour), cptime.NewFixedClock(at))
}

func TestTheEndOfTheRunningSeasonMoves(t *testing.T) {
	seasons := inmemory_season_store.New()
	require.NoError(t, seasons.Save(t.Context(), zero.Restore()))
	moved := season.Draft{Length: zero.Length + 7*24*time.Hour}

	require.NoError(t, editing(seasons, inmemory_entry_store.New(), now).Execute(t.Context(), 0, moved))

	calendar, err := seasons.Calendar(t.Context())
	require.NoError(t, err)
	assert.Equal(t, []season.Snapshot{{Number: 0, Length: moved.Length}}, calendar.TakeSnapshot())
}

func TestARefusedEditKeepsTheSeason(t *testing.T) {
	seasons := inmemory_season_store.New()
	require.NoError(t, seasons.Save(t.Context(), zero.Restore()))

	err := editing(seasons, inmemory_entry_store.New(), season.Origin.Add(zero.Length)).
		Execute(t.Context(), 0, season.Draft{Length: zero.Length + 7*24*time.Hour})

	require.ErrorIs(t, err, season.ErrEnded)
	calendar, err := seasons.Calendar(t.Context())
	require.NoError(t, err)
	assert.Equal(t, []season.Snapshot{zero}, calendar.TakeSnapshot())
}

func TestAFinaleOfASetAModuleLacksIsRefused(t *testing.T) {
	seasons := inmemory_season_store.New()
	require.NoError(t, seasons.Save(t.Context(), zero.Restore()))

	err := editing(seasons, inmemory_entry_store.New(), now).Execute(t.Context(), 0, season.Draft{Length: zero.Length, Finale: rush})

	require.ErrorIs(t, err, schedule.ErrUnknownSet)
	require.ErrorContains(t, err, "the antibot")
}

func TestAMovedEndThatPutsTheFinaleOverABattleIsRefused(t *testing.T) {
	seasons := inmemory_season_store.New()
	require.NoError(t, seasons.Save(t.Context(), season.Snapshot{Number: 0, Length: zero.Length, Finale: rush}.Restore()))
	entries := inmemory_entry_store.New()
	later := time.Date(2026, 11, 1, 21, 0, 0, 0, time.UTC)
	require.NoError(t, entries.Save(t.Context(), schedule.EntrySnapshot{
		ID: schedule.EntryID(uuid.UUID{15: 1}), Set: "rush", Kind: schedule.Battle, StartsAt: later, EndsAt: later.Add(2 * time.Hour),
		Times: 3,
	}.Restore()))
	keepers := []schedule.Keeper{schedule.FakeKeeper{Module: "planet", Sets: []schedule.SetName{"rush"}}}

	err := edit_season_usecase.New(seasons, entries, keepers, season.Cutoff(21*time.Hour), cptime.NewFixedClock(now)).
		Execute(t.Context(), 0, season.Draft{Length: zero.Length + 24*time.Hour, Finale: rush})

	require.ErrorIs(t, err, moments.ErrOverlap)
	calendar, err := seasons.Calendar(t.Context())
	require.NoError(t, err)
	assert.Equal(t, zero.Length, calendar.TakeSnapshot()[0].Length)
}
