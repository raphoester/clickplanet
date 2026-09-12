package cpipblock_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpipblock"
)

func TestVendoredListsParse(t *testing.T) {
	blocklist, err := cpipblock.New(cpipblock.Config{Enabled: true, IncludeDatacenters: true})
	require.NoError(t, err)

	sizes := blocklist.Sizes()

	assert.Greater(t, sizes[cpipblock.ListVPN], 5_000)
	assert.Greater(t, sizes[cpipblock.ListDatacenter], 20_000)
}

func TestVendoredVPNListBlocksAKnownRange(t *testing.T) {
	blocklist, err := cpipblock.New(cpipblock.Config{Enabled: true})
	require.NoError(t, err)

	list, blocked := blocklist.Blocked("2.26.157.1")
	assert.True(t, blocked)
	assert.Equal(t, cpipblock.ListVPN, list)

	_, blocked = blocklist.Blocked("8.8.8.8")
	assert.False(t, blocked, "a well-known resolver is not a VPN egress")
}

func TestDatacenterListIsOffUnlessAskedFor(t *testing.T) {
	off, err := cpipblock.New(cpipblock.Config{Enabled: true})
	require.NoError(t, err)
	assert.NotContains(t, off.Sizes(), cpipblock.ListDatacenter)

	on, err := cpipblock.New(cpipblock.Config{Enabled: true, IncludeDatacenters: true})
	require.NoError(t, err)
	assert.Contains(t, on.Sizes(), cpipblock.ListDatacenter)
}

func TestAllowlistWinsOverTheLists(t *testing.T) {
	blocklist, err := cpipblock.New(cpipblock.Config{
		Enabled: true,
		Allow:   []string{"2.26.157.0/24"},
	})
	require.NoError(t, err)

	_, blocked := blocklist.Blocked("2.26.157.1")
	assert.False(t, blocked)
}

func TestAllowlistRejectsAMalformedEntry(t *testing.T) {
	_, err := cpipblock.New(cpipblock.Config{Enabled: true, Allow: []string{"10.0.0.1"}})

	require.Error(t, err, "a bare address is not a prefix; say /32")
	assert.Contains(t, err.Error(), "allowlist")
}

func TestDisabledBlocklistBlocksNothing(t *testing.T) {
	blocklist, err := cpipblock.New(cpipblock.Config{Enabled: false})
	require.NoError(t, err)
	require.Nil(t, blocklist)

	list, blocked := blocklist.Blocked("2.26.157.1")
	assert.False(t, blocked)
	assert.Empty(t, list)
	assert.Nil(t, blocklist.Sizes())
}

func TestNewDenyList(t *testing.T) {
	t.Run("refuses the listed prefixes and nothing else", func(t *testing.T) {
		blocklist, err := cpipblock.NewDenyList([]string{"9.9.9.9/32", "203.0.113.0/24"})
		require.NoError(t, err)

		for _, ip := range []string{"9.9.9.9", "203.0.113.1", "203.0.113.255"} {
			list, blocked := blocklist.Blocked(ip)
			require.Truef(t, blocked, "%s should be refused", ip)
			require.Equal(t, cpipblock.ListDeny, list)
		}

		_, blocked := blocklist.Blocked("8.8.8.8")
		require.False(t, blocked)
	})

	t.Run("an empty list is a nil blocklist that refuses nothing", func(t *testing.T) {
		blocklist, err := cpipblock.NewDenyList(nil)
		require.NoError(t, err)
		require.Nil(t, blocklist)

		_, blocked := blocklist.Blocked("9.9.9.9")
		require.False(t, blocked)
	})

	t.Run("a malformed entry is an error", func(t *testing.T) {
		_, err := cpipblock.NewDenyList([]string{"9.9.9.9"})
		require.Error(t, err)
	})
}
