package players_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
)

func TestANameIsTrimmedAndStrippedOfControlCharacters(t *testing.T) {
	name, err := players.NameOf("  Ada\tLovelace\n\x00\r ")

	require.NoError(t, err)
	assert.Equal(t, players.Name("Ada Lovelace"), name)
}

func TestAnEmptyNameIsRefused(t *testing.T) {
	for _, value := range []string{"", "   ", "\n\x00"} {
		_, err := players.NameOf(value)
		require.ErrorIs(t, err, players.ErrInvalidName, "%q", value)
	}
}

func TestANameIsCountedInRunesNotBytes(t *testing.T) {
	name, err := players.NameOf(strings.Repeat("🌍", players.MaxNameLength))
	require.NoError(t, err)
	assert.Equal(t, players.Name(strings.Repeat("🌍", 24)), name)

	_, err = players.NameOf(strings.Repeat("a", players.MaxNameLength+1))
	assert.ErrorIs(t, err, players.ErrInvalidName)
}

func TestTheLengthIsCountedAfterCleaning(t *testing.T) {
	_, err := players.NameOf("  " + strings.Repeat("a", players.MaxNameLength) + "\n\n")

	assert.NoError(t, err)
}

func TestInvalidUTF8IsRefused(t *testing.T) {
	_, err := players.NameOf(string([]byte{0xff, 0xfe}))

	assert.ErrorIs(t, err, players.ErrInvalidName)
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
