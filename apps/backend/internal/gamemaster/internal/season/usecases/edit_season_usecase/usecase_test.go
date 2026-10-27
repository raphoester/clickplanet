package edit_season_usecase_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/season"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/season/inmemory_season_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/season/usecases/edit_season_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var (
	zero = season.Snapshot{Number: 0, StartsAt: time.Date(2026, 10, 3, 0, 0, 0, 0, time.UTC), EndsAt: time.Date(2026, 10, 31, 23, 0, 0, 0, time.UTC)}
	now  = time.Date(2026, 10, 28, 12, 0, 0, 0, time.UTC)
)

func TestTheEndOfTheRunningSeasonMoves(t *testing.T) {
	seasons := inmemory_season_store.New()
	require.NoError(t, seasons.Save(t.Context(), zero.Restore()))
	moved := season.Draft{StartsAt: zero.StartsAt, EndsAt: zero.EndsAt.AddDate(0, 0, 7)}

	require.NoError(t, edit_season_usecase.New(seasons, cptime.NewFixedClock(now)).Execute(t.Context(), 0, moved))

	calendar, err := seasons.Calendar(t.Context())
	require.NoError(t, err)
	assert.Equal(t, []season.Snapshot{{Number: 0, StartsAt: moved.StartsAt, EndsAt: moved.EndsAt}}, calendar.TakeSnapshot())
}

func TestARefusedEditKeepsTheSeason(t *testing.T) {
	seasons := inmemory_season_store.New()
	require.NoError(t, seasons.Save(t.Context(), zero.Restore()))

	err := edit_season_usecase.New(seasons, cptime.NewFixedClock(zero.EndsAt)).
		Execute(t.Context(), 0, season.Draft{StartsAt: zero.StartsAt, EndsAt: zero.EndsAt.AddDate(0, 0, 7)})

	require.ErrorIs(t, err, season.ErrEnded)
	calendar, err := seasons.Calendar(t.Context())
	require.NoError(t, err)
	assert.Equal(t, []season.Snapshot{zero}, calendar.TakeSnapshot())
}
