package ipblock

import (
	"bytes"
	"fmt"
	"io"

	"github.com/raphoester/clickplanet.lol-backend/internal/kernel/ipblock/data"
)

type Config struct {
	// Enabled off parses nothing and allocates nothing.
	Enabled bool

	// IncludeDatacenters also refuses hosting and cloud ranges, which catches a
	// self-hosted VPN on a VPS. It also refuses Apple iCloud Private Relay and
	// Cloudflare WARP, which egress from those same ranges and are on by
	// default for a lot of ordinary traffic, so it is off unless asked for.
	IncludeDatacenters bool

	// Allow is never refused, whatever the lists say.
	Allow []string
}

// List names the vendored list an address matched, which is the only useful
// label on the refusal metric: it is how you see what turning
// IncludeDatacenters on would cost before turning it on.
type List string

const (
	ListVPN        List = "vpn"
	ListDatacenter List = "datacenter"
	ListDeny       List = "deny"
)

// Blocklist is the allowlist and the vendored lists together. Immutable once
// built, so the handler goroutines that share one need no locking.
type Blocklist struct {
	allow *Set
	sets  []namedSet
}

type namedSet struct {
	name List
	set  *Set
}

// New builds the blocklist from the vendored lists. A disabled config returns a
// nil *Blocklist, which answers every lookup false — there is no separate
// no-op implementation to keep in step.
func New(config Config) (*Blocklist, error) {
	if !config.Enabled {
		return nil, nil
	}

	allow, err := Parse(readerOf(config.Allow))
	if err != nil {
		return nil, fmt.Errorf("failed to parse allowlist: %w", err)
	}

	vpn, err := Parse(bytes.NewReader(data.VPNv4), bytes.NewReader(data.VPNv6))
	if err != nil {
		return nil, fmt.Errorf("failed to parse vpn list: %w", err)
	}

	sets := []namedSet{{name: ListVPN, set: vpn}}

	if config.IncludeDatacenters {
		datacenter, err := Parse(bytes.NewReader(data.DatacenterV4), bytes.NewReader(data.DatacenterV6))
		if err != nil {
			return nil, fmt.Errorf("failed to parse datacenter list: %w", err)
		}

		sets = append(sets, namedSet{name: ListDatacenter, set: datacenter})
	}

	return &Blocklist{allow: allow, sets: sets}, nil
}

// Blocked reports whether ip should be refused, and which list said so. The
// allowlist wins: it is the escape hatch for an address the vendored lists get
// wrong, so nothing may override it.
func (b *Blocklist) Blocked(ip string) (List, bool) {
	if b == nil {
		return "", false
	}

	if b.allow.Contains(ip) {
		return "", false
	}

	for _, s := range b.sets {
		if s.set.Contains(ip) {
			return s.name, true
		}
	}

	return "", false
}

// Sizes is the merged range count per list, for the startup log line.
func (b *Blocklist) Sizes() map[List]int {
	if b == nil {
		return nil
	}

	sizes := make(map[List]int, len(b.sets))
	for _, s := range b.sets {
		sizes[s.name] = s.set.Len()
	}
	return sizes
}

// NewDenyList builds a blocklist from operator-supplied prefixes alone, with no
// vendored lists behind it — for a list that is maintained by hand in config
// rather than refreshed from upstream.
//
// It is the same Set underneath as the vendored lists, so an entry is a prefix
// and covers a range: blocking a /24 is one line rather than 256. An empty list
// returns a nil *Blocklist, which refuses nothing.
func NewDenyList(prefixes []string) (*Blocklist, error) {
	if len(prefixes) == 0 {
		return nil, nil
	}

	deny, err := Parse(readerOf(prefixes))
	if err != nil {
		return nil, fmt.Errorf("failed to parse deny list: %w", err)
	}

	if deny.Len() == 0 {
		return nil, nil
	}

	return &Blocklist{sets: []namedSet{{name: ListDeny, set: deny}}}, nil
}

func readerOf(prefixes []string) io.Reader {
	var buf bytes.Buffer
	for _, p := range prefixes {
		buf.WriteString(p)
		buf.WriteByte('\n')
	}
	return &buf
}
