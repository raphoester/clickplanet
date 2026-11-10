package screen_names_usecase

import (
	"context"
	"fmt"
	"maps"
	"slices"

	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/verdicts"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Store interface {
	Of(ctx context.Context, accounts []verdicts.AccountID) ([]verdicts.Verdict, error)
	Keep(ctx context.Context, verdicts []verdicts.Verdict) error
}

type Judge interface {
	// A judgement for each name it could judge; a name it left out stays hidden until it is judged.
	Judge(ctx context.Context, names []string) ([]verdicts.Judgement, error)
}

func New(store Store, judge Judge, clock cptime.Clock) *UseCase {
	return &UseCase{store: store, judge: judge, clock: clock}
}

type UseCase struct {
	store Store
	judge Judge
	clock cptime.Clock
}

// The names the game's posts may show. A name not judged yet is judged now, and its verdict kept.
func (u *UseCase) Execute(ctx context.Context, names map[verdicts.AccountID]string) (map[verdicts.AccountID]string, error) {
	kept, err := u.store.Of(ctx, slices.Collect(maps.Keys(names)))
	if err != nil {
		return nil, fmt.Errorf("failed to read the verdicts of the names: %w", err)
	}
	shown, unjudged := verdicts.Screen(names, kept)
	if len(unjudged) == 0 {
		return shown, nil
	}

	judgements, err := u.judge.Judge(ctx, slices.Sorted(maps.Values(unjudged)))
	if err != nil {
		return nil, fmt.Errorf("failed to judge %d names: %w", len(unjudged), err)
	}
	judged := verdicts.Judged(unjudged, judgements, u.clock.Now())
	if err := u.store.Keep(ctx, judged); err != nil {
		return nil, fmt.Errorf("failed to keep the verdicts of the names: %w", err)
	}

	newlyShown, _ := verdicts.Screen(unjudged, judged)
	maps.Copy(shown, newlyShown)
	return shown, nil
}
