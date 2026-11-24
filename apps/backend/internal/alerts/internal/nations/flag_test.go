package nations_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/nations"
)

func TestAFlagWithARoleIsAFlagTheGameShipsWith(t *testing.T) {
	flag, ok := nations.FlagOf("ie")
	require.True(t, ok)
	assert.Equal(t, nations.Flag("ie"), flag)

	for _, code := range []string{"", "IE", "zz", "ireland", "r-k3p9x2ma0b", "xb"} {
		_, ok := nations.FlagOf(code)
		assert.False(t, ok, code)
	}
}

func TestAFlagIsNamedInEnglish(t *testing.T) {
	for code, name := range map[nations.Flag]string{
		"ie": "Ireland",
		"gb": "United Kingdom",
		"us": "United States",
		"pw": "Palau",
		"xk": "Kosovo",
	} {
		assert.Equal(t, name, code.Name(), code)
	}
}

func TestAFlagIsDrawnByItsTwoLetters(t *testing.T) {
	assert.Equal(t, "🇮🇪", nations.Flag("ie").Emoji())
	assert.Equal(t, "🇵🇼", nations.Flag("pw").Emoji())
}
