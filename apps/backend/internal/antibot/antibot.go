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

// The vocabulary a caller reads. It is defined under internal/detect because the
// watchdogs share it and cannot import this package without a cycle, so these
// aliases are what publish it and there is still one definition of each.
type (
	Click    = detect.Click    // one Click RPC, as the guard sees it
	Report   = detect.Report   // one ban, with every watchdog's opinion behind it
	Opinion  = detect.Opinion  // one watchdog's standing verdict on a caller
	Verdict  = detect.Verdict  // how sure one watchdog is
	Evidence = detect.Evidence // why a watchdog returned the verdict it did
	Field    = detect.Field    // one number a watchdog wanted in the log line

	// Settings are published even though what reads them is not: they are in the file.
	JuryConfig      = jury.Config
	ShadowBanConfig = shadowban.Config
)

const (
	Clear   = detect.Clear   // looks like anybody else
	Suspect = detect.Suspect // counts only alongside another watchdog
	Certain = detect.Certain // a reading no hand produces; bans on its own
)

// Config is the `antiBot:` block. A watchdog left out of the file is off, and the
// nested types are this package's, so a new bound lands with the watchdog that
// reads it rather than here.
type Config struct {
	Enabled bool

	ShadowBan ShadowBanConfig
	Jury      JuryConfig

	Retaker   RetakerConfig
	Sequencer SequencerConfig
	Metronome MetronomeConfig
}

type RetakerConfig struct {
	Enabled  bool
	Detector retaker.Config
}

type SequencerConfig struct {
	Enabled  bool
	Detector sequencer.Config
}

type MetronomeConfig struct {
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
}

// Guard is the whole surface the click edge gates on.
type Guard interface {
	// Called before the handler runs, because the map stops remembering who held
	// the tile the moment it does.
	Inspect(click Click) (drop bool)

	// Only for a click the handler accepted: a refused one recorded as a take is
	// how the next honest clicker of that tile comes to look like it is reacting.
	Committed(click Click)

	// Flagged is how many callers are currently banned, for the gauge.
	Flagged() int

	// One runner whatever the config turned on: how many sweepers there are is
	// this package's business.
	Run(ctx context.Context)

	Describe() Description
}

// New assembles the watchdogs the config asks for, the jury that crosses them and
// the one ban they all pass. A nil Guard means the block is off, which leaves the
// click chain exactly as it was; enabling it with every watchdog off is an error,
// because that measures nothing while looking like a defence.
func New(config Config, clock cptime.Provider, observer Observer) (Guard, error) {
	if !config.Enabled {
		//nolint:nilnil // a nil Guard is the contract: the caller skips the
		// interceptor entirely. See the doc comment above.
		return nil, nil
	}

	if clock == nil {
		clock = cptime.ActualProvider{}
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

	banner := shadowban.New(config.ShadowBan, clock)
	g.runners = append(g.runners, banner.Run)

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
	runners     []func(context.Context)
	description Description
}

func (g *guard) Inspect(click Click) bool { return g.jury.Inspect(click) }

func (g *guard) Committed(click Click) { g.jury.Committed(click) }

func (g *guard) Flagged() int { return g.jury.Flagged() }

func (g *guard) Describe() Description { return g.description }

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
