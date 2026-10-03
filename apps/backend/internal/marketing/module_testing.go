//go:build testing

package marketing

import (
	"context"

	"github.com/raphoester/clickplanet.lol-backend/internal/marketing/internal/subscriptions"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpbootstrap"
)

type FakeAudience = subscriptions.FakeAudience

type Call = subscriptions.Call

func NewModuleWithFakeAudience(config Config) (cpbootstrap.Module, *FakeAudience) {
	audience := &FakeAudience{}

	return cpbootstrap.Module{
		Name:    moduleName,
		Enabled: config.Enabled,
		DiSequence: func(ctx context.Context, props cpbootstrap.Props) error {
			return build(ctx, config.withDefaults(), props, audience)
		},
	}, audience
}
