package forget_titles_usecase_test

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/titles"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/titles/inmemory_title_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/titles/usecases/forget_titles_usecase"
)

func TestADeletedAccountLosesItsTitles(t *testing.T) {
	store := inmemory_title_store.New()
	require.NoError(t, store.Grant(t.Context(), titles.Holdings{{15: 1}: {"og"}}, time.Now()))

	require.NoError(t, forget_titles_usecase.New(store).Execute(t.Context(), players.AccountID{15: 1}))

	held, err := store.Held(t.Context(), players.AccountID{15: 1})
	require.NoError(t, err)
	assert.Empty(t, held)
}

func TestAStoreFailureIsAnError(t *testing.T) {
	store := inmemory_title_store.New()
	store.FailWith(errors.New("postgres is down"))

	assert.Error(t, forget_titles_usecase.New(store).Execute(t.Context(), players.AccountID{15: 1}))
}
