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

func ClaimOf(subject string, email string, emailVerified bool) Claim {
	return accounts.ClaimOf(subject, email, emailVerified)
}

type FakeMailer = signin.FakeMailer

type FakeProviders struct {
	Google  *FakeProvider
	Discord *FakeProvider
	Mailer  *FakeMailer

	AdminGoogle *FakeProvider
}

func NewModuleWithFakeProviders(config Config) (cpbootstrap.Module, FakeProviders) {
	fakes := FakeProviders{
		Google: signin.NewFakeProvider(signin.Google), Discord: signin.NewFakeProvider(signin.Discord), Mailer: &FakeMailer{},
		AdminGoogle: signin.NewFakeProvider(signin.Google),
	}
	providers := signin.Providers{signin.Google: fakes.Google, signin.Discord: fakes.Discord}

	return cpbootstrap.Module{
		Name:    moduleName,
		Enabled: config.Enabled,
		DiSequence: func(ctx context.Context, props cpbootstrap.Props) error {
			config := config.withDefaults()
			config.Email.Enabled = true
			return build(ctx, config, props, signIns{
				game:  providers,
				admin: signin.Providers{signin.Google: fakes.AdminGoogle},
				mail:  mailing{mailer: fakes.Mailer, codes: &signin.SequentialCodes{}},
			})
		},
	}, fakes
}
