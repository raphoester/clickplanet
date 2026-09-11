package clicks

import (
	"errors"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/metronome"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/retaker"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/sequencer"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/shadowban"
	"github.com/raphoester/clickplanet.lol-backend/internal/clicks/adapters/secondary/memory_tile_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/kernel/ipblock"
	"github.com/raphoester/clickplanet.lol-backend/internal/kernel/ratelimit"
	"github.com/raphoester/clickplanet.lol-backend/internal/kernel/session"
)

// Config is squashed into the process config, so these keys sit at the top
// level of the file where they have always been.
type Config struct {
	GameMap      GameMapConfig
	TilesStorage memory_tile_storage.Config
	RateLimiter  ratelimit.Config
	VPNBlocklist ipblock.Config
	AntiBot      AntiBotConfig

	// The same `session:` keys the session context mints with. Declared here
	// rather than handed over, so this module needs nothing but its config.
	Session session.Config
}

type GameMapConfig struct {
	MaxIndex uint32
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

// Validate refuses a map of no tiles, which would refuse every click. The
// `session:` block it reads is the session context's to check, and does not
// exist without it.
func (c Config) Validate() error {
	if c.GameMap.MaxIndex == 0 {
		return errors.New("gameMap.maxIndex is zero: the map has no tiles")
	}

	return nil
}
