package save_rule_set_usecase

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rules"
)

type Sets interface {
	Save(ctx context.Context, set rules.Set) error
}

type Playing interface {
	Execute(ctx context.Context) (taken rules.Snapshot, took bool, err error)
}

func New(sets Sets, parts rules.Parts, playing Playing) *UseCase {
	return &UseCase{sets: sets, parts: parts, playing: playing}
}

type UseCase struct {
	sets    Sets
	parts   rules.Parts
	playing Playing
}

func (u *UseCase) Execute(ctx context.Context, in rules.Snapshot) error {
	set, err := in.Restore(u.parts)
	if err != nil {
		return fmt.Errorf("cannot keep the rule set %q: %w", in.Name, err)
	}

	if err := u.sets.Save(ctx, set); err != nil {
		return fmt.Errorf("could not keep the rule set %q: %w", in.Name, err)
	}

	if _, _, err := u.playing.Execute(ctx); err != nil {
		return fmt.Errorf("kept the rule set %q, but could not play the set chosen last again: %w", in.Name, err)
	}

	return nil
}
