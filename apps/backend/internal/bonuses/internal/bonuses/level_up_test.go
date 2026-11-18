package bonuses

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAGrantIdThatIsNoIdIsRefused(t *testing.T) {
	for _, bad := range []string{"", "a-token", "00000000-0000-0000-0000-000000000000"} {
		_, err := GrantIDOf(bad)
		require.ErrorIs(t, err, ErrInvalidGrant, "%q", bad)
	}
}
