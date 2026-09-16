//go:build testing

package antibot

import (
	"context"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/detect"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/evidence"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/shadowban"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

// NewInMemory is New with the bans and the evidence kept in the persistences given instead of postgres.
func NewInMemory(
	config Config,
	clock cptime.Clock,
	observer Observer,
	bans *shadowban.MemoryPersistence,
	evidences *evidence.MemoryPersistence,
) (*Guard, error) {
	if !config.Enabled {
		return &Guard{}, nil
	}

	return build(config, clock, observer, memoryDatabase{}, bans, evidences)
}

type memoryDatabase struct{}

func (memoryDatabase) open(context.Context) error { return nil }

func (memoryDatabase) close() error { return nil }

// Drop and Challenge build the two Outcomes that are not the zero one. They are
// behind the tag because nothing in production builds an Outcome — the guard
// answers one and the edge asks it two questions — but a fake guard standing in
// for this one has to be able to say both.
func Drop() Outcome { return detect.Drop() }

func Challenge() Outcome { return detect.Challenge() }
