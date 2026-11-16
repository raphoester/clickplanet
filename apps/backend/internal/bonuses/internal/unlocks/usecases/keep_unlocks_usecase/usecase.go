package keep_unlocks_usecase

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/unlocks"
)

type Unlocks interface {
	Change(ctx context.Context, account bonuses.Account, change func(unlocks.Unlocks) unlocks.Unlocks) error
}

func New(unlocks Unlocks) *UseCase {
	return &UseCase{unlocks: unlocks}
}

// UseCase keeps what progression decided an account may be drawn, when it is the latest decision.
type UseCase struct {
	unlocks Unlocks
}

func (u *UseCase) Execute(ctx context.Context, account bonuses.Account, decision unlocks.Decision) error {
	if err := decision.TimeError(); err != nil {
		return fmt.Errorf("failed to read the unlocks of %s: %w", account, err)
	}
	if err := u.unlocks.Change(ctx, account, func(kept unlocks.Unlocks) unlocks.Unlocks {
		return kept.Decided(decision)
	}); err != nil {
		return fmt.Errorf("failed to keep the unlocks of %s: %w", account, err)
	}
	return nil
}
