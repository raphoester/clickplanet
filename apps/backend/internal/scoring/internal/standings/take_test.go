package standings_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/standings"
)

func TestAnAccountIDIsAUUIDThatIsNotNil(t *testing.T) {
	account, err := standings.AccountIDOf("0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11")
	require.NoError(t, err)
	assert.Equal(t, "0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11", account.String())

	for _, value := range []string{"", "not-an-account", "00000000-0000-0000-0000-000000000000"} {
		_, err := standings.AccountIDOf(value)
		assert.ErrorIs(t, err, standings.ErrInvalidAccount, value)
	}
}
