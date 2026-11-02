package nations_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/nations"
)

func TestAFlagIsAFlagTheGameKnows(t *testing.T) {
	flag, err := nations.FlagOf("ie")
	require.NoError(t, err)
	assert.Equal(t, nations.Flag("ie"), flag)

	for _, code := range []string{"", "IE", "zz", "ireland"} {
		_, err := nations.FlagOf(code)
		assert.ErrorIs(t, err, nations.ErrUnknownFlag, code)
	}
}

func TestAFlagIsNamedInEnglish(t *testing.T) {
	for code, name := range map[nations.Flag]string{
		"ie": "Ireland",
		"gb": "United Kingdom",
		"us": "United States",
		"pw": "Palau",
		"xk": "Kosovo",
		"xb": "Brittany",
	} {
		assert.Equal(t, name, code.Name(), code)
	}
}

func TestAFlagIsDrawnByItsTwoLetters(t *testing.T) {
	assert.Equal(t, "🇮🇪", nations.Flag("ie").Emoji())
	assert.Equal(t, "🇵🇼", nations.Flag("pw").Emoji())
	assert.Equal(t, "🏴", nations.Flag("xb").Emoji())
}
