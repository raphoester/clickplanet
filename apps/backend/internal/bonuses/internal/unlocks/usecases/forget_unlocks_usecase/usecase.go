package forget_unlocks_usecase

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/bonuses"
)

type Unlocks interface {
	Forget(ctx context.Context, account bonuses.Account) error
}

func New(unlocks Unlocks) *UseCase {
	return &UseCase{unlocks: unlocks}
}

// UseCase forgets what was unlocked for an account player forgot. A raise read before the forget can keep a row again,
// as a late fact does in progression.
type UseCase struct {
	unlocks Unlocks
}

func (u *UseCase) Execute(ctx context.Context, account bonuses.Account) error {
	if err := u.unlocks.Forget(ctx, account); err != nil {
		return fmt.Errorf("failed to forget the unlocks of %s: %w", account, err)
	}
	return nil
}
