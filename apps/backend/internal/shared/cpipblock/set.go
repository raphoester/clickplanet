package cpipblock

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net/netip"
	"slices"
	"strings"
)

// v4 and v6 cannot share one slice: the v4-mapped block ::ffff:0:0/96 sits
// inside short v6 prefixes, so ::/16 would swallow every IPv4 address.
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
