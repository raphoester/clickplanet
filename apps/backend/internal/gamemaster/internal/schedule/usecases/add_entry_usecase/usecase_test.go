package add_entry_usecase_test

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/schedule"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/schedule/inmemory_entry_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/schedule/usecases/add_entry_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var (
	now    = time.Date(2026, 10, 31, 20, 0, 0, 0, time.UTC)
	finale = now.Add(time.Hour)
	first  = schedule.EntryID(uuid.UUID{15: 1})
)

func keepers(planet, antibot []schedule.SetName) []schedule.Keeper {
	return []schedule.Keeper{
		schedule.FakeKeeper{Module: "planet", Sets: planet},
		schedule.FakeKeeper{Module: "the antibot", Sets: antibot},
	}
}

func adding(store *inmemory_entry_store.Store, keepers []schedule.Keeper) *add_entry_usecase.UseCase {
	return add_entry_usecase.New(store, keepers, &schedule.FakeIDs{IDs: []schedule.EntryID{first}}, cptime.NewFixedClock(now))
}

var rush = schedule.Draft{
	Set: "rush", Kind: schedule.Battle, Description: "Triple points.", StartsAt: finale, EndsAt: finale.Add(2 * time.Hour), Priority: 1,
}

func TestAnEntryToComeOfASetEveryModuleKeepsIsKept(t *testing.T) {
	store := inmemory_entry_store.New()

	id, err := adding(store, keepers([]schedule.SetName{"default", "rush"}, []schedule.SetName{"rush"})).
		Execute(t.Context(), rush)

	require.NoError(t, err)
	assert.Equal(t, first, id)
	entry, err := store.Entry(t.Context(), first)
	require.NoError(t, err)
	assert.Equal(t, schedule.EntrySnapshot{
		ID: first, Set: "rush", Kind: schedule.Battle, Description: "Triple points.", StartsAt: finale, EndsAt: finale.Add(2 * time.Hour),
		Priority: 1, MadeAt: now,
	}, entry.TakeSnapshot())
}

func TestASetAModuleLacksIsRefusedAndNamesTheModule(t *testing.T) {
	store := inmemory_entry_store.New()

	_, err := adding(store, keepers([]schedule.SetName{"default"}, nil)).Execute(t.Context(), rush)

	require.ErrorIs(t, err, schedule.ErrUnknownSet)
	assert.Contains(t, err.Error(), `planet keeps no rule set of this name: "rush"`)
	assert.Contains(t, err.Error(), `the antibot keeps no rule set of this name: "rush"`)
	unended, err := store.Unended(t.Context(), now)
	require.NoError(t, err)
	assert.Empty(t, unended)
}

func TestAnEntryThatCannotBeScheduledIsRefusedBeforeAnyModuleIsAsked(t *testing.T) {
	asked := errors.New("asked")
	refused := rush
	refused.StartsAt = now

	_, err := adding(inmemory_entry_store.New(), []schedule.Keeper{schedule.FakeKeeper{Err: asked}}).
		Execute(t.Context(), refused)

	require.ErrorIs(t, err, schedule.ErrInvalidEntry)
	assert.NotErrorIs(t, err, asked)
}

func TestAFailedSaveIsAnError(t *testing.T) {
	store := inmemory_entry_store.New()
	down := errors.New("postgres is down")
	store.FailWith(down)

	_, err := adding(store, keepers([]schedule.SetName{"rush"}, []schedule.SetName{"rush"})).Execute(t.Context(), rush)

	require.ErrorIs(t, err, down)
}
