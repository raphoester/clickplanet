package notes_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes"
)

var ada = notes.AccountID{15: 1}

func TestAnAccountIDIsReadFromItsText(t *testing.T) {
	account, err := notes.AccountIDOf(ada.String())

	require.NoError(t, err)
	assert.Equal(t, ada, account)
}

func TestWhatIsNotAnAccountIDIsRefused(t *testing.T) {
	for _, value := range []string{"", "not-an-account", "00000000-0000-0000-0000-000000000000"} {
		_, err := notes.AccountIDOf(value)

		require.ErrorIs(t, err, notes.ErrInvalidAccount, value)
	}
}
