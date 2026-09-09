package app

import (
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/adapters/secondary/memory_chat_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/domain/chat_service"
	"github.com/raphoester/clickplanet.lol-backend/internal/clicks/adapters/secondary/memory_tile_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/clicks/domain/runner"
	"github.com/raphoester/clickplanet.lol-backend/internal/kernel/ipblock"
	"github.com/raphoester/clickplanet.lol-backend/internal/kernel/ratelimit"
)

type Config struct {
	HTTPServer   HTTPServerConfig
	GameMap      GameMapConfig
	TilesStorage memory_tile_storage.Config
	Bookkeeper   BookkeeperConfig
	RateLimiter  ratelimit.Config
	VPNBlocklist ipblock.Config

	Chat ChatConfig
}

// ChatConfig controls the live chat. Off by default, and the routes are only
// declared when it is on — a chat that is turned off is not reachable at all.
type ChatConfig struct {
	Enabled bool
	Storage memory_chat_storage.Config
	Service chat_service.Config

	// RateLimiter throttles SendMessage per source IP.
	RateLimiter ratelimit.Config

	// BlockedIPs are refused every chat RPC outright. Config-driven so an
	// abusive sender is cut off with a restart rather than a rebuild.
	BlockedIPs []string
}

type HTTPServerConfig struct {
	BindAddress string
}

type GameMapConfig struct {
	MaxIndex uint32
}

type BookkeeperConfig struct {
	Enabled bool
	Runner  runner.Config
}
