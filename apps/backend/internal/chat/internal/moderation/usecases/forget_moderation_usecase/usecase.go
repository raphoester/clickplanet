package forget_moderation_usecase

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
)

type Forgetter interface {
	Forget(ctx context.Context, account messages.AccountID) error
}

func New(history Forgetter) *UseCase {
	return &UseCase{history: history}
}

type UseCase struct {
	history Forgetter
}

// The actions taken on the account go, with the copies of what it said; an operator's account is taken off the
// actions it took, which stay.
func (u *UseCase) Execute(ctx context.Context, account messages.AccountID) error {
	if err := u.history.Forget(ctx, account); err != nil {
		return fmt.Errorf("failed to forget the account's moderation history: %w", err)
	}
	return nil
}
