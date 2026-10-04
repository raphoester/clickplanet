package rebuild_standings_usecase_test

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/standings"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/standings/inmemory_contribution_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/standings/usecases/rebuild_standings_usecase"
)

func TestARebuildForgetsTheStandingsAndStartsAtTheFirstTake(t *testing.T) {
	ada := standings.AccountID{15: 1}
	store := inmemory_contribution_store.New()
	require.NoError(t, store.Begin(t.Context(), 7))
	require.NoError(t, store.RecordTake(t.Context(), 0, standings.Take{Account: ada, Country: "fr", At: time.Now()}))

	out, err := rebuild_standings_usecase.New(store).Execute(t.Context())

	require.NoError(t, err)
	assert.Equal(t, rebuild_standings_usecase.Out{From: 0}, out)
	position, err := store.Position(t.Context())
	require.NoError(t, err)
	assert.Equal(t, standings.Position(0), position)
	assert.True(t, store.Tally(0, ada).Empty())
}

func TestStandingsThatNeverBeganCannotBeRebuilt(t *testing.T) {
	_, err := rebuild_standings_usecase.New(inmemory_contribution_store.New()).Execute(t.Context())

	assert.ErrorIs(t, err, standings.ErrNotStarted)
}

func TestAFailedRebuildIsAnError(t *testing.T) {
	store := inmemory_contribution_store.New()
	failed := errors.New("postgres is down")
	store.FailWith(failed)

	_, err := rebuild_standings_usecase.New(store).Execute(t.Context())

	assert.ErrorIs(t, err, failed)
}
