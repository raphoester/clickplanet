// Vendored from X4BNet/lists_vpn (MIT), Joe12387/open-source-vpn-ip-lists
// (CC0), az0/vpn_ip (GPL-3), the Tor Project's exit list, and RIPE and ARIN RDAP.
// Refresh with `make vpn-lists` and commit the result.
package cpdata

import _ "embed"

//go:embed vpn_ipv4.txt
var VPNv4 []byte

//go:embed vpn_ipv6.txt
var VPNv6 []byte

//go:embed vpn_providers.txt
var VPNProviders []byte

//go:embed vpn_az0.txt
var VPNAz0 []byte

//go:embed tor_exits.txt
var TorExits []byte

//go:embed vpn_netnames.txt
var VPNNetnames []byte

//go:embed datacenter_ipv4.txt
var DatacenterV4 []byte

//go:embed datacenter_ipv6.txt
var DatacenterV6 []byte
