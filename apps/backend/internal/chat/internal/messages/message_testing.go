//go:build testing

package messages

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// In the chat everyone reads when it names no room.
func Restored(t testing.TB, snapshot Snapshot) Message {
	t.Helper()

	if snapshot.Room == "" {
		snapshot.Room = World
	}

	message, err := snapshot.Restore()
	require.NoError(t, err)
	return message
}
