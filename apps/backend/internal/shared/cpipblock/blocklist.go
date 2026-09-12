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
	allow *Set
	sets  []namedSet
}

type namedSet struct {
	name List
	set  *Set
}

func New(config Config) (*Blocklist, error) {
	if !config.Enabled {
		return nil, nil
	}

	allow, err := Parse(readerOf(config.Allow))
	if err != nil {
		return nil, fmt.Errorf("failed to parse allowlist: %w", err)
	}

	vpn, err := Parse(bytes.NewReader(cpdata.VPNv4), bytes.NewReader(cpdata.VPNv6))
	if err != nil {
		return nil, fmt.Errorf("failed to parse vpn list: %w", err)
	}

	sets := []namedSet{{name: ListVPN, set: vpn}}

	if config.IncludeDatacenters {
		datacenter, err := Parse(bytes.NewReader(cpdata.DatacenterV4), bytes.NewReader(cpdata.DatacenterV6))
		if err != nil {
			return nil, fmt.Errorf("failed to parse datacenter list: %w", err)
		}

		sets = append(sets, namedSet{name: ListDatacenter, set: datacenter})
	}

	return &Blocklist{allow: allow, sets: sets}, nil
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
