package spread_viruses_usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

type Map interface {
	Viruses() []clicks.Virus
}

type Contagion interface {
	Spread(ctx context.Context, id clicks.VirusID, most int, next time.Time) (clicks.Virus, error)
}

type Outbreak interface {
	Most() int
	Next() time.Time
	Due(virus clicks.Virus) bool
}

type Executor interface {
	Execute(ctx context.Context) (int, error)
}

func New(viruses Map, contagion Contagion, outbreak Outbreak, freeze func() clicks.Freeze) *UseCase {
	return &UseCase{viruses: viruses, contagion: contagion, outbreak: outbreak, freeze: freeze}
}

type UseCase struct {
	viruses   Map
	contagion Contagion
	outbreak  Outbreak
	freeze    func() clicks.Freeze
}

var _ Executor = (*UseCase)(nil)

// Execute spreads every virus due, and none while the map takes no play: a virus waits through a freeze.
func (u *UseCase) Execute(ctx context.Context) (int, error) {
	if u.freeze() {
		return 0, nil
	}

	spread := 0
	var failed []error
	for _, virus := range u.viruses.Viruses() {
		if !u.outbreak.Due(virus) {
			continue
		}
		id := virus.TakeSnapshot().ID
		_, err := u.contagion.Spread(ctx, id, u.outbreak.Most(), u.outbreak.Next())
		switch {
		case err == nil:
			spread++
		case errors.Is(err, clicks.ErrNoVirus):
		default:
			failed = append(failed, fmt.Errorf("failed to spread virus %s: %w", id, err))
		}
	}

	return spread, errors.Join(failed...)
}
