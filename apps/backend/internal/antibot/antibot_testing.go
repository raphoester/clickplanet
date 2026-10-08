//go:build testing

package antibot

import (
	"context"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/evidence"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/shadowban"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

func NewInMemory(
	config Config,
	clock cptime.Clock,
	observer Observer,
	bans *shadowban.MemoryStore,
	accountBans *shadowban.MemoryStore,
	evidences *evidence.MemoryPersistence,
) (*Guard, error) {
	if !config.Enabled {
		return &Guard{}, nil
	}

	return build(config, clock, observer, memoryDatabase{}, bans, accountBans, evidences)
}

type memoryDatabase struct{}

func (memoryDatabase) open(context.Context) error { return nil }

func (memoryDatabase) close() error { return nil }
