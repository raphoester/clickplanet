package edit_entry_usecase_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/rules/internal/schedule"
	"github.com/raphoester/clickplanet.lol-backend/internal/rules/internal/schedule/inmemory_entry_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/rules/internal/schedule/usecases/edit_entry_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var (
	madeAt = time.Date(2026, 10, 31, 19, 0, 0, 0, time.UTC)
	finale = madeAt.Add(2 * time.Hour)
	first  = schedule.EntryID(uuid.UUID{15: 1})
	all    = []schedule.Keeper{schedule.FakeKeeper{Module: "planet", Sets: []schedule.SetName{"rush", "calm"}}}
)

func kept(t *testing.T) *inmemory_entry_store.Store {
	t.Helper()

	store := inmemory_entry_store.New()
	entry, err := schedule.NewEntry(first, schedule.Draft{Set: "rush", StartsAt: finale, EndsAt: finale.Add(2 * time.Hour), Priority: 1}, madeAt)
	require.NoError(t, err)
	require.NoError(t, store.Save(t.Context(), entry))
	return store
}

var calm = schedule.Draft{Set: "calm", StartsAt: finale.Add(time.Hour), EndsAt: finale.Add(2 * time.Hour), Priority: 2}

func TestAnEntryToComeTakesTheNewValues(t *testing.T) {
	store := kept(t)

	require.NoError(t, edit_entry_usecase.New(store, all, cptime.NewFixedClock(finale.Add(-time.Minute))).
		Execute(t.Context(), first, calm))

	entry, err := store.Entry(t.Context(), first)
	require.NoError(t, err)
	assert.Equal(t, schedule.EntrySnapshot{
		ID: first, Set: "calm", StartsAt: finale.Add(time.Hour), EndsAt: finale.Add(2 * time.Hour), Priority: 2, MadeAt: madeAt,
	}, entry.TakeSnapshot())
}

func TestAnEntryThatStartedRunsAsItIs(t *testing.T) {
	store := kept(t)

	err := edit_entry_usecase.New(store, all, cptime.NewFixedClock(finale)).Execute(t.Context(), first, calm)

	require.ErrorIs(t, err, schedule.ErrStarted)
	entry, err := store.Entry(t.Context(), first)
	require.NoError(t, err)
	assert.Equal(t, schedule.SetName("rush"), entry.TakeSnapshot().Set)
}

func TestASetAModuleLacksIsRefused(t *testing.T) {
	frozen := calm
	frozen.Set = "frozen"

	err := edit_entry_usecase.New(kept(t), all, cptime.NewFixedClock(madeAt)).Execute(t.Context(), first, frozen)

	require.ErrorIs(t, err, schedule.ErrUnknownSet)
}

func TestAnEntryNeverKeptIsUnknown(t *testing.T) {
	err := edit_entry_usecase.New(kept(t), all, cptime.NewFixedClock(madeAt)).
		Execute(t.Context(), schedule.EntryID(uuid.UUID{15: 9}), calm)

	require.ErrorIs(t, err, schedule.ErrUnknownEntry)
}
