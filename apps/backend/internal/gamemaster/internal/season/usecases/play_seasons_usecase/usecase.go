package play_seasons_usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/season"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Seasons interface {
	Calendar(ctx context.Context) (season.Calendar, error)
}

type Commands interface {
	Play(ctx context.Context, season season.Season, decidedAt time.Time) error
}

// Execute answers the seasons it told this time.
type Executor interface {
	Execute(ctx context.Context) (told []season.Season, err error)
}

func New(seasons Seasons, commands Commands, clock cptime.Clock) *UseCase {
	return &UseCase{seasons: seasons, commands: commands, clock: clock, told: map[season.Number]season.Snapshot{}}
}

// UseCase tells scoring each season once it starts, again when its window changes, and every started season once after
// each boot, when it remembers none. A command Redis did not take is sent again at the next Execute.
type UseCase struct {
	seasons  Seasons
	commands Commands
	clock    cptime.Clock

	told map[season.Number]season.Snapshot
}

func (u *UseCase) Execute(ctx context.Context) ([]season.Season, error) {
	now := u.clock.Now()
	calendar, err := u.seasons.Calendar(ctx)
	if err != nil {
		return nil, fmt.Errorf("could not read the seasons: %w", err)
	}

	var (
		told []season.Season
		errs []error
	)
	for _, started := range calendar.StartedBy(now) {
		snapshot := started.TakeSnapshot()
		if last, ok := u.told[snapshot.Number]; ok && last.StartsAt.Equal(snapshot.StartsAt) && last.EndsAt.Equal(snapshot.EndsAt) {
			continue
		}
		if err := u.commands.Play(ctx, started, now); err != nil {
			errs = append(errs, err)
			continue
		}
		u.told[snapshot.Number] = snapshot
		told = append(told, started)
	}
	return told, errors.Join(errs...)
}
