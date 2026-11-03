package cpipblock_test

import (
	"io"
	"net/netip"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpipblock"
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

func TestSetKeepsTheFamiliesApart(t *testing.T) {
	v4Only := parse(t, "10.0.0.0/8\n")
	assert.False(t, v4Only.Contains("a00::1"))

	v6Only := parse(t, "::/16\n")
	assert.False(t, v6Only.Contains("10.0.0.1"))
}

func TestSetContainsV4MappedForm(t *testing.T) {
	set := parse(t, "10.0.0.0/24\n")

	assert.True(t, set.Contains("::ffff:10.0.0.1"))
}

func TestSetMasksHostBits(t *testing.T) {
	set := parse(t, "10.0.0.7/24\n")

	assert.True(t, set.Contains("10.0.0.1"))
	assert.True(t, set.Contains("10.0.0.200"))
	assert.False(t, set.Contains("10.0.1.1"))
}

func TestSetMergesOverlappingAndAdjacentRanges(t *testing.T) {
	set := parse(t, "10.0.0.0/24\n10.0.0.128/25\n10.0.1.0/24\n")

	assert.Equal(t, 1, set.Len(), "the three fold into one range")
	assert.True(t, set.Contains("10.0.0.1"))
	assert.True(t, set.Contains("10.0.1.255"))
	assert.False(t, set.Contains("10.0.2.0"))
}

func TestSetKeepsRangesWithAGapApart(t *testing.T) {
	set := parse(t, "10.0.0.0/24\n10.0.2.0/24\n")

	assert.Equal(t, 2, set.Len())
	assert.False(t, set.Contains("10.0.1.0"))
}

func TestSetHandlesTheTopOfEachFamily(t *testing.T) {
	set := parse(t, "255.255.255.254/31\nffff:ffff:ffff:ffff:ffff:ffff:ffff:fffe/127\n")

	assert.Equal(t, 2, set.Len())
	assert.True(t, set.Contains("255.255.255.255"))
	assert.True(t, set.Contains("ffff:ffff:ffff:ffff:ffff:ffff:ffff:ffff"))
	assert.False(t, set.Contains("255.255.255.253"))
}

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

func TestParseRejectsAMalformedLine(t *testing.T) {
	_, err := cpipblock.Parse(strings.NewReader("10.0.0.0/24\nnot-a-prefix\n"))

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
	var set *cpipblock.Set

	assert.Equal(t, 0, set.Len())
	assert.False(t, set.Contains("10.0.0.1"))
}

func TestSetDifferenceCutsAHole(t *testing.T) {
	set := parse(t, "10.0.0.0/24\n").Difference(parse(t, "10.0.0.64/26\n"))

	assert.Equal(t, 2, set.Len())
	assert.True(t, set.Contains("10.0.0.63"))
	assert.False(t, set.Contains("10.0.0.64"))
	assert.False(t, set.Contains("10.0.0.127"))
	assert.True(t, set.Contains("10.0.0.128"))
}

func TestSetDifferenceDropsAWhollyRemovedRange(t *testing.T) {
	set := parse(t, "10.0.0.0/24\n10.0.2.0/24\n").Difference(parse(t, "10.0.0.0/16\n"))

	assert.Equal(t, 0, set.Len())
}

func TestSetDifferenceLetsOneRemovalSpanSeveralRanges(t *testing.T) {
	set := parse(t, "10.0.0.0/24\n10.0.2.0/24\n").Difference(parse(t, "10.0.0.128/25\n10.0.1.0/24\n10.0.2.0/25\n"))

	assert.Equal(t, 2, set.Len())
	assert.True(t, set.Contains("10.0.0.127"))
	assert.False(t, set.Contains("10.0.0.128"))
	assert.False(t, set.Contains("10.0.2.127"))
	assert.True(t, set.Contains("10.0.2.128"))
}

func TestSetDifferenceTakesSeveralRemovalsOutOfOneRange(t *testing.T) {
	set := parse(t, "10.0.0.0/24\n").Difference(parse(t, "9.0.0.0/8\n10.0.0.10/32\n10.0.0.20/32\n11.0.0.0/8\n"))

	assert.Equal(t, 3, set.Len())
	assert.True(t, set.Contains("10.0.0.0"))
	assert.False(t, set.Contains("10.0.0.10"))
	assert.True(t, set.Contains("10.0.0.15"))
	assert.False(t, set.Contains("10.0.0.20"))
	assert.True(t, set.Contains("10.0.0.255"))
}

func TestSetDifferenceKeepsTheFamiliesApart(t *testing.T) {
	set := parse(t, "10.0.0.0/8\n2001:db8::/32\n").Difference(parse(t, "::/0\n"))

	assert.True(t, set.Contains("10.1.2.3"))
	assert.False(t, set.Contains("2001:db8::1"))
}

func TestSetDifferenceHandlesTheTopOfEachFamily(t *testing.T) {
	set := parse(t, "255.255.255.0/24\nffff:ffff:ffff:ffff:ffff:ffff:ffff:ff00/120\n").
		Difference(parse(t, "255.255.255.255/32\nffff:ffff:ffff:ffff:ffff:ffff:ffff:ffff/128\n"))

	assert.True(t, set.Contains("255.255.255.254"))
	assert.False(t, set.Contains("255.255.255.255"))
	assert.True(t, set.Contains("ffff:ffff:ffff:ffff:ffff:ffff:ffff:fffe"))
	assert.False(t, set.Contains("ffff:ffff:ffff:ffff:ffff:ffff:ffff:ffff"))
}

func TestSetPrefixesAreTheFewestThatTileEachRange(t *testing.T) {
	set := parse(t, "10.0.0.1/32\n10.0.0.2/31\n10.0.0.4/31\n10.0.0.6/32\n10.1.0.0/25\n10.1.0.128/25\n")

	assert.Equal(t, prefixes("10.0.0.1/32", "10.0.0.2/31", "10.0.0.4/31", "10.0.0.6/32", "10.1.0.0/24"), set.Prefixes())
}

func TestSetPrefixesWriteEachFamilyInItsOwnForm(t *testing.T) {
	set := parse(t, "0.0.0.0/0\n2001:db8::/33\n2001:db8:8000::/33\n")

	assert.Equal(t, prefixes("0.0.0.0/0", "2001:db8::/32"), set.Prefixes())
}

func TestSetPrefixesReachTheTopOfEachFamily(t *testing.T) {
	set := parse(t, "255.255.255.254/31\nffff:ffff:ffff:ffff:ffff:ffff:ffff:fffe/127\n")

	assert.Equal(t, prefixes("255.255.255.254/31", "ffff:ffff:ffff:ffff:ffff:ffff:ffff:fffe/127"), set.Prefixes())
}

func TestParseFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "prefixes.txt")
	require.NoError(t, os.WriteFile(path, []byte("10.0.0.0/24\n"), 0o600))

	set, err := cpipblock.ParseFile(path)
	require.NoError(t, err)
	assert.True(t, set.Contains("10.0.0.1"))

	_, err = cpipblock.ParseFile(filepath.Join(t.TempDir(), "missing.txt"))
	require.Error(t, err)
}

func prefixes(texts ...string) []netip.Prefix {
	out := make([]netip.Prefix, 0, len(texts))
	for _, text := range texts {
		out = append(out, netip.MustParsePrefix(text))
	}
	return out
}

func parse(t *testing.T, lists ...string) *cpipblock.Set {
	t.Helper()

	readers := make([]io.Reader, 0, len(lists))
	for _, l := range lists {
		readers = append(readers, strings.NewReader(l))
	}

	set, err := cpipblock.Parse(readers...)
	require.NoError(t, err)
	return set
}
