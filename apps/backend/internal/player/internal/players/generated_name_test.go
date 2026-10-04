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
	assert.Equal(t, players.Name("BraveFox42"), profile.Name, "a second Assign draws nothing")
	assert.True(t, namedAt.Equal(profile.UpdatedAt))
}

func TestAssignLeavesAChosenNameAlone(t *testing.T) {
	store := inmemory_player_store.New()
	require.NoError(t, store.SaveProfile(t.Context(), players.Profile{Account: players.AccountID{15: 1}, Name: "Ada", UpdatedAt: namedAt}))

	require.NoError(t, players.NewGeneratedNames(store, players.NewRepeatedNames("BraveFox42")).
		Assign(t.Context(), players.AccountID{15: 1}, namedAt.Add(time.Hour)))

	profile, err := store.Profile(t.Context(), players.AccountID{15: 1})
	require.NoError(t, err)
	assert.Equal(t, players.Name("Ada"), profile.Name)
}

func TestAssignDrawsAgainWhenTheNameIsTaken(t *testing.T) {
	store := inmemory_player_store.New()
	require.NoError(t, store.SaveProfile(t.Context(), players.Profile{Account: players.AccountID{15: 1}, Name: "bravefox42", UpdatedAt: namedAt}))
	require.NoError(t, store.SaveProfile(t.Context(), players.Profile{Account: players.AccountID{15: 3}, Name: "SlyOtter17", UpdatedAt: namedAt}))

	require.NoError(t, players.NewGeneratedNames(store, players.NewRepeatedNames("BraveFox42", "SlyOtter17", "IronOwl55")).
		Assign(t.Context(), players.AccountID{15: 2}, namedAt))

	profile, err := store.Profile(t.Context(), players.AccountID{15: 2})
	require.NoError(t, err)
	assert.Equal(t, players.Name("IronOwl55"), profile.Name)
}

func TestAssignDrawsTenTimesAtMost(t *testing.T) {
	store := inmemory_player_store.New()
	require.NoError(t, store.SaveProfile(t.Context(), players.Profile{Account: players.AccountID{15: 1}, Name: "BraveFox42", UpdatedAt: namedAt}))
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

func TestOnlyALinkedAccountWithNoNameIsNameless(t *testing.T) {
	linked, named, guest, unknown := players.AccountID{15: 1}, players.AccountID{15: 2}, players.AccountID{15: 3}, players.AccountID{15: 4}
	page := []players.Stats{{Account: linked}, {Account: named}, {Account: guest}, {Account: unknown}}

	nameless := players.NamelessLinked(page,
		map[players.AccountID]players.Account{linked: {Linked: true}, named: {Linked: true}, guest: {}},
		map[players.AccountID]players.Name{named: "Ada"})

	assert.Equal(t, []players.AccountID{linked}, nameless)
	assert.Equal(t, []players.AccountID{linked, named, guest, unknown}, players.AccountsOf(page))
}
