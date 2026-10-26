package delete_entry_usecase_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/rules/internal/schedule"
	"github.com/raphoester/clickplanet.lol-backend/internal/rules/internal/schedule/inmemory_entry_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/rules/internal/schedule/usecases/delete_entry_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var (
	madeAt = time.Date(2026, 10, 31, 19, 0, 0, 0, time.UTC)
	finale = madeAt.Add(2 * time.Hour)
	first  = schedule.EntryID(uuid.UUID{15: 1})
)

func kept(t *testing.T) *inmemory_entry_store.Store {
	t.Helper()

	store := inmemory_entry_store.New()
	entry, err := schedule.NewEntry(first, schedule.Draft{Set: "rush", StartsAt: finale, EndsAt: finale.Add(2 * time.Hour), Priority: 1}, madeAt)
	require.NoError(t, err)
	require.NoError(t, store.Save(t.Context(), entry))
	return store
}

func TestAnEntryToComeIsDeleted(t *testing.T) {
	store := kept(t)

	require.NoError(t, delete_entry_usecase.New(store, cptime.NewFixedClock(finale.Add(-time.Second))).Execute(t.Context(), first))

	_, err := store.Entry(t.Context(), first)
	require.ErrorIs(t, err, schedule.ErrUnknownEntry)
}

func TestAnEntryThatStartedIsNotDeleted(t *testing.T) {
	store := kept(t)

	err := delete_entry_usecase.New(store, cptime.NewFixedClock(finale)).Execute(t.Context(), first)

	require.ErrorIs(t, err, schedule.ErrStarted)
	_, err = store.Entry(t.Context(), first)
	require.NoError(t, err)
}

func TestAnEntryNeverKeptIsUnknown(t *testing.T) {
	err := delete_entry_usecase.New(kept(t), cptime.NewFixedClock(madeAt)).Execute(t.Context(), schedule.EntryID(uuid.UUID{15: 9}))

	require.ErrorIs(t, err, schedule.ErrUnknownEntry)
}
