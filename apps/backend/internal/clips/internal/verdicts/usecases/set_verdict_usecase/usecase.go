package set_verdict_usecase

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/verdicts"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Store interface {
	Verdict(ctx context.Context, account verdicts.AccountID) (verdicts.Verdict, error)
	Keep(ctx context.Context, verdicts []verdicts.Verdict) error
}

func New(store Store, clock cptime.Clock) *UseCase {
	return &UseCase{store: store, clock: clock}
}

type UseCase struct {
	store Store
	clock cptime.Clock
}

// The operator's verdict of the name the player is judged under. verdicts.ErrNoVerdict for a name never judged.
func (u *UseCase) Execute(ctx context.Context, account verdicts.AccountID, shown bool, reason string) (verdicts.Verdict, error) {
	kept, err := u.store.Verdict(ctx, account)
	if err != nil {
		return verdicts.Verdict{}, fmt.Errorf("failed to read the verdict of the player's name: %w", err)
	}
	overruled := kept.Overruled(shown, reason, u.clock.Now())
	if err := u.store.Keep(ctx, []verdicts.Verdict{overruled}); err != nil {
		return verdicts.Verdict{}, fmt.Errorf("failed to keep the operator's verdict: %w", err)
	}
	return overruled, nil
}
