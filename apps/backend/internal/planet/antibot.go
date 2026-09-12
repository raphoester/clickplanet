package planet

import (
	"fmt"

	"connectrpc.com/connect"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/adapters/primary/http/planetv1controller"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpbootstrap"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cplogging/cplf"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

// newAntiBotInterceptor wraps the click edge in an antibot guard, nil when off.
func newAntiBotInterceptor(
	config antibot.Config,
	owner planetv1controller.TileOwner,
	props cpbootstrap.Props,
) (connect.Interceptor, error) {
	clock := cptime.ActualProvider{}

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
		cplf.Any("watchdogs", described.Watchdogs),
		cplf.Int("minSuspects", described.MinSuspects),
		cplf.Bool("enforce", described.Enforcing),
	)

	interceptor, err := planetv1controller.NewAntiBotInterceptor(guard, owner, clock, props.Metrics)
	if err != nil {
		return nil, fmt.Errorf("failed to create the antibot interceptor: %w", err)
	}

	return interceptor, nil
}
