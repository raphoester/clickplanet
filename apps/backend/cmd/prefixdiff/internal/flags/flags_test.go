package flags_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/cmd/prefixdiff/internal/flags"
)

func TestFlagsNameBothFiles(t *testing.T) {
	parsed, err := flags.Parse([]string{"-keep", "warp.txt", "-remove", "relay.txt"})

	require.NoError(t, err)
	assert.Equal(t, flags.Config{KeepFile: "warp.txt", RemoveFile: "relay.txt"}, parsed)
}

func TestFlagsRefuseAMissingFile(t *testing.T) {
	for _, args := range [][]string{
		{"-keep", "warp.txt"},
		{"-remove", "relay.txt"},
		{"-keep", "warp.txt", "-remove", "relay.txt", "extra.txt"},
	} {
		_, err := flags.Parse(args)
		require.ErrorIsf(t, err, flags.ErrArguments, "%v", args)
	}
}
