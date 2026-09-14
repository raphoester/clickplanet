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

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/catcher"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/defender"
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
// itself, and an Examination's Readings are already strings; the verdict ladder,
// the rule that tripped and the numbers behind it never leave this package as
// vocabulary the edge has to speak.
type (
	Click    = detect.Click       // one Click RPC, as the guard sees it
	Report   = detect.Report      // one ban, with every watchdog's opinion behind it
	Sentence = shadowban.Sentence // a scope's ban, as the operator tools read it

	Examination = detect.Examination // what the jury holds on one scope, as InspectPlayer reads it
	Reading     = detect.Reading     // one watchdog's opinion, already worded
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
	Defender  defenderConfig
	Catcher   catcherConfig
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

type defenderConfig struct {
	Enabled  bool
	Detector defender.Config
}

type catcherConfig struct {
	Enabled  bool
	Detector catcher.Config
}

// Observer is how a finding leaves this package, which measures and judges but
// logs and counts nothing itself. Both hooks are optional.
type Observer struct {
	// Every reaction, not only the ones arguing for a ban: the shape of the whole
	// distribution is what shows the bot band.
	OnReaction func(delay time.Duration)

	// Each caller's retake share, once a sweep, whether or not a bound is set to judge it.
	OnRetakeShare func(share float64)

	OnFlag func(report Report)

	// Bans that could not be restored at boot or saved since.
	OnStateError func(err error)

	// What was turned on, once, when the guard starts running. Never called when the block is off.
	OnStart func(description Description)
}

// New assembles the watchdogs the config asks for, the jury that crosses them and
// the one ban they all pass. With the block off it hands back a guard that drops
// and bans nothing, so a caller wires it the same way either way; enabling it with
// every watchdog off is an error, because that measures nothing while looking like a defence.
func New(config Config, clock cptime.Clock, observer Observer) (*Guard, error) {
	if !config.Enabled {
		return &Guard{}, nil
	}

	if clock == nil {
		clock = cptime.SystemClock{}
	}

	g := &Guard{onStart: observer.OnStart}

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

	if config.Defender.Enabled {
		watchdog := defender.New(config.Defender.Detector, clock, observer.OnRetakeShare)
		g.runners = append(g.runners, watchdog.Run)
		watchdogs = append(watchdogs, watchdog)
		names = append(names, defender.Name)
	}

	if config.Catcher.Enabled {
		watchdog := catcher.New(config.Catcher.Detector, clock)
		g.runners = append(g.runners, watchdog.Run)
		watchdogs = append(watchdogs, watchdog)
		names = append(names, catcher.Name)
		g.catcher = watchdog
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

// Description is what a guard says about itself, through OnStart, for the caller's boot line. Not
// the config back: this is what was turned on, after the defaults were applied.
type Description struct {
	Watchdogs   []string
	MinSuspects int
	Enforcing   bool
}

// Guard is what the click edge gates on. It is a struct, not an interface: each
// caller declares the one or two methods it uses, so the zero Guard New hands
// back when the block is off is enough. That one drops nothing, bans nothing and
// its Run returns at once.
type Guard struct {
	jury        *jury.Jury        // nil when the block is off
	banner      *shadowban.Banner // nil when the block is off
	catcher     *catcher.Watchdog // nil when the catcher is off
	runners     []func(context.Context)
	description Description
	onStart     func(Description)
}

// Attempted is every click tried, before the throttle: a loop's timing survives only here.
func (g *Guard) Attempted(click Click) {
	if g.Enabled() {
		g.jury.Attempted(click)
	}
}

// Inspect is called before the handler runs, because the map stops remembering
// who held the tile the moment it does.
func (g *Guard) Inspect(click Click) (drop bool) {
	return g.Enabled() && g.jury.Inspect(click)
}

// Committed is only for a click the handler accepted: a refused one recorded as a
// take is how the next honest clicker of that tile comes to look like it is reacting.
func (g *Guard) Committed(click Click) {
	if g.Enabled() {
		g.jury.Committed(click)
	}
}

// Caught and Missed tell the guard what a caller did with a bonus box offered to
// them: claimed after a delay, or let lapse. They say nothing about the click
// that follows until the jury next asks.
func (g *Guard) Caught(scope string, after time.Duration) {
	if g.catcher != nil {
		g.catcher.Caught(scope, after)
	}
}

func (g *Guard) Missed(scope string) {
	if g.catcher != nil {
		g.catcher.Missed(scope)
	}
}

// LoadBans reads the bans saved at the last shutdown, reporting a bad file through OnStateError.
func (g *Guard) LoadBans() {
	if g.Enabled() {
		g.banner.LoadState()
	}
}

// Flagged is how many callers are currently banned, for the gauge.
func (g *Guard) Flagged() int {
	if !g.Enabled() {
		return 0
	}

	return g.jury.Flagged()
}

// Ban is an operator's ban on a scope; a zero duration takes the ladder's.
func (g *Guard) Ban(scope string, duration time.Duration) Sentence {
	if !g.Enabled() {
		return Sentence{}
	}

	return g.banner.Ban(scope, duration)
}

func (g *Guard) Sentence(scope string) (Sentence, bool) {
	if !g.Enabled() {
		return Sentence{}, false
	}

	return g.banner.Sentence(scope)
}

// Examine reads every watchdog's opinion, the jury's decision and any ban on a scope, and changes nothing.
func (g *Guard) Examine(scope string) Examination {
	if !g.Enabled() {
		return Examination{Scope: scope}
	}

	examination := g.jury.Examine(scope)

	if sentence, banned := g.banner.Sentence(scope); banned {
		examination.Banned = true
		examination.Flags = sentence.Flags
		examination.Offence = sentence.Offence
		examination.BannedUntil = sentence.Until
	}

	return examination
}

// Enforcing says whether a ban drops anything.
func (g *Guard) Enforcing() bool { return g.Enabled() && g.banner.Enforcing() }

// Banned says whether a scope's actions should be dropped: false while enforce is off.
func (g *Guard) Banned(scope string) bool { return g.Enabled() && g.banner.Banned(scope) }

// Enabled is false for the guard New hands back when the block is off.
func (g *Guard) Enabled() bool { return g.jury != nil }

func (g *Guard) Name() string { return "antibot" }

// Run fans out to every sweeper enabled and blocks until they all return. One
// runner whatever the config turned on: how many sweepers there are is this
// package's business.
func (g *Guard) Run(ctx context.Context) {
	if !g.Enabled() {
		return
	}

	if g.onStart != nil {
		g.onStart(g.description)
	}

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
