//go:build testing

package auth

import (
	"context"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/signin"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpbootstrap"
)

type FakeProvider = signin.FakeProvider

type Claim = accounts.Claim

type FakeProviders struct {
	Google  *FakeProvider
	Discord *FakeProvider
}

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
