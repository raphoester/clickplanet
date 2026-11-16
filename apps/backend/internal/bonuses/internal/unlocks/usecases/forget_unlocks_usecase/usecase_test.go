package forget_unlocks_usecase_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/unlocks"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/unlocks/inmemory_unlock_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/unlocks/usecases/forget_unlocks_usecase"
)

const ada = bonuses.Account("0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11")

func TestAForgottenAccountHasNothingUnlocked(t *testing.T) {
	store := inmemory_unlock_store.New()
	require.NoError(t, store.Change(t.Context(), ada, func(u unlocks.Unlocks) unlocks.Unlocks {
		return u.Decided(unlocks.DecisionOf(bonuses.Unlocked{bonuses.KindBomb}, time.Now()))
	}))

	require.NoError(t, forget_unlocks_usecase.New(store).Execute(t.Context(), ada))

	kept, err := store.Unlocked(t.Context(), ada)
	require.NoError(t, err)
	assert.Empty(t, kept)
}
