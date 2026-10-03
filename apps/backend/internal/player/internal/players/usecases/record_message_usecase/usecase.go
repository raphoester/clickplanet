package record_message_usecase

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
)

type Stats interface {
	RecordMessage(ctx context.Context, account players.AccountID) error
}

type UseCase struct {
	stats Stats
}

func New(stats Stats) *UseCase {
	return &UseCase{stats: stats}
}

type In struct {
	Account players.AccountID
}

func (u *UseCase) Execute(ctx context.Context, in In) error {
	if err := u.stats.RecordMessage(ctx, in.Account); err != nil {
		return fmt.Errorf("failed to record the message: %w", err)
	}
	return nil
}
