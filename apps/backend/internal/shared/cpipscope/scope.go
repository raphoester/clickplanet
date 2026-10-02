package cpipscope

import "net/netip"

const PrefixBits = 64

func Of(ip string) string {
	addr, err := netip.ParseAddr(ip)
	if err != nil {
		return ip
	}

	// Unmap first, or every v4-mapped address would share one /64 budget.
	addr = addr.Unmap()
	if addr.Is4() {
		return addr.String()
	}

	prefix, err := addr.Prefix(PrefixBits)
	if err != nil {
		return ip
	}

	return prefix.String()
}

func Parse(text string) (string, bool) {
	if addr, err := netip.ParseAddr(text); err == nil {
		return Of(addr.String()), true
	}

	prefix, err := netip.ParsePrefix(text)
	if err != nil || Of(prefix.Addr().String()) != prefix.String() {
		return "", false
	}

	return prefix.String(), true
}
