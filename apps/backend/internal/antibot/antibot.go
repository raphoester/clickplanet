// Package antibot holds what is left after the address blocklist and the session
// mint: a caller who solved Turnstile in a real browser and then pointed a script
// at the API. Nothing about the address or the token separates them from a
// player, so everything here reads behaviour instead.
//
// This file is the whole of what a caller may use. The watchdogs, the jury that
// crosses what they say and the ban they all pass live under internal/, so the
// click edge cannot assemble them itself — how a bot is recognised is this
// package's business and changing it is one package's edit.
package antibot

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/detect"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/jury"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/metronome"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/retaker"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/sequencer"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/shadowban"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

// The two types a caller writes down, because they appear in the signatures it
// implements: it builds a Click and it is handed a Report. They are defined
// under internal/detect because the watchdogs share them and cannot import this
// package without a cycle, so these aliases are what publish them and there is
// still one definition of each.
//
// Nothing else is aliased, because nothing else has to be named. A caller ranges
// a Report's opinions and asks each one whether it Fired and what it says for
// itself; the verdict ladder, the rule that tripped and the numbers behind it
// never leave this package as vocabulary the edge has to speak.
type (
	Click    = detect.Click       // one Click RPC, as the guard sees it
	Report   = detect.Report      // one ban, with every watchdog's opinion behind it
	Sentence = shadowban.Sentence // a scope's ban, as the operator tools read it
)

// Config is the `antiBot:` block. A watchdog left out of the file is off, and the
// nested types are this package's or its interior's, so a new bound lands with
// the watchdog that reads it rather than here. None of them is named outside:
// koanf fills them by reflection and a caller still sets their fields, it just
// cannot write the type — the settings are published because they are in the
// file, the code that reads them is not.
type Config struct {
	Enabled bool

	ShadowBan shadowban.Config
	Jury      jury.Config

	Retaker   retakerConfig
	Sequencer sequencerConfig
	Metronome metronomeConfig
}

type retakerConfig struct {
	Enabled  bool
	Detector retaker.Config
}

type sequencerConfig struct {
	Enabled  bool
	Detector sequencer.Config
}

type metronomeConfig struct {
	Enabled  bool
	Detector metronome.Config
}

// Observer is how a finding leaves this package, which measures and judges but
// logs and counts nothing itself. Both hooks are optional.
type Observer struct {
	// Every reaction, not only the ones arguing for a ban: the shape of the whole
	// distribution is what shows the bot band.
	OnReaction func(delay time.Duration)

	OnFlag func(report Report)

	// Bans that could not be restored at boot or saved since.
	OnStateError func(err error)
}

// Guard is the whole surface the click edge gates on.
type Guard interface {
	// Called before the handler runs, because the map stops remembering who held
	// the tile the moment it does.
	Inspect(click Click) (drop bool)

	// Only for a click the handler accepted: a refused one recorded as a take is
	// how the next honest clicker of that tile comes to look like it is reacting.
	Committed(click Click)

	// LoadBans reads the bans saved at the last shutdown, reporting a bad file through OnStateError.
	LoadBans()

	// Flagged is how many callers are currently banned, for the gauge.
	Flagged() int

	// Banned says whether a scope's actions should be dropped: false while enforce is off.
	Banned(scope string) bool

	// Ban is an operator's ban on a scope; a zero duration takes the ladder's. Enforcing says whether it drops anything.
	Ban(scope string, duration time.Duration) Sentence
	Sentence(scope string) (Sentence, bool)
	Enforcing() bool

	// One runner whatever the config turned on: how many sweepers there are is
	// this package's business.
	Name() string
	Run(ctx context.Context)

	Describe() Description
}

// New assembles the watchdogs the config asks for, the jury that crosses them and
// the one ban they all pass. A nil Guard means the block is off, which leaves the
// click chain exactly as it was; enabling it with every watchdog off is an error,
// because that measures nothing while looking like a defence.
func New(config Config, clock cptime.Clock, observer Observer) (Guard, error) {
	if !config.Enabled {
		//nolint:nilnil // a nil Guard is the contract: the caller skips the
		// interceptor entirely. See the doc comment above.
		return nil, nil
	}

	if clock == nil {
		clock = cptime.SystemClock{}
	}

	g := &guard{}

	var (
		watchdogs []detect.Watchdog
		names     []string
	)

	if config.Retaker.Enabled {
		watchdog := retaker.New(config.Retaker.Detector, clock, observer.OnReaction)
		g.runners = append(g.runners, watchdog.Run)
		watchdogs = append(watchdogs, watchdog)
		names = append(names, retaker.Name)
	}

	if config.Sequencer.Enabled {
		watchdog := sequencer.New(config.Sequencer.Detector, clock)
		g.runners = append(g.runners, watchdog.Run)
		watchdogs = append(watchdogs, watchdog)
		names = append(names, sequencer.Name)
	}

	if config.Metronome.Enabled {
		watchdog := metronome.New(config.Metronome.Detector, clock)
		g.runners = append(g.runners, watchdog.Run)
		watchdogs = append(watchdogs, watchdog)
		names = append(names, metronome.Name)
	}

	if len(watchdogs) == 0 {
		return nil, fmt.Errorf("antiBot is enabled with no watchdog turned on")
	}

	// Defaulted here so the description carries the bounds actually enforced.
	juryConfig := config.Jury.WithDefaults()

	banner := shadowban.New(config.ShadowBan, clock, observer.OnStateError)
	g.runners = append(g.runners, banner.Run)

	g.banner = banner
	g.jury = jury.New(juryConfig, banner, clock, observer.OnFlag, watchdogs...)
	g.runners = append(g.runners, g.jury.Run)

	g.description = Description{
		Watchdogs:   names,
		MinSuspects: juryConfig.MinSuspects,
		Enforcing:   banner.Enforcing(),
	}

	return g, nil
}

// Description is what a guard says about itself for the caller's boot line. Not
// the config back: this is what was turned on, after the defaults were applied.
type Description struct {
	Watchdogs   []string
	MinSuspects int
	Enforcing   bool
}

type guard struct {
	jury        *jury.Jury
	banner      *shadowban.Banner
	runners     []func(context.Context)
	description Description
}

func (g *guard) Inspect(click Click) bool { return g.jury.Inspect(click) }

func (g *guard) Committed(click Click) { g.jury.Committed(click) }

func (g *guard) LoadBans() { g.banner.LoadState() }

func (g *guard) Flagged() int { return g.jury.Flagged() }

func (g *guard) Describe() Description { return g.description }

func (g *guard) Ban(scope string, duration time.Duration) Sentence {
	return g.banner.Ban(scope, duration)
}

func (g *guard) Sentence(scope string) (Sentence, bool) { return g.banner.Sentence(scope) }

func (g *guard) Enforcing() bool { return g.banner.Enforcing() }

func (g *guard) Banned(scope string) bool { return g.banner.Banned(scope) }

func (g *guard) Name() string { return "antibot" }

// Run fans out to every sweeper enabled and blocks until they all return.
func (g *guard) Run(ctx context.Context) {
	var wg sync.WaitGroup

	for _, run := range g.runners {
		wg.Add(1)
		go func() {
			defer wg.Done()
			run(ctx)
		}()
	}

	wg.Wait()
}
