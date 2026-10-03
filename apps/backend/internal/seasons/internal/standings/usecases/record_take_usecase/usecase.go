package record_take_usecase

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/calendar"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/standings"
)

type Recorder interface {
	RecordTake(ctx context.Context, season calendar.Number, take standings.Take) error
}

type UseCase struct {
	seasons  calendar.Calendar
	recorder Recorder
}

func New(seasons calendar.Calendar, recorder Recorder) *UseCase {
	return &UseCase{seasons: seasons, recorder: recorder}
}

func (u *UseCase) Execute(ctx context.Context, take standings.Take) error {
	season, ok := take.Season(u.seasons)
	if !ok {
		return nil
	}
	if err := u.recorder.RecordTake(ctx, season, take); err != nil {
		return fmt.Errorf("failed to count the take: %w", err)
	}
	return nil
}
