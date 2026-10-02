// Package embedded_disposable_domains is the disposable-email-domains blocklist (CC0),
// vendored and embedded. Refresh it with `make disposable-domains` and commit the result.
package embedded_disposable_domains

import (
	"bufio"
	"bytes"
	_ "embed"
	"fmt"
	"strings"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/signin"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcolls"
)

//go:embed disposable_email_blocklist.conf
var vendored []byte

// minDomains is far below the ~9,000 the list holds: fewer is a truncated download.
const minDomains = 5000

type Blocklist struct {
	domains *cpcolls.Set[string]
}

var _ signin.Blocklist = (*Blocklist)(nil)

func New() *Blocklist {
	return &Blocklist{domains: cpcolls.NewSet[string]()}
}

func (b *Blocklist) Load() error {
	scanner := bufio.NewScanner(bytes.NewReader(vendored))
	for scanner.Scan() {
		domain := strings.ToLower(strings.TrimSpace(scanner.Text()))
		if domain == "" || strings.HasPrefix(domain, "#") {
			continue
		}
		b.domains.Add(domain)
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("failed to read the disposable domains: %w", err)
	}
	if b.domains.Len() < minDomains {
		return fmt.Errorf("the disposable domains list holds %d domains, under %d: refresh it", b.domains.Len(), minDomains)
	}
	return nil
}

// Disposable checks the parents too: a subdomain of a disposable service hands out addresses as freely.
func (b *Blocklist) Disposable(domain string) bool {
	for name := strings.ToLower(domain); name != ""; {
		if b.domains.Contains(name) {
			return true
		}
		_, parent, found := strings.Cut(name, ".")
		if !found {
			return false
		}
		name = parent
	}
	return false
}

func (b *Blocklist) Size() int {
	return b.domains.Len()
}
