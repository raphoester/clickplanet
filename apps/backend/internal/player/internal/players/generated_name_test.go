package players_test

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/inmemory_player_store"
)

var namedAt = time.Date(2026, 10, 4, 12, 0, 0, 0, time.UTC)

func TestAssignGivesANamelessAccountAGeneratedName(t *testing.T) {
	store := inmemory_player_store.New()
	names := players.NewGeneratedNames(store, players.NewRepeatedNames("BraveFox42", "SlyOtter17"))

	require.NoError(t, names.Assign(t.Context(), players.AccountID{15: 1}, namedAt))
	require.NoError(t, names.Assign(t.Context(), players.AccountID{15: 1}, namedAt.Add(time.Hour)))

	profile, err := store.Profile(t.Context(), players.AccountID{15: 1})
	require.NoError(t, err)
	assert.Equal(t, players.Name("BraveFox42"), profile.TakeSnapshot().Name, "a second Assign draws nothing")
	assert.True(t, namedAt.Equal(profile.TakeSnapshot().UpdatedAt))
}

func TestAssignLeavesAChosenNameAlone(t *testing.T) {
	store := inmemory_player_store.New()
	require.NoError(t, store.SaveProfile(t.Context(), players.NewProfile(players.AccountID{15: 1}, "Ada", namedAt)))

	require.NoError(t, players.NewGeneratedNames(store, players.NewRepeatedNames("BraveFox42")).
		Assign(t.Context(), players.AccountID{15: 1}, namedAt.Add(time.Hour)))

	profile, err := store.Profile(t.Context(), players.AccountID{15: 1})
	require.NoError(t, err)
	assert.Equal(t, players.Name("Ada"), profile.TakeSnapshot().Name)
}

func TestAssignDrawsAgainWhenTheNameIsTaken(t *testing.T) {
	store := inmemory_player_store.New()
	require.NoError(t, store.SaveProfile(t.Context(), players.NewProfile(players.AccountID{15: 1}, "bravefox42", namedAt)))
	require.NoError(t, store.SaveProfile(t.Context(), players.NewProfile(players.AccountID{15: 3}, "SlyOtter17", namedAt)))

	require.NoError(t, players.NewGeneratedNames(store, players.NewRepeatedNames("BraveFox42", "SlyOtter17", "IronOwl55")).
		Assign(t.Context(), players.AccountID{15: 2}, namedAt))

	profile, err := store.Profile(t.Context(), players.AccountID{15: 2})
	require.NoError(t, err)
	assert.Equal(t, players.Name("IronOwl55"), profile.TakeSnapshot().Name)
}

func TestAssignDrawsTenTimesAtMost(t *testing.T) {
	store := inmemory_player_store.New()
	require.NoError(t, store.SaveProfile(t.Context(), players.NewProfile(players.AccountID{15: 1}, "BraveFox42", namedAt)))
	names := players.NewRepeatedNames("BraveFox42")

	err := players.NewGeneratedNames(store, names).Assign(t.Context(), players.AccountID{15: 2}, namedAt)

	require.ErrorIs(t, err, players.ErrNoFreeName)
	assert.Equal(t, 10, names.Drawn())
	_, err = store.Profile(t.Context(), players.AccountID{15: 2})
	require.ErrorIs(t, err, players.ErrNoProfile)
}

func TestAssigningANameFailsWhenTheStoreDoes(t *testing.T) {
	store := inmemory_player_store.New()
	store.FailWith(errors.New("boom"))

	require.Error(t, players.NewGeneratedNames(store, players.NewRepeatedNames("BraveFox42")).
		Assign(t.Context(), players.AccountID{15: 1}, namedAt))
}
