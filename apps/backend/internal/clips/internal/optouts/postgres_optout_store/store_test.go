package postgres_optout_store_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/optouts"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/optouts/postgres_optout_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func TestAPlayerKeptOutIsFoundAmongOthersUntilForgotten(t *testing.T) {
	store := postgres_optout_store.New(cppg.StartTestServer(t).OpenSchema(t, "clips", migrations.FS))
	ada, bob := optouts.AccountID(uuid.New()), optouts.AccountID(uuid.New())

	require.NoError(t, store.Keep(t.Context(), ada))
	require.NoError(t, store.Keep(t.Context(), ada), "kept twice")
	out, err := store.Among(t.Context(), []optouts.AccountID{ada, bob})
	require.NoError(t, err)
	assert.Equal(t, []optouts.AccountID{ada}, out)

	require.NoError(t, store.Forget(t.Context(), ada))
	out, err = store.Among(t.Context(), []optouts.AccountID{ada, bob})
	require.NoError(t, err)
	assert.Empty(t, out)
}
