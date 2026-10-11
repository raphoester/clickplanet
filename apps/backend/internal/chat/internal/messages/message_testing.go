//go:build testing

package messages

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Restored(t testing.TB, snapshot Snapshot) Message {
	t.Helper()

	message, err := snapshot.Restore()
	require.NoError(t, err)
	return message
}
