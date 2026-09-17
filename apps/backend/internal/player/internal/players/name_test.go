package players_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
)

func TestANameKeepsTheCaseItWasTypedIn(t *testing.T) {
	name, err := players.NameOf("Ada_Lovelace_1815")

	require.NoError(t, err)
	assert.Equal(t, players.Name("Ada_Lovelace_1815"), name)
}

func TestANameIsThreeToTwentyCharacters(t *testing.T) {
	for _, value := range []string{"abc", strings.Repeat("a", players.MaxNameLength)} {
		_, err := players.NameOf(value)
		require.NoError(t, err, "%q", value)
	}

	for _, value := range []string{"", "ab", strings.Repeat("a", players.MaxNameLength+1)} {
		_, err := players.NameOf(value)
		require.ErrorIs(t, err, players.ErrInvalidName, "%q", value)
	}
}

func TestANameIsOnlyASCIILettersDigitsAndUnderscores(t *testing.T) {
	for _, value := range []string{"Ada Lovelace", " Ada", "Ada\n", "Ada-L", "Émile", "🌍🌍🌍", "Ada.L", string([]byte{0xff, 0xfe, 0xfd})} {
		_, err := players.NameOf(value)
		require.ErrorIs(t, err, players.ErrInvalidName, "%q", value)
	}
}

func TestANameNeverStartsWithTheGuestPrefixInAnyCase(t *testing.T) {
	for _, value := range []string{"guest_ada", "GUEST_ada", "Guest_", "gUeSt_1"} {
		_, err := players.NameOf(value)
		require.ErrorIs(t, err, players.ErrInvalidName, "%q", value)
	}

	for _, value := range []string{"guest", "guestada", "ada_guest_"} {
		_, err := players.NameOf(value)
		require.NoError(t, err, "%q", value)
	}
}

func TestTwoNamesThatDifferOnlyInCaseFoldTheSame(t *testing.T) {
	assert.Equal(t, players.Name("ada_L").Folded(), players.Name("ADA_l").Folded())
}

func TestAnAccountIDIsAUUIDAndNeverTheNilOne(t *testing.T) {
	account, err := players.AccountIDOf("0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11")
	require.NoError(t, err)
	assert.Equal(t, "0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11", account.String())

	for _, value := range []string{"", "not-a-uuid", "00000000-0000-0000-0000-000000000000"} {
		_, err := players.AccountIDOf(value)
		require.ErrorIs(t, err, players.ErrInvalidAccount, "%q", value)
	}
}
