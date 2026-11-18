package flags_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/cmd/translations/internal/flags"
	"github.com/raphoester/clickplanet.lol-backend/internal/translation"
)

func TestAPlanTakesEveryLanguageAndBatchesOf400ByDefault(t *testing.T) {
	config, err := flags.Parse([]string{"plan", "-work", "out"})

	require.NoError(t, err)
	assert.Equal(t, flags.Config{Command: flags.Plan, Page: "../frontend", Work: "out", Batch: 400}, config)
}

func TestTheLanguagesAreAFlag(t *testing.T) {
	config, err := flags.Parse([]string{"apply", "-work", "out", "-languages", "fr, pt-BR"})

	require.NoError(t, err)
	assert.Equal(t, []translation.Language{"fr", "pt-BR"}, config.Languages)
}

func TestWrongArgumentsAreRefused(t *testing.T) {
	for _, args := range [][]string{
		nil,
		{"translate", "-work", "out"},
		{"plan"},
		{"plan", "-work", "out", "fr"},
		{"plan", "-work", "out", "-batch", "0"},
		{"apply", "-work", "out", "-batch", "10"},
	} {
		_, err := flags.Parse(args)

		require.ErrorIs(t, err, flags.ErrArguments, "%q", args)
	}
}
