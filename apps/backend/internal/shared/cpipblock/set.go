package cpipblock

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net/netip"
	"os"
	"slices"
	"strings"
)

// v4 and v6 apart: ::/16 contains ::ffff:0:0/96, so it would match every IPv4 address.
type Set struct {
	v4, v6 []addrRange
}

type addrRange struct {
	lo, hi [16]byte
}

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

func ParseFile(path string) (*Set, error) {
	file, err := os.Open(path) //nolint:gosec // the file the operator of the tool named.
	if err != nil {
		return nil, fmt.Errorf("failed to open %s: %w", path, err)
	}
	defer func() { _ = file.Close() }()

	return Parse(file)
}

func (s *Set) Difference(other *Set) *Set {
	return &Set{v4: difference(s.v4, other.v4), v6: difference(s.v6, other.v6)}
}

func (s *Set) Prefixes() []netip.Prefix {
	var prefixes []netip.Prefix
	for _, r := range s.v4 {
		for _, p := range r.prefixes() {
			prefixes = append(prefixes, netip.PrefixFrom(p.Addr().Unmap(), p.Bits()-96))
		}
	}
	for _, r := range s.v6 {
		prefixes = append(prefixes, r.prefixes()...)
	}
	return prefixes
}

func (s *Set) Contains(ip string) bool {
	if s == nil {
		return false
	}

	addr, err := netip.ParseAddr(ip)
	if err != nil {
		return false
	}

	addr = addr.Unmap()

	ranges := s.v6
	if addr.Is4() {
		ranges = s.v4
	}
	if len(ranges) == 0 {
		return false
	}

	key := addr.As16()

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

func (s *Set) Len() int {
	if s == nil {
		return 0
	}
	return len(s.v4) + len(s.v6)
}

func rangeOf(prefix netip.Prefix) addrRange {
	masked := prefix.Masked()
	lo := masked.Addr().Unmap().As16()

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

func difference(keep, remove []addrRange) []addrRange {
	var kept []addrRange
	j := 0
	for _, r := range keep {
		for j < len(remove) && bytes.Compare(remove[j].hi[:], r.lo[:]) < 0 {
			j++
		}

		lo, removedToTheEnd := r.lo, false
		for k := j; k < len(remove) && bytes.Compare(remove[k].lo[:], r.hi[:]) <= 0; k++ {
			if bytes.Compare(remove[k].lo[:], lo[:]) > 0 {
				kept = append(kept, addrRange{lo: lo, hi: previous(remove[k].lo)})
			}
			if bytes.Compare(remove[k].hi[:], r.hi[:]) >= 0 {
				removedToTheEnd = true
				break
			}
			lo = next(remove[k].hi)
		}

		if !removedToTheEnd {
			kept = append(kept, addrRange{lo: lo, hi: r.hi})
		}
	}
	return kept
}

func (r addrRange) prefixes() []netip.Prefix {
	var prefixes []netip.Prefix
	for lo := r.lo; ; {
		bits := 128
		for bits > 0 {
			wider := rangeOf(netip.PrefixFrom(netip.AddrFrom16(lo), bits-1))
			if wider.lo != lo || bytes.Compare(wider.hi[:], r.hi[:]) > 0 {
				break
			}
			bits--
		}

		block := netip.PrefixFrom(netip.AddrFrom16(lo), bits)
		prefixes = append(prefixes, block)

		hi := rangeOf(block).hi
		if hi == r.hi {
			return prefixes
		}
		lo = next(hi)
	}
}

func previous(addr [16]byte) [16]byte {
	for i := 15; i >= 0; i-- {
		addr[i]--
		if addr[i] != 0xff {
			return addr
		}
	}
	return addr
}

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
