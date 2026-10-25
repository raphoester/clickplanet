package forget_seasons_usecase

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/seasons"
)

type Records interface {
	Record(ctx context.Context, account players.AccountID) (seasons.Record, error)
	Save(ctx context.Context, record seasons.Record) error
}

type UseCase struct {
	records Records
}

func New(records Records) *UseCase {
	return &UseCase{records: records}
}

func (u *UseCase) Execute(ctx context.Context, account players.AccountID) error {
	record, err := u.records.Record(ctx, account)
	if err != nil {
		return fmt.Errorf("failed to read the seasons played: %w", err)
	}
	if err := u.records.Save(ctx, record.Forgotten()); err != nil {
		return fmt.Errorf("failed to forget the seasons played: %w", err)
	}
	return nil
}
