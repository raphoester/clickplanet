package add_season_usecase_test

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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

func store(t *testing.T) *inmemory_season_store.Store {
	t.Helper()
	seasons := inmemory_season_store.New()
	require.NoError(t, seasons.Save(t.Context(), zero.Restore()))
	return seasons
}

func TestASeasonIsAddedUnderTheNextNumber(t *testing.T) {
	seasons := store(t)

	number, err := add_season_usecase.New(seasons, cptime.NewFixedClock(now)).Execute(t.Context(), one)

	require.NoError(t, err)
	assert.Equal(t, season.Number(1), number)
	calendar, err := seasons.Calendar(t.Context())
	require.NoError(t, err)
	assert.Equal(t, []season.Snapshot{zero, {Number: 1, Break: one.Break, Length: one.Length}}, calendar.TakeSnapshot())
}

func TestARefusedSeasonIsNotKept(t *testing.T) {
	seasons := store(t)

	_, err := add_season_usecase.New(seasons, cptime.NewFixedClock(now)).Execute(t.Context(), season.Draft{Break: time.Hour})

	require.ErrorIs(t, err, season.ErrInvalidSeason)
	calendar, err := seasons.Calendar(t.Context())
	require.NoError(t, err)
	assert.Len(t, calendar.TakeSnapshot(), 1)
}

func TestSeasonsThatCannotBeReadAddNone(t *testing.T) {
	seasons := store(t)
	down := errors.New("postgres is down")
	seasons.FailWith(down)

	_, err := add_season_usecase.New(seasons, cptime.NewFixedClock(now)).Execute(t.Context(), one)

	require.ErrorIs(t, err, down)
}
