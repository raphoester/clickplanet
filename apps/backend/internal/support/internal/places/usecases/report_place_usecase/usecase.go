package report_place_usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/places"
)

type Saver interface {
	Save(ctx context.Context, reports ...places.Report) error
}

type In struct {
	Reporter  places.AccountID
	Tile      places.TileID
	Complaint places.Complaint
}

const timeout = 5 * time.Second

func New(ids places.IDProvider, clock cptime.Clock, saver Saver) *UseCase {
	return &UseCase{ids: ids, clock: clock, saver: saver}
}

type UseCase struct {
	ids   places.IDProvider
	clock cptime.Clock
	saver Saver
}

func (u *UseCase) Execute(ctx context.Context, in In) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	id, err := u.ids.NewID()
	if err != nil {
		return fmt.Errorf("failed to draw a report id: %w", err)
	}

	report, err := places.NewReport(id, in.Reporter, in.Tile, in.Complaint, u.clock.Now())
	if err != nil {
		return fmt.Errorf("cannot take the report: %w", err)
	}

	if err := u.saver.Save(ctx, report); err != nil {
		return fmt.Errorf("failed to keep the report: %w", err)
	}
	return nil
}
