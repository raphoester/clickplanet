package flags_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/cmd/migrationcheck/internal/flags"
)

func TestFlagsTakeAnOptionalBase(t *testing.T) {
	parsed, err := flags.Parse([]string{"-base", "origin/main"})
	require.NoError(t, err)
	assert.Equal(t, flags.Config{Base: "origin/main"}, parsed)

	parsed, err = flags.Parse([]string{})
	require.NoError(t, err)
	assert.Equal(t, flags.Config{}, parsed)
}

func TestFlagsRefuseAnArgument(t *testing.T) {
	_, err := flags.Parse([]string{"origin/main"})

	require.ErrorIs(t, err, flags.ErrArguments)
}
