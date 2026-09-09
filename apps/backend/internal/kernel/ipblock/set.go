// Package ipblock answers one question fast: is this address inside one of
// these tens of thousands of CIDRs?
//
// Like ratelimit, it holds its state in this process and nothing else needs to
// know about it. A Set is immutable once built, so the handler goroutines that
// share one need no locking.
package ipblock

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net/netip"
	"slices"
	"strings"
)

// Set is a sorted, merged collection of address ranges. Every address is held
// as its 16-byte form, IPv4 as the v4-mapped ::ffff:a.b.c.d, because big-endian
// bytes compare in the same order as the numbers they encode — which is what
// makes one binary search serve both families.
//
// The two families are nonetheless kept in separate slices. They cannot share
// one, because the v4-mapped block ::ffff:0:0/96 sits *inside* short v6
// prefixes: with a single ordering, a list containing ::/16 would silently
// swallow every IPv4 address on the internet.
type Set struct {
	v4, v6 []addrRange
}

type addrRange struct {
	lo, hi [16]byte
}

// Parse reads one CIDR per line from each reader. Blank lines and lines
// starting with '#' are skipped; anything else that is not a prefix is an
// error, because these lists are vendored and a malformed line means the
// vendoring went wrong rather than that the world is messy.
func Parse(readers ...io.Reader) (*Set, error) {
	var v4, v6 []addrRange

	for _, r := range readers {
		scanner := bufio.NewScanner(r)
		for line := 1; scanner.Scan(); line++ {
			text := strings.TrimSpace(scanner.Text())
			if text == "" || strings.HasPrefix(text, "#") {
				continue
			}

			prefix, err := netip.ParsePrefix(text)
			if err != nil {
				return nil, fmt.Errorf("line %d: failed to parse prefix %q: %w", line, text, err)
			}

			if prefix.Addr().Unmap().Is4() {
				v4 = append(v4, rangeOf(prefix))
			} else {
				v6 = append(v6, rangeOf(prefix))
			}
		}

		if err := scanner.Err(); err != nil {
			return nil, fmt.Errorf("failed to read prefixes: %w", err)
		}
	}

	return &Set{v4: merge(v4), v6: merge(v6)}, nil
}

// Contains reports whether ip falls in any range. An address it cannot parse is
// not contained: this list exists to refuse the known-bad, not to refuse
// everything it fails to read.
func (s *Set) Contains(ip string) bool {
	if s == nil {
		return false
	}

	addr, err := netip.ParseAddr(ip)
	if err != nil {
		return false
	}

	// Unmap first: an IPv4 address written as ::ffff:10.0.0.1 is the same
	// address as 10.0.0.1 and has to be looked up in the same slice.
	addr = addr.Unmap()

	ranges := s.v6
	if addr.Is4() {
		ranges = s.v4
	}
	if len(ranges) == 0 {
		return false
	}

	key := addr.As16()

	// The ranges are sorted by lo and do not overlap, so the only candidate is
	// the last one starting at or before the key.
	i, found := slices.BinarySearchFunc(ranges, key, func(r addrRange, key [16]byte) int {
		return bytes.Compare(r.lo[:], key[:])
	})
	if !found {
		if i == 0 {
			return false
		}
		i--
	}

	return bytes.Compare(ranges[i].hi[:], key[:]) >= 0
}

// Len is the number of ranges held, after merging. Worth logging at startup:
// it is far below the number of lines parsed, and a zero here means a list
// that was configured but read as empty.
func (s *Set) Len() int {
	if s == nil {
		return 0
	}
	return len(s.v4) + len(s.v6)
}

func rangeOf(prefix netip.Prefix) addrRange {
	// Masked() clears the host bits, so lo is the network address even when the
	// list writes a prefix as 10.0.0.7/24.
	masked := prefix.Masked()
	lo := masked.Addr().Unmap().As16()

	// The prefix was written for its own family but lo is 16 bytes, so a /24 of
	// IPv4 has to become the /120 that covers the same addresses.
	bits := masked.Bits()
	if masked.Addr().Is4() {
		bits += 96
	}

	hi := lo
	for i := bits; i < 128; i++ {
		hi[i/8] |= 1 << (7 - i%8)
	}

	return addrRange{lo: lo, hi: hi}
}

// merge sorts the ranges and folds together the ones that touch or overlap. The
// vpn and datacenter lists share a lot of ground when both are loaded, and each
// list contains prefixes nested inside others; without this the search would
// walk over duplicates and, worse, a nested prefix sorted after its parent
// could hide it from the "last range starting at or before" rule.
func merge(ranges []addrRange) []addrRange {
	if len(ranges) == 0 {
		return nil
	}

	slices.SortFunc(ranges, func(a, b addrRange) int {
		if c := bytes.Compare(a.lo[:], b.lo[:]); c != 0 {
			return c
		}
		return bytes.Compare(a.hi[:], b.hi[:])
	})

	merged := ranges[:1]
	for _, r := range ranges[1:] {
		last := &merged[len(merged)-1]

		// Adjacent counts as overlapping: 1.0.0.0/24 and 1.0.1.0/24 are one
		// range of 512 addresses, and folding them keeps the slice small.
		adjacent := next(last.hi)
		if bytes.Compare(r.lo[:], adjacent[:]) <= 0 {
			if bytes.Compare(r.hi[:], last.hi[:]) > 0 {
				last.hi = r.hi
			}
			continue
		}

		merged = append(merged, r)
	}

	return slices.Clip(merged)
}

// next is addr+1, saturating at the all-ones address. Saturation is what keeps
// the adjacency test above from wrapping around to zero and merging the top of
// the address space with the bottom.
func next(addr [16]byte) [16]byte {
	for i := 15; i >= 0; i-- {
		addr[i]++
		if addr[i] != 0 {
			return addr
		}
	}

	for i := range addr {
		addr[i] = 0xff
	}
	return addr
}
