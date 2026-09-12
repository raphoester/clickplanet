package planet

import (
	"fmt"
	"log/slog"

	"connectrpc.com/connect"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/adapters/primary/http/planetv1controller"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpbootstrap"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

// newAntiBotInterceptor wraps the click edge in an antibot guard, nil when off.
func newAntiBotInterceptor(
	config antibot.Config,
	owner planetv1controller.TileOwner,
	props cpbootstrap.Props,
) (connect.Interceptor, error) {
	clock := cptime.SystemClock{}

	observer, err := planetv1controller.NewAntiBotObserver(props.Logger, props.Metrics)
	if err != nil {
		return nil, fmt.Errorf("failed to create the antibot observer: %w", err)
	}

	guard, err := antibot.New(config, clock, observer)
	if err != nil {
		return nil, fmt.Errorf("failed to build the antibot guard: %w", err)
	}
	if guard == nil {
		//nolint:nilnil // antibot.New returns a nil Guard when the block is off.
		return nil, nil
	}

	props.Runners.Add("antibot", guard.Run)

	described := guard.Describe()
	props.Logger.Info("antibot enabled",
		slog.Any("watchdogs", described.Watchdogs),
		slog.Int("minSuspects", described.MinSuspects),
		slog.Bool("enforce", described.Enforcing),
	)

	interceptor, err := planetv1controller.NewAntiBotInterceptor(guard, owner, clock, props.Metrics)
	if err != nil {
		return nil, fmt.Errorf("failed to create the antibot interceptor: %w", err)
	}

	return interceptor, nil
}
