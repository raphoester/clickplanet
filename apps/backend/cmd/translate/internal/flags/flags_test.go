package flags_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/cmd/translate/internal/flags"
	"github.com/raphoester/clickplanet.lol-backend/internal/translation"
)

func TestTheDefaultsTranslateEveryLanguageWithSonnet(t *testing.T) {
	t.Setenv(flags.KeyVariable, "sk-ant-test") //nolint:gosec // a test's key.

	config, err := flags.Parse(nil)

	require.NoError(t, err)
	assert.Equal(t, flags.Config{Page: "../frontend", Model: "claude-sonnet-5-5", APIKey: "sk-ant-test"}, config) //nolint:gosec // a test's key.
}

func TestTheLanguagesAndThePlanAreFlags(t *testing.T) {
	t.Setenv(flags.KeyVariable, "")

	config, err := flags.Parse([]string{"-languages", "fr, pt-BR", "-plan", "-model", "claude-opus-5-5"})

	require.NoError(t, err)
	assert.Equal(t, []translation.Language{"fr", "pt-BR"}, config.Languages)
	assert.True(t, config.Plan)
	assert.Equal(t, "claude-opus-5-5", config.Model)
}

func TestAnArgumentThatIsNotAFlagIsRefused(t *testing.T) {
	_, err := flags.Parse([]string{"fr"})

	require.ErrorIs(t, err, flags.ErrArguments)
}
