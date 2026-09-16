package signin

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts"
)

const (
	Google  = "google"
	Discord = "discord"
)

var (
	ErrSignInOff       = errors.New("sign-in is off on this server")
	ErrUnknownProvider = errors.New("this provider is not offered")
	ErrFlowInvalid     = errors.New("no sign-in of this browser matches")
	ErrProviderRefused = errors.New("the provider refused the sign-in")
)

// Provider is one place a player signs in.
type Provider interface {
	AuthorizationURL(flow *Flow) string
	// Exchange trades the callback's code for the user it signed in. A code or an answer the provider refused is ErrProviderRefused.
	Exchange(ctx context.Context, code string, flow *Flow) (*accounts.Claim, error)
}

// Providers is every provider this server offers, by name. Empty is sign-in off.
type Providers map[string]Provider

// Names is every provider offered, in order.
func (p Providers) Names() []string {
	return slices.Sorted(maps.Keys(p))
}

func (p Providers) Off() bool {
	return len(p) == 0
}

func (p Providers) Get(name string) (Provider, error) {
	if p.Off() {
		return nil, ErrSignInOff
	}
	provider, found := p[name]
	if !found {
		return nil, fmt.Errorf("%w: %q", ErrUnknownProvider, name)
	}
	return provider, nil
}

// Client is one provider's block in the file: the id is public, the secret comes from the environment.
type Client struct {
	ClientID     string
	ClientSecret string
}

// Configured is a provider this server offers: a client id is set.
func (c Client) Configured() bool {
	return c.ClientID != ""
}
