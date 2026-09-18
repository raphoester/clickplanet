package players_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/inmemory_player_store"
)

func TestAGuestIsShownByItsCodeAndAPlayerByItsName(t *testing.T) {
	assert.Equal(t, "guest_a1b2c3", players.DisplayNameOf("", "a1b2c3"))
	assert.Equal(t, "Ada", players.DisplayNameOf("Ada", "a1b2c3"))
}

func TestAGuestCodeIsSixLowercaseHexCharacters(t *testing.T) {
	code, err := players.GuestCodeOf("a1b2c3")
	require.NoError(t, err)
	assert.Equal(t, players.GuestCode("a1b2c3"), code)

	for _, value := range []string{"", "a1b2c", "a1b2c3d", "A1B2C3", "g1b2c3", "a1 2c3"} {
		_, err := players.GuestCodeOf(value)
		require.ErrorIs(t, err, players.ErrInvalidGuestCode, "%q", value)
	}
}

func TestAssignGivesAnAccountItsCodeOnce(t *testing.T) {
	store := inmemory_player_store.New()
	codes := players.NewGuestCodes(store, &players.SequentialCodes{})

	require.NoError(t, codes.Assign(t.Context(), players.AccountID{15: 1}))
	require.NoError(t, codes.Assign(t.Context(), players.AccountID{15: 1}))
	require.NoError(t, codes.Assign(t.Context(), players.AccountID{15: 2}))

	first, err := store.GuestCode(t.Context(), players.AccountID{15: 1})
	require.NoError(t, err)
	assert.Equal(t, players.GuestCode("000001"), first, "a second Assign draws nothing")
	second, err := store.GuestCode(t.Context(), players.AccountID{15: 2})
	require.NoError(t, err)
	assert.Equal(t, players.GuestCode("000002"), second)
}

func TestAssignDrawsAgainWhenTheCodeIsTaken(t *testing.T) {
	store := inmemory_player_store.New()
	require.NoError(t, store.SaveGuestCode(t.Context(), players.AccountID{15: 1}, "aaaaaa"))
	codes := players.NewGuestCodes(store, players.NewRepeatedCodes("aaaaaa", "bbbbbb"))

	require.NoError(t, codes.Assign(t.Context(), players.AccountID{15: 2}))

	code, err := store.GuestCode(t.Context(), players.AccountID{15: 2})
	require.NoError(t, err)
	assert.Equal(t, players.GuestCode("bbbbbb"), code)
}

func TestAssignGivesUpOnAGeneratorThatRepeatsItself(t *testing.T) {
	store := inmemory_player_store.New()
	require.NoError(t, store.SaveGuestCode(t.Context(), players.AccountID{15: 1}, "aaaaaa"))
	codes := players.NewGuestCodes(store, players.NewRepeatedCodes("aaaaaa"))

	err := codes.Assign(t.Context(), players.AccountID{15: 2})

	require.ErrorIs(t, err, players.ErrNoFreeGuestCode)
	_, err = store.GuestCode(t.Context(), players.AccountID{15: 2})
	require.ErrorIs(t, err, players.ErrNoGuestCode)
}

func TestAssignFailsWhenTheStoreDoes(t *testing.T) {
	store := inmemory_player_store.New()
	store.FailWith(errors.New("boom"))
	codes := players.NewGuestCodes(store, &players.SequentialCodes{})

	require.Error(t, codes.Assign(t.Context(), players.AccountID{15: 1}))
}
