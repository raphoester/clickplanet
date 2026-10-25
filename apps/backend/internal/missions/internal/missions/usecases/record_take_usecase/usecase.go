package record_take_usecase

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missions"
)

type Briefing interface {
	Of(ctx context.Context, day missions.Day, flag string) (missions.Brief, error)
}

type Sheets interface {
	ChangeSheet(ctx context.Context, day missions.Day, account missions.AccountID, change func(missions.Sheet) missions.Sheet) error
}

func New(calendar missions.Calendar, briefing Briefing, sheets Sheets) *UseCase {
	return &UseCase{calendar: calendar, briefing: briefing, sheets: sheets}
}

type UseCase struct {
	calendar missions.Calendar
	briefing Briefing
	sheets   Sheets
}

func (u *UseCase) Execute(ctx context.Context, take missions.Take) error {
	day := u.calendar.DayOf(take.At)

	brief, err := u.briefing.Of(ctx, day, take.Flag)
	if err != nil {
		return fmt.Errorf("failed to read %s's missions: %w", take.Flag, err)
	}

	if err := u.sheets.ChangeSheet(ctx, day, take.Account, func(sheet missions.Sheet) missions.Sheet {
		return sheet.WithTake(take, brief)
	}); err != nil {
		return fmt.Errorf("failed to count the take on the day's missions: %w", err)
	}
	return nil
}
