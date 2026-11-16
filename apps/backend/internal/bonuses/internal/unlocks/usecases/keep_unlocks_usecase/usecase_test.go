package keep_unlocks_usecase_test

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/unlocks"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/unlocks/inmemory_unlock_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/unlocks/usecases/keep_unlocks_usecase"
)

const ada = bonuses.Account("0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11")

var monday = time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC)

func TestTheLatestDecisionIsKept(t *testing.T) {
	store := inmemory_unlock_store.New()
	useCase := keep_unlocks_usecase.New(store)

	require.NoError(t, useCase.Execute(t.Context(), ada, unlocks.DecisionOf(bonuses.Unlocked{bonuses.KindEncloseClicks}, monday.Add(time.Hour))))
	require.NoError(t, useCase.Execute(t.Context(), ada, unlocks.DecisionOf(bonuses.Unlocked{bonuses.KindBomb}, monday)))

	kept, err := store.Unlocked(t.Context(), ada)
	require.NoError(t, err)
	assert.Equal(t, bonuses.Unlocked{bonuses.KindEncloseClicks}, kept, "the late decision was older")
}

func TestADecisionWithNoTimeIsRefused(t *testing.T) {
	err := keep_unlocks_usecase.New(inmemory_unlock_store.New()).Execute(t.Context(), ada, unlocks.DecisionOf(nil, time.Time{}))

	require.ErrorIs(t, err, unlocks.ErrNoTime)
}

func TestAStoreThatFailsIsTriedAgain(t *testing.T) {
	store := inmemory_unlock_store.New()
	store.FailWith(errors.New("postgres is gone"))

	err := keep_unlocks_usecase.New(store).Execute(t.Context(), ada, unlocks.DecisionOf(nil, monday))

	require.Error(t, err)
}
