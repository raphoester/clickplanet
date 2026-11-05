package get_missions_usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/days"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/flagmissions"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missions"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Calendar interface {
	DayAt(ctx context.Context, at time.Time) (days.Day, error)
}

type Sheets interface {
	Sheet(ctx context.Context, day days.Day, account missions.AccountID) (missions.Sheet, error)
}

type Briefing interface {
	Of(ctx context.Context, day days.Day, account missions.AccountID) (missions.Brief, error)
}

type Flags interface {
	OfPlayer(ctx context.Context, day days.Day, account missions.AccountID) (flagmissions.Day, error)
}

type In struct {
	Account missions.AccountID
}

// Out is the zero Out when no day runs: no missions then.
type Out struct {
	Day   days.Day
	Brief missions.Brief
	Sheet missions.Sheet
	Flag  flagmissions.Day
}

func New(calendar Calendar, clock cptime.Clock, sheets Sheets, briefing Briefing, flags Flags) *UseCase {
	return &UseCase{calendar: calendar, clock: clock, sheets: sheets, briefing: briefing, flags: flags}
}

type UseCase struct {
	calendar Calendar
	clock    cptime.Clock
	sheets   Sheets
	briefing Briefing
	flags    Flags
}

func (u *UseCase) Execute(ctx context.Context, in In) (Out, error) {
	day, err := u.calendar.DayAt(ctx, u.clock.Now())
	if errors.Is(err, days.ErrNoDay) {
		return Out{}, nil
	}
	if err != nil {
		return Out{}, fmt.Errorf("failed to read the day: %w", err)
	}

	sheet, err := u.sheets.Sheet(ctx, day, in.Account)
	if err != nil {
		return Out{}, fmt.Errorf("failed to read the day's progress: %w", err)
	}
	brief, err := u.briefing.Of(ctx, day, in.Account)
	if err != nil {
		return Out{}, fmt.Errorf("failed to read the day's missions: %w", err)
	}
	flag, err := u.flags.OfPlayer(ctx, day, in.Account)
	if err != nil {
		return Out{}, fmt.Errorf("failed to read the flag's mission: %w", err)
	}
	return Out{Day: day, Brief: brief, Sheet: sheet, Flag: flag}, nil
}
