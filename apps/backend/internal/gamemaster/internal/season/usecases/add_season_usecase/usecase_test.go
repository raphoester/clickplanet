package add_season_usecase_test

import (
	"errors"
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
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/season/usecases/add_season_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var (
	zero = season.Snapshot{Number: 0, Length: time.Date(2026, 10, 31, 23, 0, 0, 0, time.UTC).Sub(season.Origin)}
	now  = time.Date(2026, 10, 28, 12, 0, 0, 0, time.UTC)
	one  = season.Draft{Break: time.Hour, Length: 28 * 24 * time.Hour}
)

func adding(seasons *inmemory_season_store.Store, entries *inmemory_entry_store.Store) *add_season_usecase.UseCase {
	keepers := []schedule.Keeper{
		schedule.FakeKeeper{Module: "planet", Sets: []schedule.SetName{"default", "rush"}},
		schedule.FakeKeeper{Module: "the antibot", Sets: []schedule.SetName{"default", "rush"}},
	}
	return add_season_usecase.New(seasons, entries, keepers, season.Cutoff(21*time.Hour), cptime.NewFixedClock(now))
}

func store(t *testing.T) *inmemory_season_store.Store {
	t.Helper()
	seasons := inmemory_season_store.New()
	require.NoError(t, seasons.Save(t.Context(), zero.Restore()))
	return seasons
}

func TestASeasonIsAddedUnderTheNextNumber(t *testing.T) {
	seasons := store(t)

	number, err := adding(seasons, inmemory_entry_store.New()).Execute(t.Context(), one)

	require.NoError(t, err)
	assert.Equal(t, season.Number(1), number)
	calendar, err := seasons.Calendar(t.Context())
	require.NoError(t, err)
	assert.Equal(t, []season.Snapshot{zero, {Number: 1, Break: one.Break, Length: one.Length}}, calendar.TakeSnapshot())
}

func TestARefusedSeasonIsNotKept(t *testing.T) {
	seasons := store(t)

	_, err := adding(seasons, inmemory_entry_store.New()).Execute(t.Context(), season.Draft{Break: time.Hour})

	require.ErrorIs(t, err, season.ErrInvalidSeason)
	calendar, err := seasons.Calendar(t.Context())
	require.NoError(t, err)
	assert.Len(t, calendar.TakeSnapshot(), 1)
}

func TestSeasonsThatCannotBeReadAddNone(t *testing.T) {
	seasons := store(t)
	down := errors.New("postgres is down")
	seasons.FailWith(down)

	_, err := adding(seasons, inmemory_entry_store.New()).Execute(t.Context(), one)

	require.ErrorIs(t, err, down)
}

func TestASeasonIsAddedWithItsFinale(t *testing.T) {
	seasons := store(t)
	draft := season.Draft{Break: one.Break, Length: one.Length, Finale: season.Finale{Length: 3 * time.Hour, Set: "rush", Times: 3}}

	_, err := adding(seasons, inmemory_entry_store.New()).Execute(t.Context(), draft)

	require.NoError(t, err)
	calendar, err := seasons.Calendar(t.Context())
	require.NoError(t, err)
	assert.Equal(t, draft.Finale, calendar.TakeSnapshot()[1].Finale)
}

func TestAFinaleOfASetAModuleLacksIsRefused(t *testing.T) {
	seasons := store(t)
	draft := season.Draft{Break: one.Break, Length: one.Length, Finale: season.Finale{Length: 2 * time.Hour, Set: "storm", Times: 3}}

	_, err := adding(seasons, inmemory_entry_store.New()).Execute(t.Context(), draft)

	require.ErrorIs(t, err, schedule.ErrUnknownSet)
	require.ErrorContains(t, err, "planet")
	calendar, err := seasons.Calendar(t.Context())
	require.NoError(t, err)
	assert.Len(t, calendar.TakeSnapshot(), 1)
}

func TestAFinaleNoBattleOverlapsIsAdded(t *testing.T) {
	seasons := store(t)
	ends := time.Date(2026, 10, 31, 23, 0, 0, 0, time.UTC).Add(one.Break + one.Length)
	entries := inmemory_entry_store.New()
	require.NoError(t, entries.Save(t.Context(), schedule.EntrySnapshot{
		ID: schedule.EntryID(uuid.UUID{15: 1}), Set: "rush", Kind: schedule.Battle, StartsAt: ends.Add(-3 * time.Hour),
		EndsAt: ends.Add(-2 * time.Hour), Times: 3,
	}.Restore()))
	draft := season.Draft{Break: one.Break, Length: one.Length, Finale: season.Finale{Length: 3 * time.Hour, Set: "rush", Times: 3}}

	_, err := adding(seasons, entries).Execute(t.Context(), draft)

	require.ErrorIs(t, err, moments.ErrOverlap)
}

func TestAFinaleAtNoCutoffIsRefused(t *testing.T) {
	seasons := store(t)
	draft := season.Draft{Break: one.Break, Length: one.Length, Finale: season.Finale{Length: 2 * time.Hour, Set: "rush", Times: 3}}

	_, err := adding(seasons, inmemory_entry_store.New()).Execute(t.Context(), draft)

	require.ErrorIs(t, err, moments.ErrOffCutoff)
	calendar, err := seasons.Calendar(t.Context())
	require.NoError(t, err)
	assert.Len(t, calendar.TakeSnapshot(), 1)
}
