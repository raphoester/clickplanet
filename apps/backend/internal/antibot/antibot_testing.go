//go:build testing

package antibot

import (
	"context"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/bans"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/convictions"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/evidence"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

func NewInMemory(
	config Config,
	clock cptime.Clock,
	observer Observer,
	bans *bans.MemoryStore,
	accountBans *bans.MemoryStore,
	convicted *convictions.MemoryStore,
	evidences *evidence.MemoryPersistence,
) (*Guard, error) {
	if !config.Enabled {
		return &Guard{}, nil
	}

	return build(config, clock, observer, memoryDatabase{}, bans, accountBans, convicted, evidences)
}

type memoryDatabase struct{}

func (memoryDatabase) open(context.Context) error { return nil }

func (memoryDatabase) close() error { return nil }
