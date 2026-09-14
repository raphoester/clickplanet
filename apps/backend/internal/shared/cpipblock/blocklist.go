package cpipblock

import (
	"bytes"
	"fmt"
	"io"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpipblock/cpdata"
)

type Config struct {
	Enabled bool

	IncludeDatacenters bool

	Allow []string
}

type List string

const (
	ListVPN        List = "vpn"
	ListDatacenter List = "datacenter"
	ListDeny       List = "deny"
)

type Blocklist struct {
	config Config
	allow  *Set
	sets   []namedSet
}

type namedSet struct {
	name List
	set  *Set
}

// New returns nil when the config is off: a nil Blocklist loads nothing and blocks nothing.
func New(config Config) *Blocklist {
	if !config.Enabled {
		return nil
	}

	return &Blocklist{config: config}
}

// Load parses the allowlist and the vendored lists the config asks for.
func (b *Blocklist) Load() error {
	if b == nil {
		return nil
	}

	allow, err := Parse(readerOf(b.config.Allow))
	if err != nil {
		return fmt.Errorf("failed to parse allowlist: %w", err)
	}

	vpn, err := Parse(
		bytes.NewReader(cpdata.VPNv4),
		bytes.NewReader(cpdata.VPNv6),
		bytes.NewReader(cpdata.VPNExtra),
	)
	if err != nil {
		return fmt.Errorf("failed to parse vpn list: %w", err)
	}

	sets := []namedSet{{name: ListVPN, set: vpn}}

	if b.config.IncludeDatacenters {
		datacenter, err := Parse(bytes.NewReader(cpdata.DatacenterV4), bytes.NewReader(cpdata.DatacenterV6))
		if err != nil {
			return fmt.Errorf("failed to parse datacenter list: %w", err)
		}

		sets = append(sets, namedSet{name: ListDatacenter, set: datacenter})
	}

	b.allow, b.sets = allow, sets

	return nil
}

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

func NewDenyList(prefixes []string) (*Blocklist, error) {
	if len(prefixes) == 0 {
		//nolint:nilnil // no prefixes is "no blocking configured", not a failure.
		return nil, nil
	}

	deny, err := Parse(readerOf(prefixes))
	if err != nil {
		return nil, fmt.Errorf("failed to parse deny list: %w", err)
	}

	if deny.Len() == 0 {
		//nolint:nilnil // an empty deny list is "no blocking configured".
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
