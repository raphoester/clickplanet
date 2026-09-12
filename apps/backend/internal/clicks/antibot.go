package clicks

import (
	"fmt"

	"connectrpc.com/connect"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot"
	"github.com/raphoester/clickplanet.lol-backend/internal/clicks/internal/adapters/primary/http/planetv1controller"
	"github.com/raphoester/clickplanet.lol-backend/internal/kernel/bootstrap"
	"github.com/raphoester/clickplanet.lol-backend/internal/kernel/logging/lf"
	"github.com/raphoester/clickplanet.lol-backend/internal/kernel/xtime"
)

// newAntiBotInterceptor wraps the click edge in an antibot guard, nil when off.
func newAntiBotInterceptor(
	config antibot.Config,
	owner planetv1controller.TileOwner,
	props bootstrap.Props,
) (connect.Interceptor, error) {
	clock := xtime.ActualProvider{}

	observer, err := planetv1controller.NewAntiBotObserver(props.Logger, props.Metrics)
	if err != nil {
		return nil, fmt.Errorf("failed to create the antibot observer: %w", err)
	}

	guard, err := antibot.New(config, clock, observer)
	if err != nil {
		return nil, fmt.Errorf("failed to build the antibot guard: %w", err)
	}
	if guard == nil {
		return nil, nil
	}

	props.Runners.Add("antibot", guard.Run)

	described := guard.Describe()
	props.Logger.Info("antibot enabled",
		lf.Any("watchdogs", described.Watchdogs),
		lf.Int("minSuspects", described.MinSuspects),
		lf.Bool("enforce", described.Enforcing),
	)

	interceptor, err := planetv1controller.NewAntiBotInterceptor(guard, owner, clock, props.Metrics)
	if err != nil {
		return nil, fmt.Errorf("failed to create the antibot interceptor: %w", err)
	}

	return interceptor, nil
}
