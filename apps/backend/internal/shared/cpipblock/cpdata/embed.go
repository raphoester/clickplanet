// Vendored from https://github.com/X4BNet/lists_vpn (MIT).
// Refresh with `make vpn-lists` and commit the result.
package cpdata

import _ "embed"

//go:embed vpn_ipv4.txt
var VPNv4 []byte

//go:embed vpn_ipv6.txt
var VPNv6 []byte

// Kept by hand, not vendored: `make vpn-lists` leaves it alone.
//
//go:embed vpn_extra.txt
var VPNExtra []byte

//go:embed datacenter_ipv4.txt
var DatacenterV4 []byte

//go:embed datacenter_ipv6.txt
var DatacenterV6 []byte
