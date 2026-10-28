package get_missions_usecase

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missions"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Sheets interface {
	Sheet(ctx context.Context, day missions.Day, account missions.AccountID) (missions.Sheet, error)
}

type Briefing interface {
	Of(ctx context.Context, day missions.Day, account missions.AccountID) (missions.Brief, error)
}

type In struct {
	Account missions.AccountID
}

type Out struct {
	Day   missions.Day
	Brief missions.Brief
	Sheet missions.Sheet
}

func New(calendar missions.Calendar, clock cptime.Clock, sheets Sheets, briefing Briefing) *UseCase {
	return &UseCase{calendar: calendar, clock: clock, sheets: sheets, briefing: briefing}
}

type UseCase struct {
	calendar missions.Calendar
	clock    cptime.Clock
	sheets   Sheets
	briefing Briefing
}

func (u *UseCase) Execute(ctx context.Context, in In) (Out, error) {
	day := u.calendar.DayOf(u.clock.Now())

	sheet, err := u.sheets.Sheet(ctx, day, in.Account)
	if err != nil {
		return Out{}, fmt.Errorf("failed to read the day's progress: %w", err)
	}
	brief, err := u.briefing.Of(ctx, day, in.Account)
	if err != nil {
		return Out{}, fmt.Errorf("failed to read the day's missions: %w", err)
	}
	return Out{Day: day, Brief: brief, Sheet: sheet}, nil
}
