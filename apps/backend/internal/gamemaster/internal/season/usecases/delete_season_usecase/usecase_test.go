package delete_season_usecase_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/season"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/season/inmemory_season_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/season/usecases/delete_season_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var (
	zero = season.Snapshot{Number: 0, StartsAt: time.Date(2026, 10, 3, 0, 0, 0, 0, time.UTC), EndsAt: time.Date(2026, 10, 31, 23, 0, 0, 0, time.UTC)}
	one  = season.Snapshot{Number: 1, StartsAt: zero.EndsAt.Add(time.Hour), EndsAt: zero.EndsAt.AddDate(0, 1, 0)}
	now  = time.Date(2026, 10, 28, 12, 0, 0, 0, time.UTC)
)

func seasons(t *testing.T) *inmemory_season_store.Store {
	t.Helper()
	store := inmemory_season_store.New()
	require.NoError(t, store.Save(t.Context(), zero.Restore()))
	require.NoError(t, store.Save(t.Context(), one.Restore()))
	return store
}

func TestTheLastSeasonToComeIsDeleted(t *testing.T) {
	store := seasons(t)

	require.NoError(t, delete_season_usecase.New(store, cptime.NewFixedClock(now)).Execute(t.Context(), 1))

	calendar, err := store.Calendar(t.Context())
	require.NoError(t, err)
	assert.Equal(t, []season.Snapshot{zero}, calendar.TakeSnapshot())
}

func TestAStartedSeasonIsKept(t *testing.T) {
	store := seasons(t)

	err := delete_season_usecase.New(store, cptime.NewFixedClock(now)).Execute(t.Context(), 0)

	require.ErrorIs(t, err, season.ErrStarted)
	calendar, err := store.Calendar(t.Context())
	require.NoError(t, err)
	assert.Len(t, calendar.TakeSnapshot(), 2)
}
