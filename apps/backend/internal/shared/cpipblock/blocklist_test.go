package cpipblock_test

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpipblock"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpipblock/cpdata"
)

func TestVendoredListsParse(t *testing.T) {
	blocklist := cpipblock.New(cpipblock.Config{Enabled: true, IncludeDatacenters: true})
	require.NoError(t, blocklist.Load())

	sizes := blocklist.Sizes()

	assert.Greater(t, sizes[cpipblock.ListVPN], 5_000)
	assert.Greater(t, sizes[cpipblock.ListDatacenter], 20_000)
}

func TestVendoredVPNListBlocksAKnownRange(t *testing.T) {
	blocklist := cpipblock.New(cpipblock.Config{Enabled: true})
	require.NoError(t, blocklist.Load())

	list, blocked := blocklist.Blocked("2.26.157.1")
	assert.True(t, blocked)
	assert.Equal(t, cpipblock.ListVPN, list)

	_, blocked = blocklist.Blocked("8.8.8.8")
	assert.False(t, blocked, "a well-known resolver is not a VPN egress")
}

func TestVendoredSourcesAreNotTruncated(t *testing.T) {
	for name, data := range map[string][]byte{
		"providers": cpdata.VPNProviders,
		"az0":       cpdata.VPNAz0,
		"tor exits": cpdata.TorExits,
		"netnames":  cpdata.VPNNetnames,
	} {
		set, err := cpipblock.Parse(bytes.NewReader(data))
		require.NoErrorf(t, err, "%s should parse", name)
		assert.Positivef(t, set.Len(), "%s should not be empty", name)
	}

	for name, data := range map[string][]byte{"providers": cpdata.VPNProviders, "az0": cpdata.VPNAz0} {
		set, err := cpipblock.Parse(bytes.NewReader(data))
		require.NoError(t, err)
		assert.Greaterf(t, set.Len(), 5_000, "%s looks truncated", name)
	}
}

func TestVPNListBlocksFirefoxVPN(t *testing.T) {
	blocklist := cpipblock.New(cpipblock.Config{Enabled: true})
	require.NoError(t, blocklist.Load())

	for _, ip := range []string{"2a00:8c40:f0c8:8e3::1", "63.245.216.1"} {
		list, blocked := blocklist.Blocked(ip)
		require.Truef(t, blocked, "%s should be refused", ip)
		require.Equal(t, cpipblock.ListVPN, list)
	}
}

func TestDatacenterListIsOffUnlessAskedFor(t *testing.T) {
	off := cpipblock.New(cpipblock.Config{Enabled: true})
	require.NoError(t, off.Load())
	assert.NotContains(t, off.Sizes(), cpipblock.ListDatacenter)

	on := cpipblock.New(cpipblock.Config{Enabled: true, IncludeDatacenters: true})
	require.NoError(t, on.Load())
	assert.Contains(t, on.Sizes(), cpipblock.ListDatacenter)
}

func TestAllowlistWinsOverTheLists(t *testing.T) {
	blocklist := cpipblock.New(cpipblock.Config{
		Enabled: true,
		Allow:   []string{"2.26.157.0/24"},
	})
	require.NoError(t, blocklist.Load())

	_, blocked := blocklist.Blocked("2.26.157.1")
	assert.False(t, blocked)
}

func TestAllowlistRejectsAMalformedEntry(t *testing.T) {
	err := cpipblock.New(cpipblock.Config{Enabled: true, Allow: []string{"10.0.0.1"}}).Load()

	require.Error(t, err, "a bare address is not a prefix; say /32")
	assert.Contains(t, err.Error(), "allowlist")
}

func TestDisabledBlocklistBlocksNothing(t *testing.T) {
	blocklist := cpipblock.New(cpipblock.Config{Enabled: false})
	require.NoError(t, blocklist.Load())
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
