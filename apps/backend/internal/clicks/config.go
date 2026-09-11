package clicks

import (
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/metronome"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/retaker"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/sequencer"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/shadowban"
	"github.com/raphoester/clickplanet.lol-backend/internal/clicks/adapters/secondary/memory_tile_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/clicks/domain/runner"
	"github.com/raphoester/clickplanet.lol-backend/internal/kernel/ipblock"
	"github.com/raphoester/clickplanet.lol-backend/internal/kernel/ratelimit"
)

// Config is squashed into the process config, so these keys sit at the top
// level of the file where they have always been.
type Config struct {
	GameMap      GameMapConfig
	TilesStorage memory_tile_storage.Config
	Bookkeeper   BookkeeperConfig
	RateLimiter  ratelimit.Config
	VPNBlocklist ipblock.Config
	AntiBot      AntiBotConfig
}

type GameMapConfig struct {
	MaxIndex uint32
}

type BookkeeperConfig struct {
	Enabled bool
	Runner  runner.Config
}

// AntiBotConfig holds the watchdogs, the jury that crosses what they say, and
// the one shadow ban they all pass. A watchdog left out of the file is off, and
// the server names the ones it is actually running at boot.
type AntiBotConfig struct {
	Enabled bool

	ShadowBan shadowban.Config
	Jury      antibot.Config

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
