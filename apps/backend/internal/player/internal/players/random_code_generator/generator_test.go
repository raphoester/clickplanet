package random_code_generator_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/random_code_generator"
)

func TestACodeIsSixHexCharacters(t *testing.T) {
	code, err := random_code_generator.Generator{}.NewGuestCode()

	require.NoError(t, err)
	_, err = players.GuestCodeOf(string(code))
	require.NoError(t, err)
}

func TestTwoCodesDiffer(t *testing.T) {
	first, err := random_code_generator.Generator{}.NewGuestCode()
	require.NoError(t, err)
	second, err := random_code_generator.Generator{}.NewGuestCode()
	require.NoError(t, err)

	assert.NotEqual(t, first, second)
}
