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

	created, err := names.Assign(t.Context(), players.AccountID{15: 1}, "", namedAt)
	require.NoError(t, err)
	assert.True(t, created)
	created, err = names.Assign(t.Context(), players.AccountID{15: 1}, "", namedAt.Add(time.Hour))
	require.NoError(t, err)
	assert.False(t, created, "the profile was there")

	profile, err := store.Profile(t.Context(), players.AccountID{15: 1})
	require.NoError(t, err)
	assert.Equal(t, players.Name("BraveFox42"), profile.TakeSnapshot().Name, "a second Assign draws nothing")
	assert.True(t, namedAt.Equal(profile.TakeSnapshot().UpdatedAt))
}

func TestAssignLeavesAChosenNameAlone(t *testing.T) {
	store := inmemory_player_store.New()
	require.NoError(t, store.SaveProfile(t.Context(), players.NewProfile(players.AccountID{15: 1}, "Ada", namedAt)))

	created, err := players.NewGeneratedNames(store, players.NewRepeatedNames("BraveFox42")).
		Assign(t.Context(), players.AccountID{15: 1}, "Nova.42", namedAt.Add(time.Hour))
	require.NoError(t, err)
	assert.False(t, created)

	profile, err := store.Profile(t.Context(), players.AccountID{15: 1})
	require.NoError(t, err)
	assert.Equal(t, players.Name("Ada"), profile.TakeSnapshot().Name)
}

func TestAssignGivesTheSuggestedNameMadeAUsername(t *testing.T) {
	for suggested, want := range map[string]players.Name{
		"Nova.42":          "Nova_42",
		"the.longest.user": "the_longest_use",
		"Ελληνικό.όνομα12": "Ελληνικό_όνομα1",
	} {
		store := inmemory_player_store.New()
		names := players.NewRepeatedNames("BraveFox42")

		created, err := players.NewGeneratedNames(store, names).Assign(t.Context(), players.AccountID{15: 1}, suggested, namedAt)
		require.NoError(t, err)
		assert.True(t, created)

		profile, err := store.Profile(t.Context(), players.AccountID{15: 1})
		require.NoError(t, err)
		assert.Equal(t, want, profile.TakeSnapshot().Name, suggested)
		assert.Zero(t, names.Drawn(), suggested)
	}
}

func TestAssignDrawsANameWhenTheSuggestedOneIsNoUsername(t *testing.T) {
	for _, suggested := range []string{"", "ab", "guest.nova", "nova♥42"} {
		store := inmemory_player_store.New()

		_, err := players.NewGeneratedNames(store, players.NewRepeatedNames("BraveFox42")).
			Assign(t.Context(), players.AccountID{15: 1}, suggested, namedAt)
		require.NoError(t, err)

		profile, err := store.Profile(t.Context(), players.AccountID{15: 1})
		require.NoError(t, err)
		assert.Equal(t, players.Name("BraveFox42"), profile.TakeSnapshot().Name, suggested)
	}
}

func TestAssignDrawsANameWhenTheSuggestedOneIsTaken(t *testing.T) {
	store := inmemory_player_store.New()
	require.NoError(t, store.SaveProfile(t.Context(), players.NewProfile(players.AccountID{15: 1}, "nova_42", namedAt)))

	created, err := players.NewGeneratedNames(store, players.NewRepeatedNames("BraveFox42")).
		Assign(t.Context(), players.AccountID{15: 2}, "Nova.42", namedAt)
	require.NoError(t, err)
	assert.True(t, created)

	profile, err := store.Profile(t.Context(), players.AccountID{15: 2})
	require.NoError(t, err)
	assert.Equal(t, players.Name("BraveFox42"), profile.TakeSnapshot().Name)
}

func TestAssignDrawsAgainWhenTheNameIsTaken(t *testing.T) {
	store := inmemory_player_store.New()
	require.NoError(t, store.SaveProfile(t.Context(), players.NewProfile(players.AccountID{15: 1}, "bravefox42", namedAt)))
	require.NoError(t, store.SaveProfile(t.Context(), players.NewProfile(players.AccountID{15: 3}, "SlyOtter17", namedAt)))

	created, err := players.NewGeneratedNames(store, players.NewRepeatedNames("BraveFox42", "SlyOtter17", "IronOwl55")).
		Assign(t.Context(), players.AccountID{15: 2}, "", namedAt)
	require.NoError(t, err)
	assert.True(t, created)

	profile, err := store.Profile(t.Context(), players.AccountID{15: 2})
	require.NoError(t, err)
	assert.Equal(t, players.Name("IronOwl55"), profile.TakeSnapshot().Name)
}

func TestAssignDrawsTenTimesAtMost(t *testing.T) {
	store := inmemory_player_store.New()
	require.NoError(t, store.SaveProfile(t.Context(), players.NewProfile(players.AccountID{15: 1}, "BraveFox42", namedAt)))
	names := players.NewRepeatedNames("BraveFox42")

	_, err := players.NewGeneratedNames(store, names).Assign(t.Context(), players.AccountID{15: 2}, "", namedAt)

	require.ErrorIs(t, err, players.ErrNoFreeName)
	assert.Equal(t, 10, names.Drawn())
	_, err = store.Profile(t.Context(), players.AccountID{15: 2})
	require.ErrorIs(t, err, players.ErrNoProfile)
}

func TestAssigningANameFailsWhenTheStoreDoes(t *testing.T) {
	store := inmemory_player_store.New()
	store.FailWith(errors.New("boom"))

	_, err := players.NewGeneratedNames(store, players.NewRepeatedNames("BraveFox42")).
		Assign(t.Context(), players.AccountID{15: 1}, "", namedAt)
	require.Error(t, err)
}
