package forget_seen_usecase

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
)

type Deleter interface {
	DeleteSeen(ctx context.Context, account messages.AccountID) error
}

func New(deleter Deleter) *UseCase {
	return &UseCase{deleter: deleter}
}

type UseCase struct {
	deleter Deleter
}

func (u *UseCase) Execute(ctx context.Context, account messages.AccountID) error {
	if err := u.deleter.DeleteSeen(ctx, account); err != nil {
		return fmt.Errorf("failed to forget what the account has seen: %w", err)
	}
	return nil
}
