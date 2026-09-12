package ipscope_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/ipscope"
)

func TestAnIPv4AddressIsChargedToItself(t *testing.T) {
	assert.Equal(t, "203.0.113.7", ipscope.Of("203.0.113.7"))

	// Neighbours on IPv4 are separate subscribers and keep separate budgets.
	assert.NotEqual(t, ipscope.Of("203.0.113.7"), ipscope.Of("203.0.113.8"))
}

func TestAnIPv4MappedAddressIsChargedAsIPv4(t *testing.T) {
	// Otherwise the whole v4 space collapses into the ::ffff:0:0/64 bucket and
	// every v4 caller shares one throttle.
	assert.Equal(t, ipscope.Of("203.0.113.7"), ipscope.Of("::ffff:203.0.113.7"))
}

func TestEveryAddressInAV6PrefixIsChargedTogether(t *testing.T) {
	// The point of the whole package: one subscriber owns all of these, so all
	// of them spend one budget.
	first := ipscope.Of("2001:db8:1:2::1")

	for _, other := range []string{
		"2001:db8:1:2::2",
		"2001:db8:1:2:aaaa:bbbb:cccc:dddd",
		"2001:db8:1:2:ffff:ffff:ffff:ffff",
	} {
		assert.Equal(t, first, ipscope.Of(other), "%s should share a budget", other)
	}

	assert.Equal(t, "2001:db8:1:2::/64", first)
}

func TestSeparateV6PrefixesAreChargedSeparately(t *testing.T) {
	// Two /64s are two allocations, and merging them would throttle one
	// subscriber for what another does.
	assert.NotEqual(t, ipscope.Of("2001:db8:1:2::1"), ipscope.Of("2001:db8:1:3::1"))
}

func TestAValueThatIsNotAnAddressIsReturnedUnchanged(t *testing.T) {
	// Including the empty string, which is what an unset X-Real-IP produces.
	for _, input := range []string{"", "garbage", "203.0.113.7:443"} {
		assert.Equal(t, input, ipscope.Of(input))
	}
}

func TestAZonedAddressDoesNotPanic(t *testing.T) {
	assert.Equal(t, "fe80::/64", ipscope.Of("fe80::1%eth0"))
}
