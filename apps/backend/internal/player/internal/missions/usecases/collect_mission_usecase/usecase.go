package collect_mission_usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/missions"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var (
	ErrNotLinked = errors.New("only a signed-in account collects a mission")
	ErrNoRoom    = errors.New("the account holds as much of the reward as it may")
)

type Sheets interface {
	Sheet(ctx context.Context, day missions.Day, account players.AccountID) (missions.Sheet, error)
	Collect(ctx context.Context, day missions.Day, account players.AccountID, slot missions.Slot, at time.Time,
		pay func(context.Context) error) error
}

type Briefing interface {
	Of(ctx context.Context, day missions.Day, flag string) (missions.Brief, error)
}

type Charger interface {
	Grant(ctx context.Context, account players.AccountID, reward missions.Reward) (uint32, error)
}

type In struct {
	Account players.AccountID
	Linked  bool
	Slot    uint32
}

type Out struct {
	Reward missions.Reward
}

func New(calendar missions.Calendar, clock cptime.Clock, sheets Sheets, briefing Briefing, charger Charger) *UseCase {
	return &UseCase{calendar: calendar, clock: clock, sheets: sheets, briefing: briefing, charger: charger}
}

type UseCase struct {
	calendar missions.Calendar
	clock    cptime.Clock
	sheets   Sheets
	briefing Briefing
	charger  Charger
}

func (u *UseCase) Execute(ctx context.Context, in In) (Out, error) {
	if !in.Linked {
		return Out{}, ErrNotLinked
	}
	slot, err := missions.SlotOf(in.Slot)
	if err != nil {
		return Out{}, err //nolint:wrapcheck // the domain named it.
	}

	now := u.clock.Now()
	day := u.calendar.DayOf(now)
	sheet, err := u.sheets.Sheet(ctx, day, in.Account)
	if err != nil {
		return Out{}, fmt.Errorf("failed to read the day's missions: %w", err)
	}
	if !sheet.Enlisted() {
		return Out{}, missions.ErrNotDone
	}

	brief, err := u.briefing.Of(ctx, day, sheet.Snapshot().Flag)
	if err != nil {
		return Out{}, fmt.Errorf("failed to read the day's missions: %w", err)
	}
	mission, err := brief.Mission(slot)
	if err != nil {
		return Out{}, err //nolint:wrapcheck // the domain named it.
	}
	if err := sheet.CollectError(mission); err != nil {
		return Out{}, err //nolint:wrapcheck // the domain named it.
	}

	reward := mission.Snapshot().Reward
	if err := u.sheets.Collect(ctx, day, in.Account, slot, now, func(ctx context.Context) error {
		kept, err := u.charger.Grant(ctx, in.Account, reward)
		if err != nil {
			return err //nolint:wrapcheck // the charger named it.
		}
		if kept == 0 {
			return ErrNoRoom
		}
		reward.Amount = kept
		return nil
	}); err != nil {
		return Out{}, fmt.Errorf("failed to collect mission %d: %w", slot, err)
	}

	return Out{Reward: reward}, nil
}
