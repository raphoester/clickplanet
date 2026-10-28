package set_name_shown_usecase_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/optouts"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/optouts/usecases/set_name_shown_usecase"
)

func TestAPlayerTurnsTheirNameOffAndOnAgain(t *testing.T) {
	ada, bob := optouts.AccountID(uuid.New()), optouts.AccountID(uuid.New())
	store := optouts.NewMemoryStore()
	useCase := set_name_shown_usecase.New(store)
	among := func() []optouts.AccountID {
		out, err := store.Among(t.Context(), []optouts.AccountID{ada, bob})
		require.NoError(t, err)
		return out
	}

	require.NoError(t, useCase.Execute(t.Context(), ada, false))
	require.NoError(t, useCase.Execute(t.Context(), ada, false))
	assert.Equal(t, []optouts.AccountID{ada}, among(), "turned off twice, kept out once")

	require.NoError(t, useCase.Execute(t.Context(), ada, true))
	assert.Empty(t, among())
}
