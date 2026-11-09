package arenas

import (
	"context"
	"fmt"
	"sync/atomic"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

// InPlay is the arena open now, in memory: every play asks it whether the map is frozen.
type InPlay struct {
	open  atomic.Pointer[Opening]
	rules func() Rules
	clock cptime.Clock
}

func NewInPlay(rules func() Rules, clock cptime.Clock) *InPlay {
	return &InPlay{rules: rules, clock: clock}
}

// Load reads the arena open at boot, if any.
func (p *InPlay) Load(ctx context.Context, kept Store) error {
	playing, err := kept.Playing(ctx)
	if err != nil {
		return fmt.Errorf("failed to read the arena open: %w", err)
	}
	if playing.Opened {
		p.Open(playing.Opening)
	}
	return nil
}

func (p *InPlay) Open(opening Opening) {
	p.open.Store(&opening)
}

// Close forgets the arena open when it is this one.
func (p *InPlay) Close(id ID) {
	if open := p.open.Load(); open != nil && open.ID == id {
		p.open.CompareAndSwap(open, nil)
	}
}

// Freeze is the arena's freeze, as long as the rules in force say.
func (p *InPlay) Freeze() clicks.Freeze {
	open := p.open.Load()
	return clicks.Freeze(open != nil && open.Freezes(p.clock.Now(), p.rules().Freeze))
}
