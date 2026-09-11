package app

import (
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/metronome"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/retaker"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/sequencer"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/shadowban"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/adapters/secondary/memory_chat_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/domain/chat_service"
	"github.com/raphoester/clickplanet.lol-backend/internal/clicks/adapters/secondary/memory_tile_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/clicks/domain/runner"
	"github.com/raphoester/clickplanet.lol-backend/internal/kernel/ipblock"
	"github.com/raphoester/clickplanet.lol-backend/internal/kernel/ratelimit"
	"github.com/raphoester/clickplanet.lol-backend/internal/kernel/turnstile"
)

type Config struct {
	HTTPServer   HTTPServerConfig
	GameMap      GameMapConfig
	TilesStorage memory_tile_storage.Config
	Bookkeeper   BookkeeperConfig
	RateLimiter  ratelimit.Config
	VPNBlocklist ipblock.Config
	AntiBot      AntiBotConfig

	Session SessionConfig

	Chat ChatConfig
}

// SessionConfig gates the Click RPC on a token this server minted, which is the
// one thing an address-based defence cannot do: refuse a caller that never
// proved anything, however many addresses it has.
type SessionConfig struct {
	// Off registers nothing: session.v1.SessionService/ 404s and clicks are
	// judged on address alone, as they were before this existed.
	Enabled bool

	// Off counts what enforcing would refuse without refusing it. Ship in this
	// mode, watch click_session_checks, then turn it on.
	Enforce bool

	// Signs the tokens. Empty regenerates one at boot, which invalidates every
	// session in flight on each restart.
	Secret string

	// How long a minted token is accepted for.
	TTL time.Duration

	// Per-IP throttle on minting. Minting costs a siteverify round trip, so it
	// needs its own budget rather than the click one.
	RateLimiter ratelimit.Config

	Turnstile turnstile.Config
}

const (
	defaultSessionTTL             = time.Hour
	defaultSessionTurnstileAction = "session"
)

func (c SessionConfig) withDefaults() SessionConfig {
	if c.TTL <= 0 {
		c.TTL = defaultSessionTTL
	}
	if c.Turnstile.Action == "" {
		c.Turnstile.Action = defaultSessionTurnstileAction
	}
	return c
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

type ChatConfig struct {
	Enabled bool
	Storage memory_chat_storage.Config
	Service chat_service.Config

	RateLimiter ratelimit.Config

	BlockedIPs []string
}

type HTTPServerConfig struct {
	BindAddress string

	// Must stay well under the proxy's idle cut: Cloudflare answers 524 at ~125s.
	StreamHeartbeat time.Duration
}

type GameMapConfig struct {
	MaxIndex uint32
}

type BookkeeperConfig struct {
	Enabled bool
	Runner  runner.Config
}
