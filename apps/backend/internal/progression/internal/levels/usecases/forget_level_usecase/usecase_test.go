package forget_level_usecase_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/levels"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/levels/inmemory_level_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/levels/usecases/forget_level_usecase"
)

func TestAForgottenAccountLosesItsLevelTwiceOver(t *testing.T) {
	store := inmemory_level_store.New()
	ada, bob := accounts.AccountID{15: 1}, accounts.AccountID{15: 2}
	require.NoError(t, store.Set(t.Context(), levels.ByAccount{ada: 4, bob: 2}, time.Now()))

	for range 2 {
		require.NoError(t, forget_level_usecase.New(store).Execute(t.Context(), ada))
	}

	kept, err := store.Levels(t.Context(), []accounts.AccountID{ada, bob})
	require.NoError(t, err)
	assert.Equal(t, levels.ByAccount{bob: 2}, kept)
}
