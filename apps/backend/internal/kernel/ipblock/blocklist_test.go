package ipblock_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/kernel/ipblock"
)

// The lists are vendored, so a parse failure is a broken checkout rather than a
// runtime condition. Asserting it here is what lets New treat the error as
// impossible in practice, and what catches a bad `make vpn-lists` before it
// ships.
func TestVendoredListsParse(t *testing.T) {
	blocklist, err := ipblock.New(ipblock.Config{Enabled: true, IncludeDatacenters: true})
	require.NoError(t, err)

	sizes := blocklist.Sizes()

	// Loose bounds on purpose: the lists are refreshed from upstream and a tight
	// assertion would fail on every refresh. These only catch a list that came
	// back empty or truncated.
	assert.Greater(t, sizes[ipblock.ListVPN], 5_000)
	assert.Greater(t, sizes[ipblock.ListDatacenter], 20_000)
}

// A spot check against real entries, so the vendored data is exercised end to
// end and not just counted.
func TestVendoredVPNListBlocksAKnownRange(t *testing.T) {
	blocklist, err := ipblock.New(ipblock.Config{Enabled: true})
	require.NoError(t, err)

	list, blocked := blocklist.Blocked("2.26.157.1") // 2.26.157.0/24, first line of vpn_ipv4.txt
	assert.True(t, blocked)
	assert.Equal(t, ipblock.ListVPN, list)

	_, blocked = blocklist.Blocked("8.8.8.8")
	assert.False(t, blocked, "a well-known resolver is not a VPN egress")
}

// Without IncludeDatacenters the datacenter list is never even parsed, so a
// hosting range has to come back clean.
func TestDatacenterListIsOffUnlessAskedFor(t *testing.T) {
	off, err := ipblock.New(ipblock.Config{Enabled: true})
	require.NoError(t, err)
	assert.NotContains(t, off.Sizes(), ipblock.ListDatacenter)

	on, err := ipblock.New(ipblock.Config{Enabled: true, IncludeDatacenters: true})
	require.NoError(t, err)
	assert.Contains(t, on.Sizes(), ipblock.ListDatacenter)
}

// The allowlist is the escape hatch for an address the vendored lists get
// wrong, so nothing may override it.
func TestAllowlistWinsOverTheLists(t *testing.T) {
	blocklist, err := ipblock.New(ipblock.Config{
		Enabled: true,
		Allow:   []string{"2.26.157.0/24"},
	})
	require.NoError(t, err)

	_, blocked := blocklist.Blocked("2.26.157.1")
	assert.False(t, blocked)
}

func TestAllowlistRejectsAMalformedEntry(t *testing.T) {
	_, err := ipblock.New(ipblock.Config{Enabled: true, Allow: []string{"10.0.0.1"}})

	require.Error(t, err, "a bare address is not a prefix; say /32")
	assert.Contains(t, err.Error(), "allowlist")
}

// Disabled means no list is parsed and none is held. The nil blocklist answers
// every lookup false, so there is no second no-op implementation to keep in step.
func TestDisabledBlocklistBlocksNothing(t *testing.T) {
	blocklist, err := ipblock.New(ipblock.Config{Enabled: false})
	require.NoError(t, err)
	require.Nil(t, blocklist)

	list, blocked := blocklist.Blocked("2.26.157.1")
	assert.False(t, blocked)
	assert.Empty(t, list)
	assert.Nil(t, blocklist.Sizes())
}

// NewDenyList is the operator-maintained half: no vendored lists behind it,
// prefixes straight from config.
func TestNewDenyList(t *testing.T) {
	t.Run("refuses the listed prefixes and nothing else", func(t *testing.T) {
		blocklist, err := ipblock.NewDenyList([]string{"9.9.9.9/32", "203.0.113.0/24"})
		require.NoError(t, err)

		for _, ip := range []string{"9.9.9.9", "203.0.113.1", "203.0.113.255"} {
			list, blocked := blocklist.Blocked(ip)
			require.Truef(t, blocked, "%s should be refused", ip)
			require.Equal(t, ipblock.ListDeny, list)
		}

		_, blocked := blocklist.Blocked("8.8.8.8")
		require.False(t, blocked)
	})

	t.Run("an empty list is a nil blocklist that refuses nothing", func(t *testing.T) {
		blocklist, err := ipblock.NewDenyList(nil)
		require.NoError(t, err)
		require.Nil(t, blocklist)

		_, blocked := blocklist.Blocked("9.9.9.9")
		require.False(t, blocked)
	})

	// Same bargain the vendored lists make: a malformed entry means the config
	// is wrong, and failing at startup beats silently refusing nobody.
	t.Run("a malformed entry is an error", func(t *testing.T) {
		_, err := ipblock.NewDenyList([]string{"9.9.9.9"}) // needs /32
		require.Error(t, err)
	})
}
