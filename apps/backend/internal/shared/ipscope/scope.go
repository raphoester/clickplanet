// Package ipscope collapses a source address to the unit a budget is charged
// to.
//
// A per-address budget assumes an address costs something to come by. That
// holds for IPv4, where a residential line has one. It does not hold for IPv6:
// the smallest allocation a subscriber is handed is a /64, and most are handed
// far more, so a budget keyed on a full v6 address is one the same line walks
// out of by picking the next address in a prefix it already owns. One home
// connection is then thousands of independent callers, each with its own
// throttle and its own session, at no cost.
//
// Only budgets are charged to the prefix. Blocking stays on the full address:
// the VPN and datacenter lists are precise prefixes of their own, and widening
// a hit to the surrounding /64 would refuse neighbours who are not on them.
package ipscope

import "net/netip"

// PrefixBits is the v6 prefix a budget is charged to. A /64 is the smallest
// allocation a subscriber ever receives, so charging to it never merges two
// subscribers who were handed separate prefixes — while a single line cannot
// buy itself a second budget out of the addresses inside its own.
//
// Larger allocations (a /56, a /48) do get one budget per /64 they contain.
// Widening this to cover them costs more than it buys: those prefixes are also
// handed to whole buildings and campuses, and merging those into one bucket
// throttles a crowd for what one of them does.
const PrefixBits = 64

// Of returns the key ip is charged under: the address itself for IPv4, and the
// /64 containing it for IPv6.
//
// Anything that does not parse is returned unchanged. It is not an address and
// cannot be reasoned about as one, so it keeps whatever bucket it names rather
// than being folded in with every other unparseable value.
func Of(ip string) string {
	addr, err := netip.ParseAddr(ip)
	if err != nil {
		return ip
	}

	// Unmap first: ::ffff:192.0.2.7 is an IPv4 address wearing a v6 shape, and
	// charging it to a /64 would hand every v4 address a shared budget.
	addr = addr.Unmap()
	if addr.Is4() {
		return addr.String()
	}

	prefix, err := addr.Prefix(PrefixBits)
	if err != nil {
		return ip
	}

	// Prefix drops the zone, so fe80::1%eth0 and fe80::2%eth1 collapse together.
	// Nothing routable carries one, and a link-local reaching this is already
	// not the address the budget was meant to price.
	return prefix.String()
}
