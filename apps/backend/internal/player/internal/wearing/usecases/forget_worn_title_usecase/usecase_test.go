package forget_worn_title_usecase_test

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/wearing/inmemory_worn_title_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/wearing/usecases/forget_worn_title_usecase"
)

func TestADeletedAccountWearsNothing(t *testing.T) {
	store := inmemory_worn_title_store.New()
	require.NoError(t, store.Wear(t.Context(), players.AccountID{15: 1}, "og", time.Now()))

	require.NoError(t, forget_worn_title_usecase.New(store).Execute(t.Context(), players.AccountID{15: 1}))

	choice, err := store.Choice(t.Context(), players.AccountID{15: 1})
	require.NoError(t, err)
	assert.Empty(t, choice)
}

func TestAStoreFailureIsAnError(t *testing.T) {
	store := inmemory_worn_title_store.New()
	store.FailWith(errors.New("postgres is down"))

	assert.Error(t, forget_worn_title_usecase.New(store).Execute(t.Context(), players.AccountID{15: 1}))
}
