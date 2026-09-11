package app

import (
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat"
	"github.com/raphoester/clickplanet.lol-backend/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/session"
)

type Config struct {
	HTTPServer HTTPServerConfig

	// Squashed: the clicks keys sit at the top level of the file.
	Clicks clicks.Config `koanf:",squash"`

	Session session.Config
	Chat    chat.Config
}

type HTTPServerConfig struct {
	BindAddress string

	// Must stay well under the proxy's idle cut: Cloudflare answers 524 at ~125s.
	StreamHeartbeat time.Duration
}
