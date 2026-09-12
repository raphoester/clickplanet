package clicks

import (
	"fmt"

	"connectrpc.com/connect"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/metronome"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/retaker"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/sequencer"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/shadowban"
	"github.com/raphoester/clickplanet.lol-backend/internal/clicks/adapters/primary/http/planetv1controller"
	"github.com/raphoester/clickplanet.lol-backend/internal/kernel/bootstrap"
	"github.com/raphoester/clickplanet.lol-backend/internal/kernel/logging/lf"
	"github.com/raphoester/clickplanet.lol-backend/internal/kernel/xtime"
)

// newAntiBotInterceptor assembles the watchdogs the file asks for, the jury that
// crosses them and the one shadow ban they all pass. It returns nil when
// nothing is enabled, which leaves the click chain exactly as it was.
//
// It lives in the clicks module rather than beside internal/antibot because
// nothing calls antibot: the clicks edge gates on it, the way it gates on the
// session signature.
func newAntiBotInterceptor(
	config AntiBotConfig,
	owner planetv1controller.TileOwner,
	props bootstrap.Props,
) (connect.Interceptor, error) {
	if !config.Enabled {
		return nil, nil
	}

	clock := xtime.ActualProvider{}

	onReaction, onFlag, err := planetv1controller.NewAntiBotReporter(props.Logger, props.Metrics)
	if err != nil {
		return nil, fmt.Errorf("failed to create the antibot reporter: %w", err)
	}

	var (
		watchdogs []antibot.Watchdog
		names     []string
	)

	if config.Retaker.Enabled {
		watchdog := retaker.New(config.Retaker.Detector, clock, onReaction)
		props.Runners.Add(retaker.Name, watchdog.Run)
		watchdogs = append(watchdogs, watchdog)
		names = append(names, retaker.Name)
	}

	if config.Sequencer.Enabled {
		watchdog := sequencer.New(config.Sequencer.Detector, clock)
		props.Runners.Add(sequencer.Name, watchdog.Run)
		watchdogs = append(watchdogs, watchdog)
		names = append(names, sequencer.Name)
	}

	if config.Metronome.Enabled {
		watchdog := metronome.New(config.Metronome.Detector, clock)
		props.Runners.Add(metronome.Name, watchdog.Run)
		watchdogs = append(watchdogs, watchdog)
		names = append(names, metronome.Name)
	}

	if len(watchdogs) == 0 {
		return nil, fmt.Errorf("antiBot is enabled with no watchdog turned on")
	}

	banner := shadowban.New(config.ShadowBan, clock)
	props.Runners.Add("shadow-ban", banner.Run)

	jury := antibot.NewJury(config.Jury, banner, clock, onFlag, watchdogs...)
	props.Runners.Add("jury", jury.Run)

	props.Logger.Info("antibot enabled",
		lf.Any("watchdogs", names),
		lf.Int("minSuspects", config.Jury.MinSuspects),
		lf.Bool("enforce", banner.Enforcing()),
	)

	interceptor, err := planetv1controller.NewAntiBotInterceptor(jury, owner, clock, props.Metrics)
	if err != nil {
		return nil, fmt.Errorf("failed to create the antibot interceptor: %w", err)
	}

	return interceptor, nil
}
