package save_rule_set_usecase

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rules"
)

type Sets interface {
	Save(ctx context.Context, set rules.Set) error
}

func New(sets Sets) *UseCase {
	return &UseCase{sets: sets}
}

// UseCase keeps a set as an operator wrote it; the game follows it within a second if it is in force.
type UseCase struct {
	sets Sets
}

func (u *UseCase) Execute(ctx context.Context, in rules.Snapshot) error {
	set, err := in.Restore()
	if err != nil {
		return fmt.Errorf("cannot keep the rule set %q: %w", in.Name, err)
	}

	if err := u.sets.Save(ctx, set); err != nil {
		return fmt.Errorf("could not keep the rule set %q: %w", in.Name, err)
	}

	return nil
}
