// Package data holds the vendored VPN and datacenter prefix lists.
//
// The lists come from https://github.com/X4BNet/lists_vpn (MIT), which derives
// them from ASN ownership and rebuilds them daily. They are checked in and
// embedded rather than fetched at boot: cmd/api is a self-contained container
// with no startup dependencies, and a boot that can fail because GitHub is down
// is a worse trade than a list that ages between deploys. The Cloudflare ranges
// in deploy/vps/Caddyfile are maintained the same way.
//
// Refresh them with `make vpn-lists` and commit the result.
package data

import _ "embed"

//go:embed vpn_ipv4.txt
var VPNv4 []byte

//go:embed vpn_ipv6.txt
var VPNv6 []byte

//go:embed datacenter_ipv4.txt
var DatacenterV4 []byte

//go:embed datacenter_ipv6.txt
var DatacenterV6 []byte
