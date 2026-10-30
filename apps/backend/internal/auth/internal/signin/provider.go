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
	Google     = "google"
	Discord    = "discord"
	Email      = "email"
	CrazyGames = "crazygames"
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

// A TokenProvider signed the player in on its own page, and hands the game a token that says who: no redirect.
type TokenProvider interface {
	Claim(ctx context.Context, token string) (*accounts.Claim, error)
}

type TokenProviders map[string]TokenProvider

func (p TokenProviders) Provider(name string) (TokenProvider, error) {
	if len(p) == 0 {
		return nil, ErrSignInOff
	}
	provider, found := p[name]
	if !found {
		return nil, fmt.Errorf("%w: %q", ErrUnknownProvider, name)
	}
	return provider, nil
}

type Offer struct {
	providers Providers
	tokens    TokenProviders
	email     bool
}

func NewOffer(providers Providers, tokens TokenProviders, email bool) Offer {
	return Offer{providers: providers, tokens: tokens, email: email}
}

func (o Offer) Names() []string {
	names := slices.Concat(o.providers.Names(), slices.Collect(maps.Keys(o.tokens)))
	slices.Sort(names)
	if o.email {
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
