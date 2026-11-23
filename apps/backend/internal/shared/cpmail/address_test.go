package cpmail_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpmail"
)

func TestAnythingButOneBareAddressIsRefused(t *testing.T) {
	for _, raw := range []string{
		"",
		"player",
		"player@",
		"@example.com",
		"Player <player@example.com>",
		"player@example.com (comment)",
		`"player one"@example.com`,
		"player@localhost",
		"player@[192.0.2.1]",
		"player@example..com",
		"player@-example.com",
		"player@example.com, other@example.com",
		strings.Repeat("a", 250) + "@example.com",
	} {
		t.Run(raw, func(t *testing.T) {
			_, err := cpmail.AddressOf(raw)

			assert.ErrorIs(t, err, cpmail.ErrAddressInvalid)
		})
	}
}

func TestAnAddressIsKeptTrimmedAndInLowerCase(t *testing.T) {
	address, err := cpmail.AddressOf("  Jane.Doe+Clicks@Example.COM ")

	require.NoError(t, err)
	assert.Equal(t, cpmail.Address("jane.doe+clicks@example.com"), address)
	assert.Equal(t, "example.com", address.Domain())
}
