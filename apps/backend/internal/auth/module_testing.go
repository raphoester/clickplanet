//go:build testing

package auth

import (
	"context"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/signin"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpbootstrap"
)

// FakeProvider plays a provider in a test that boots the module: Grant a code, then complete the sign-in with it.
type FakeProvider = signin.FakeProvider

// Claim is who a granted code signs in.
type Claim = accounts.Claim

// FakeProviders are the providers NewModuleWithFakeProviders signs in with.
type FakeProviders struct {
	Google  *FakeProvider
	Discord *FakeProvider
}

// NewModuleWithFakeProviders is NewModule with sign-in on and no network: every provider is a fake.
func NewModuleWithFakeProviders(config Config) (cpbootstrap.Module, FakeProviders) {
	fakes := FakeProviders{Google: signin.NewFakeProvider(signin.Google), Discord: signin.NewFakeProvider(signin.Discord)}
	providers := signin.Providers{signin.Google: fakes.Google, signin.Discord: fakes.Discord}

	return cpbootstrap.Module{
		Name:    moduleName,
		Enabled: config.Enabled,
		DiSequence: func(ctx context.Context, props cpbootstrap.Props) error {
			return build(ctx, config.withDefaults(), props, providers)
		},
	}, fakes
}
