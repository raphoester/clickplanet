//go:build testing

package signin

import (
	"context"
	"fmt"
	"slices"
	"sync"
)

type SequentialCodes struct {
	mu   sync.Mutex
	next int
}

func (c *SequentialCodes) NewCode() (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.next++
	return fmt.Sprintf("%06d", c.next), nil
}

type Sent struct {
	To     Address
	Letter Letter
}

type FakeMailer struct {
	mu   sync.Mutex
	sent []Sent
	err  error
}

var _ Mailer = (*FakeMailer)(nil)

func (m *FakeMailer) Send(_ context.Context, to Address, letter Letter) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.err != nil {
		return m.err
	}
	m.sent = append(m.sent, Sent{To: to, Letter: letter})
	return nil
}

func (m *FakeMailer) FailWith(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.err = err
}

func (m *FakeMailer) Sent() []Sent {
	m.mu.Lock()
	defer m.mu.Unlock()

	return slices.Clone(m.sent)
}

type BlockedDomains []string

func (b BlockedDomains) Disposable(domain string) bool {
	return slices.Contains(b, domain)
}
