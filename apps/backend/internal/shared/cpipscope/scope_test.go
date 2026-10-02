package cpipscope_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpipscope"
)

func TestAnIPv4AddressIsChargedToItself(t *testing.T) {
	assert.Equal(t, "203.0.113.7", cpipscope.Of("203.0.113.7"))

	assert.NotEqual(t, cpipscope.Of("203.0.113.7"), cpipscope.Of("203.0.113.8"))
}

func TestAnIPv4MappedAddressIsChargedAsIPv4(t *testing.T) {
	assert.Equal(t, cpipscope.Of("203.0.113.7"), cpipscope.Of("::ffff:203.0.113.7"))
}

func TestEveryAddressInAV6PrefixIsChargedTogether(t *testing.T) {
	first := cpipscope.Of("2001:db8:1:2::1")

	for _, other := range []string{
		"2001:db8:1:2::2",
		"2001:db8:1:2:aaaa:bbbb:cccc:dddd",
		"2001:db8:1:2:ffff:ffff:ffff:ffff",
	} {
		assert.Equal(t, first, cpipscope.Of(other), "%s should share a budget", other)
	}

	assert.Equal(t, "2001:db8:1:2::/64", first)
}

func TestSeparateV6PrefixesAreChargedSeparately(t *testing.T) {
	assert.NotEqual(t, cpipscope.Of("2001:db8:1:2::1"), cpipscope.Of("2001:db8:1:3::1"))
}

func TestAValueThatIsNotAnAddressIsReturnedUnchanged(t *testing.T) {
	for _, input := range []string{"", "garbage", "203.0.113.7:443"} {
		assert.Equal(t, input, cpipscope.Of(input))
	}
}

func TestAZonedAddressDoesNotPanic(t *testing.T) {
	assert.Equal(t, "fe80::/64", cpipscope.Of("fe80::1%eth0"))
}

func TestParseTakesAnAddressOrAScope(t *testing.T) {
	for text, want := range map[string]string{
		"203.0.113.7":       "203.0.113.7",
		"2001:db8:1:2::1":   "2001:db8:1:2::/64",
		"2001:db8:1:2::/64": "2001:db8:1:2::/64",
	} {
		scope, ok := cpipscope.Parse(text)
		assert.True(t, ok, text)
		assert.Equal(t, want, scope, text)
	}

	for _, text := range []string{"", "bot", "203.0.113.7/32", "2001:db8:1:2::1/64", "2001:db8::/48"} {
		_, ok := cpipscope.Parse(text)
		assert.False(t, ok, text)
	}
}
