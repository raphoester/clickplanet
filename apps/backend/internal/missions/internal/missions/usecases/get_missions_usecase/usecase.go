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
	Of(ctx context.Context, day missions.Day, flag string) (missions.Brief, error)
}

type Allegiances interface {
	FlagOf(ctx context.Context, account missions.AccountID) (string, error)
}

type In struct {
	Account missions.AccountID
}

type Out struct {
	Day   missions.Day
	Flag  string
	Brief missions.Brief
	Sheet missions.Sheet
}

func New(calendar missions.Calendar, clock cptime.Clock, sheets Sheets, briefing Briefing, allegiances Allegiances) *UseCase {
	return &UseCase{calendar: calendar, clock: clock, sheets: sheets, briefing: briefing, allegiances: allegiances}
}

type UseCase struct {
	calendar    missions.Calendar
	clock       cptime.Clock
	sheets      Sheets
	briefing    Briefing
	allegiances Allegiances
}

func (u *UseCase) Execute(ctx context.Context, in In) (Out, error) {
	day := u.calendar.DayOf(u.clock.Now())

	sheet, err := u.sheets.Sheet(ctx, day, in.Account)
	if err != nil {
		return Out{}, fmt.Errorf("failed to read the day's missions: %w", err)
	}

	out := Out{Day: day, Flag: sheet.Snapshot().Flag, Sheet: sheet}
	if !sheet.Enlisted() {
		out.Flag, err = u.allegiances.FlagOf(ctx, in.Account)
		if err != nil {
			return Out{}, fmt.Errorf("failed to read the flag the account plays for: %w", err)
		}
	}
	if out.Flag == "" {
		return out, nil
	}

	out.Brief, err = u.briefing.Of(ctx, day, out.Flag)
	if err != nil {
		return Out{}, fmt.Errorf("failed to read %s's missions: %w", out.Flag, err)
	}
	return out, nil
}
