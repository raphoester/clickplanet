# VPN blocklist

`NewVPNBlockInterceptor` refuses **`Click` only**, with `CodePermissionDenied` (HTTP 403), when the source address falls in a vendored VPN range. It sits **outside the rate limiter** in the interceptor chain, deliberately: a refused address must not also spend a token, or the next click would come back 429 and the web app would show the throttle dialog instead of the VPN one.

**It exists because of the rate limiter, not instead of it.** The scope's bucket is keyed on an address, and a commercial VPN is the cheapest way to get a fresh one; refusing those addresses is what makes the bucket hold. It raises the floor rather than closing the door — residential proxies appear in no public list, and nothing here stops one. **That gap is what [Sessions](sessions.md) closes**, by requiring something an address cannot buy; chasing list completeness instead is a treadmill.

Reads and the streams are untouched. A VPN user still loads the planet and follows it live; they cannot paint. That is also what keeps a false positive readable: the page works and says why, instead of failing to load.

**The ranges are vendored and embedded**, from [X4BNet/lists_vpn](https://github.com/X4BNet/lists_vpn) (MIT, rebuilt daily from ASN ownership), in `internal/shared/cpipblock/cpdata`. Not fetched at boot: `cmd/api` is a self-contained container with no startup dependencies, and a boot that can fail because GitHub is down is a worse trade than a list that ages between deploys — the Cloudflare ranges in `deploy/vps/caddy/Caddyfile` are maintained the same way. Refresh with `make vpn-lists` and commit; the tests assert the lists still parse and are not truncated.

**X4BNet works from ASN ownership, so the `vpn` list folds in four more sources**, all fetched by the same target (it needs `jq`):

- `vpn_providers.txt` — [Joe12387/open-source-vpn-ip-lists](https://github.com/Joe12387/open-source-vpn-ip-lists) (CC0): each provider's own server list, which catches servers rented inside networks X4BNet does not attribute to a VPN.
- `vpn_az0.txt` — [az0/vpn_ip](https://github.com/az0/vpn_ip) (GPL-3; IP addresses are facts): hostnames pulled from VPN apps, APIs and browser extensions (Hola, CyberGhost, …) and resolved.
- `tor_exits.txt` — the Tor Project's exit list.
- `vpn_netnames.txt` — every range registered under a name in `VPN_NETNAMES`, from the RIPE and ARIN RDAP servers. **This is how Firefox's built-in VPN is caught**: its egress (`MOZILLA-FIREFOX-VPN`) is Mozilla's address space announced by Fastly, so no ASN list can find it, and it hands each browser session its own /64 — one client walks the range and gets a fresh throttle bucket and ban scope each time.

Measured on 2026-09-14 against the X4BNet vpn and datacenter lists together: the provider lists add ~1,250 addresses, az0 ~1,800 beyond those, Tor ~750. iCloud Private Relay's published egress list is left out on purpose, for the reason the datacenter list is off by default.

`cpipblock` holds them as sorted, merged `[lo, hi]` ranges of 16-byte addresses and binary-searches them — ~63k prefixes fold to far fewer ranges, about 2 MB resident and well under 100 ns per lookup. **IPv4 and IPv6 live in separate slices.** They cannot share one: an IPv4 address in its v4-mapped form sits inside `::ffff:0:0/96`, so a single ordering would let a v6 prefix as short as `::/16` silently swallow every IPv4 address on the internet.

`vpnBlocklist.includeDatacenters` adds the much broader hosting list, which catches a self-hosted VPN on a VPS. It is off by default because it also refuses Apple iCloud Private Relay and Cloudflare WARP — both egress from datacenter ranges, both on by default for a lot of ordinary mobile traffic. `blocked_clicks{list}` is labelled per list precisely so you can see what turning it on would cost before turning it on. `vpnBlocklist.allow` is the escape hatch and beats both lists.

**The `X-Real-IP` caveat applies here too, and matters more.** Anything that reaches `backend:8080` directly bypasses the blocklist by sending its own header, exactly as it bypasses the throttle. Caddy replaces the header on every backend route, so this is only reachable if the backend port is exposed — but the bypass is now security-relevant rather than merely an abuse nuisance.
