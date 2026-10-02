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
	Email   = "email"
)

var (
	ErrSignInOff       = errors.New("sign-in is off on this server")
	ErrUnknownProvider = errors.New("this provider is not offered")
	ErrFlowInvalid     = errors.New("no sign-in of this browser matches")
	ErrProviderRefused = errors.New("the provider refused the sign-in")
)

type Provider interface {
	AuthorizationURL(flow *Flow) string
	Exchange(ctx context.Context, code string, flow *Flow) (*accounts.Claim, error)
}

type Providers map[string]Provider

func (p Providers) Names() []string {
	return slices.Sorted(maps.Keys(p))
}

func (p Providers) Off() bool {
	return len(p) == 0
}

func (p Providers) Provider(name string) (Provider, error) {
	if p.Off() {
		return nil, ErrSignInOff
	}
	provider, found := p[name]
	if !found {
		return nil, fmt.Errorf("%w: %q", ErrUnknownProvider, name)
	}
	return provider, nil
}

type Offer struct {
	Providers Providers
	Email     bool
}

func (o Offer) Names() []string {
	names := o.Providers.Names()
	if o.Email {
		names = append(names, Email)
	}
	return names
}

type Client struct {
	ClientID     string
	ClientSecret string
}

func (c Client) Configured() bool {
	return c.ClientID != ""
}
