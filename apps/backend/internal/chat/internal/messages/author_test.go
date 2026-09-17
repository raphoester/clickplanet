package messages_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
)

func TestAGuestNameIsTheCleanedNamePrefixed(t *testing.T) {
	name, err := limits.GuestName("  Bob\tthe builder\n")

	require.NoError(t, err)
	assert.Equal(t, "guest_Bob the builder", name)
}

func TestTheGuestNameLimitIsOnWhatTheGuestTyped(t *testing.T) {
	name, err := limits.GuestName(strings.Repeat("a", 24))
	require.NoError(t, err)
	assert.Equal(t, "guest_"+strings.Repeat("a", 24), name)

	_, err = limits.GuestName(strings.Repeat("a", 25))
	assert.ErrorIs(t, err, messages.ErrInvalidMessage)
}

func TestAnEmptyGuestNameIsRefused(t *testing.T) {
	_, err := limits.GuestName(" \n")

	assert.ErrorIs(t, err, messages.ErrInvalidMessage)
}

func TestAnAccountIsReadOffTheContextsValue(t *testing.T) {
	assert.Equal(t, "0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11", messages.AccountIDOf("0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11").String())

	for _, value := range []string{"", "not-a-uuid"} {
		assert.Equal(t, cpsession.NoAccount, messages.AccountIDOf(value), "%q", value)
	}
}
