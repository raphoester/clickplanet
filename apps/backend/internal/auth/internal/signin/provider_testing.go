//go:build testing

package signin

import (
	"context"
	"fmt"
	"net/url"
	"sync"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts"
)

type FakeProvider struct {
	name string

	mu         sync.Mutex
	grants     map[string]accounts.Claim
	challenges map[string]string
	failWith   error
}

var _ Provider = (*FakeProvider)(nil)

func NewFakeProvider(name string) *FakeProvider {
	return &FakeProvider{name: name, grants: map[string]accounts.Claim{}, challenges: map[string]string{}}
}

func (p *FakeProvider) Grant(code string, claim accounts.Claim) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.grants[code] = claim
}

func (p *FakeProvider) FailWith(err error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.failWith = err
}

func (p *FakeProvider) AuthorizationURL(flow *Flow) string {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.challenges[flow.State()] = flow.Challenge()
	return "https://" + p.name + ".example/authorize?" + url.Values{"state": {flow.State()}}.Encode()
}

func (p *FakeProvider) Exchange(_ context.Context, code string, flow *Flow) (*accounts.Claim, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.failWith != nil {
		return nil, p.failWith
	}
	claim, granted := p.grants[code]
	if !granted {
		return nil, fmt.Errorf("%w: %s knows no code %q", ErrProviderRefused, p.name, code)
	}
	if p.challenges[flow.State()] != flow.Challenge() {
		return nil, fmt.Errorf("%w: the verifier does not match the challenge", ErrProviderRefused)
	}

	delete(p.grants, code)
	return &claim, nil
}

// A token is good as many times as it is shown, as a real one is until it expires.
type FakeTokenProvider struct {
	name string

	mu       sync.Mutex
	tokens   map[string]accounts.Claim
	failWith error
}

var _ TokenProvider = (*FakeTokenProvider)(nil)

func NewFakeTokenProvider(name string) *FakeTokenProvider {
	return &FakeTokenProvider{name: name, tokens: map[string]accounts.Claim{}}
}

func (p *FakeTokenProvider) Sign(token string, claim accounts.Claim) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.tokens[token] = claim
}

func (p *FakeTokenProvider) FailWith(err error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.failWith = err
}

func (p *FakeTokenProvider) Claim(_ context.Context, token string) (*accounts.Claim, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.failWith != nil {
		return nil, p.failWith
	}
	claim, signed := p.tokens[token]
	if !signed {
		return nil, fmt.Errorf("%w: %s signed no token %q", ErrProviderRefused, p.name, token)
	}
	return &claim, nil
}

type SequentialSecrets struct {
	mu   sync.Mutex
	next int
}

func (s *SequentialSecrets) NewSecret() (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.next++
	return fmt.Sprintf("secret-%d", s.next), nil
}
