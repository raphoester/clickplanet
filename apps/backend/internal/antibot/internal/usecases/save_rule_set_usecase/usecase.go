package save_rule_set_usecase

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/rules"
)

type Sets interface {
	Save(ctx context.Context, set rules.Snapshot) error
}

func New(sets Sets, parts rules.Parts) *UseCase {
	return &UseCase{sets: sets, parts: parts}
}

// UseCase keeps a set as an operator wrote it; the antibot follows it within a second if it is the one it plays.
type UseCase struct {
	sets  Sets
	parts rules.Parts
}

func (u *UseCase) Execute(ctx context.Context, in rules.Snapshot) error {
	set, err := in.Restore(u.parts)
	if err != nil {
		return fmt.Errorf("cannot keep the rule set %q: %w", in.Name, err)
	}

	if err := u.sets.Save(ctx, set.TakeSnapshot()); err != nil {
		return fmt.Errorf("could not keep the rule set %q: %w", in.Name, err)
	}

	return nil
}
