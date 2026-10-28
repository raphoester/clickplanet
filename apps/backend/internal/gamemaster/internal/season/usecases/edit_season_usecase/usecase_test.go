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
	zero = season.Snapshot{Number: 0, Length: time.Date(2026, 10, 31, 23, 0, 0, 0, time.UTC).Sub(season.Origin)}
	now  = time.Date(2026, 10, 28, 12, 0, 0, 0, time.UTC)
)

func TestTheEndOfTheRunningSeasonMoves(t *testing.T) {
	seasons := inmemory_season_store.New()
	require.NoError(t, seasons.Save(t.Context(), zero.Restore()))
	moved := season.Draft{Length: zero.Length + 7*24*time.Hour}

	require.NoError(t, edit_season_usecase.New(seasons, cptime.NewFixedClock(now)).Execute(t.Context(), 0, moved))

	calendar, err := seasons.Calendar(t.Context())
	require.NoError(t, err)
	assert.Equal(t, []season.Snapshot{{Number: 0, Length: moved.Length}}, calendar.TakeSnapshot())
}

func TestARefusedEditKeepsTheSeason(t *testing.T) {
	seasons := inmemory_season_store.New()
	require.NoError(t, seasons.Save(t.Context(), zero.Restore()))

	err := edit_season_usecase.New(seasons, cptime.NewFixedClock(season.Origin.Add(zero.Length))).
		Execute(t.Context(), 0, season.Draft{Length: zero.Length + 7*24*time.Hour})

	require.ErrorIs(t, err, season.ErrEnded)
	calendar, err := seasons.Calendar(t.Context())
	require.NoError(t, err)
	assert.Equal(t, []season.Snapshot{zero}, calendar.TakeSnapshot())
}
