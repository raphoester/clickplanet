package get_missions_usecase

import (
	"context"
	"errors"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missions"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var ErrUnknownCountry = errors.New("not a country")

type Sheets interface {
	Sheet(ctx context.Context, day missions.Day, account missions.AccountID) (missions.Sheet, error)
}

type Briefing interface {
	Of(ctx context.Context, day missions.Day, flag string) (missions.Brief, error)
}

type Countries interface {
	CheckCountry(country string) bool
}

type In struct {
	Account missions.AccountID
	Country string
}

type Out struct {
	Day   missions.Day
	Flag  string
	Brief missions.Brief
	Sheet missions.Sheet
}

func New(calendar missions.Calendar, clock cptime.Clock, sheets Sheets, briefing Briefing, countries Countries) *UseCase {
	return &UseCase{calendar: calendar, clock: clock, sheets: sheets, briefing: briefing, countries: countries}
}

type UseCase struct {
	calendar  missions.Calendar
	clock     cptime.Clock
	sheets    Sheets
	briefing  Briefing
	countries Countries
}

func (u *UseCase) Execute(ctx context.Context, in In) (Out, error) {
	day := u.calendar.DayOf(u.clock.Now())

	sheet, err := u.sheets.Sheet(ctx, day, in.Account)
	if err != nil {
		return Out{}, fmt.Errorf("failed to read the day's missions: %w", err)
	}

	out := Out{Day: day, Flag: in.Country, Sheet: sheet}
	if sheet.Enlisted() {
		out.Flag = sheet.Snapshot().Flag
	}
	if out.Flag == "" {
		return out, nil
	}
	if !sheet.Enlisted() && !u.countries.CheckCountry(out.Flag) {
		return Out{}, fmt.Errorf("%w: %q", ErrUnknownCountry, out.Flag)
	}

	out.Brief, err = u.briefing.Of(ctx, day, out.Flag)
	if err != nil {
		return Out{}, fmt.Errorf("failed to read %s's missions: %w", out.Flag, err)
	}
	return out, nil
}
