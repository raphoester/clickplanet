package ipblock_test

import (
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/kernel/ipblock"
)

func TestSetContainsIPv4(t *testing.T) {
	set := parse(t, "10.0.0.0/24\n192.168.5.7/32\n")

	assert.True(t, set.Contains("10.0.0.0"), "the network address is in the range")
	assert.True(t, set.Contains("10.0.0.128"))
	assert.True(t, set.Contains("10.0.0.255"), "the broadcast address is in the range")
	assert.True(t, set.Contains("192.168.5.7"), "a /32 holds exactly its own address")

	assert.False(t, set.Contains("9.255.255.255"), "one below the range")
	assert.False(t, set.Contains("10.0.1.0"), "one above the range")
	assert.False(t, set.Contains("192.168.5.8"))
}

func TestSetContainsIPv6(t *testing.T) {
	set := parse(t, "2001:db8::/32\n")

	assert.True(t, set.Contains("2001:db8::"))
	assert.True(t, set.Contains("2001:db8:ffff:ffff:ffff:ffff:ffff:ffff"))
	assert.False(t, set.Contains("2001:db9::"))
	assert.False(t, set.Contains("2001:db7:ffff:ffff:ffff:ffff:ffff:ffff"))
}

// The two families share one 16-byte ordering, so a v4 address must not be able
// to land inside a v6 range that happens to share its bytes, or vice versa.
func TestSetKeepsTheFamiliesApart(t *testing.T) {
	v4Only := parse(t, "10.0.0.0/8\n")
	assert.False(t, v4Only.Contains("a00::1"))

	v6Only := parse(t, "::/16\n")
	assert.False(t, v6Only.Contains("10.0.0.1"))
}

// An IPv4 address written in its v4-mapped form is the same address, so it has
// to look up the same place.
func TestSetContainsV4MappedForm(t *testing.T) {
	set := parse(t, "10.0.0.0/24\n")

	assert.True(t, set.Contains("::ffff:10.0.0.1"))
}

// A list is free to write a prefix with its host bits set. It names the same
// network either way.
func TestSetMasksHostBits(t *testing.T) {
	set := parse(t, "10.0.0.7/24\n")

	assert.True(t, set.Contains("10.0.0.1"))
	assert.True(t, set.Contains("10.0.0.200"))
	assert.False(t, set.Contains("10.0.1.1"))
}

func TestSetMergesOverlappingAndAdjacentRanges(t *testing.T) {
	// A nested prefix, an overlapping one, and a neighbour that merely touches.
	set := parse(t, "10.0.0.0/24\n10.0.0.128/25\n10.0.1.0/24\n")

	assert.Equal(t, 1, set.Len(), "the three fold into one range")
	assert.True(t, set.Contains("10.0.0.1"))
	assert.True(t, set.Contains("10.0.1.255"))
	assert.False(t, set.Contains("10.0.2.0"))
}

// Merging must not fold ranges that only look adjacent. A gap of one address is
// still a gap.
func TestSetKeepsRangesWithAGapApart(t *testing.T) {
	set := parse(t, "10.0.0.0/24\n10.0.2.0/24\n")

	assert.Equal(t, 2, set.Len())
	assert.False(t, set.Contains("10.0.1.0"))
}

// A range sitting at the very top of its family is ordinary, and merging must
// not run off the end of the address space while folding it.
func TestSetHandlesTheTopOfEachFamily(t *testing.T) {
	set := parse(t, "255.255.255.254/31\nffff:ffff:ffff:ffff:ffff:ffff:ffff:fffe/127\n")

	assert.Equal(t, 2, set.Len())
	assert.True(t, set.Contains("255.255.255.255"))
	assert.True(t, set.Contains("ffff:ffff:ffff:ffff:ffff:ffff:ffff:ffff"))
	assert.False(t, set.Contains("255.255.255.253"))
}

// The two families live in separate slices, so nothing in one can merge with
// anything in the other however their bytes happen to sort.
func TestSetNeverMergesAcrossFamilies(t *testing.T) {
	set := parse(t, "0.0.0.0/0\n::/0\n")

	assert.Equal(t, 2, set.Len())
	assert.True(t, set.Contains("8.8.8.8"))
	assert.True(t, set.Contains("2001:db8::1"))
}

func TestSetSkipsBlankAndCommentLines(t *testing.T) {
	set := parse(t, "# a comment\n\n   \n10.0.0.0/24\n")

	assert.Equal(t, 1, set.Len())
	assert.True(t, set.Contains("10.0.0.1"))
}

// These lists are vendored, so a line that is not a prefix means the vendoring
// went wrong — better to fail loudly than to silently enforce a shorter list.
func TestParseRejectsAMalformedLine(t *testing.T) {
	_, err := ipblock.Parse(strings.NewReader("10.0.0.0/24\nnot-a-prefix\n"))

	require.Error(t, err)
	assert.Contains(t, err.Error(), "line 2")
}

func TestSetContainsRejectsAnUnparseableAddress(t *testing.T) {
	set := parse(t, "10.0.0.0/8\n")

	assert.False(t, set.Contains(""))
	assert.False(t, set.Contains("not-an-ip"))
	assert.False(t, set.Contains("10.0.0.1:8080"), "a host:port is not an address")
}

func TestEmptySetContainsNothing(t *testing.T) {
	set := parse(t, "")

	assert.Equal(t, 0, set.Len())
	assert.False(t, set.Contains("10.0.0.1"))
}

func TestNilSetContainsNothing(t *testing.T) {
	var set *ipblock.Set

	assert.Equal(t, 0, set.Len())
	assert.False(t, set.Contains("10.0.0.1"))
}

func parse(t *testing.T, lists ...string) *ipblock.Set {
	t.Helper()

	readers := make([]io.Reader, 0, len(lists))
	for _, l := range lists {
		readers = append(readers, strings.NewReader(l))
	}

	set, err := ipblock.Parse(readers...)
	require.NoError(t, err)
	return set
}
