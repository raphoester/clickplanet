package record_take_usecase_test

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/calendar"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/standings"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/standings/inmemory_contribution_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/standings/usecases/record_take_usecase"
)

var (
	seasonZeroEnds = time.Date(2026, 10, 31, 23, 0, 0, 0, time.UTC)
	ada            = standings.AccountID{0: 1, 15: 1}
)

func seasonZero() calendar.Calendar {
	return calendar.New(calendar.Config{List: []calendar.Entry{
		{Number: 0, EndsAt: seasonZeroEnds, Finale: 2 * time.Hour},
	}})
}

func TestATakeCountsOneTileForItsFlagInItsSeason(t *testing.T) {
	store := inmemory_contribution_store.New()
	useCase := record_take_usecase.New(seasonZero(), store)

	require.NoError(t, useCase.Execute(t.Context(), standings.Take{Account: ada, Country: "fr", At: seasonZeroEnds.Add(-time.Hour)}))
	require.NoError(t, useCase.Execute(t.Context(), standings.Take{Account: ada, Country: "fr", At: seasonZeroEnds.Add(-time.Minute)}))

	assert.Equal(t, standings.Tally{Main: "fr", Tiles: map[standings.Country]uint64{"fr": 2}}, store.Tally(0, ada))
}

func TestATakeAfterTheLastSeasonCountsNothing(t *testing.T) {
	store := inmemory_contribution_store.New()

	require.NoError(t, record_take_usecase.New(seasonZero(), store).Execute(t.Context(),
		standings.Take{Account: ada, Country: "fr", At: seasonZeroEnds}))

	assert.True(t, store.Tally(0, ada).Empty())
}

func TestAFailureToCountIsAnError(t *testing.T) {
	store := inmemory_contribution_store.New()
	refused := errors.New("the database is down")
	store.FailWith(refused)

	err := record_take_usecase.New(seasonZero(), store).Execute(t.Context(),
		standings.Take{Account: ada, Country: "fr", At: seasonZeroEnds.Add(-time.Hour)})

	assert.ErrorIs(t, err, refused)
}
