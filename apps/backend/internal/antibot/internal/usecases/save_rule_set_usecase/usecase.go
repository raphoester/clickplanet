package save_rule_set_usecase

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/rules"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/usecases/play_rules_usecase"
)

type Sets interface {
	Save(ctx context.Context, set rules.Snapshot) error
}

type Players interface {
	Execute(ctx context.Context) (play_rules_usecase.Taken, bool, error)
}

func New(sets Sets, parts rules.Parts, players Players) *UseCase {
	return &UseCase{sets: sets, parts: parts, players: players}
}

// UseCase keeps a set as an operator wrote it; the antibot judges with it on return when it is the one it plays.
type UseCase struct {
	sets    Sets
	parts   rules.Parts
	players Players
}

func (u *UseCase) Execute(ctx context.Context, in rules.Snapshot) error {
	set, err := in.Restore(u.parts)
	if err != nil {
		return fmt.Errorf("cannot keep the rule set %q: %w", in.Name, err)
	}

	if err := u.sets.Save(ctx, set.TakeSnapshot()); err != nil {
		return fmt.Errorf("could not keep the rule set %q: %w", in.Name, err)
	}

	if _, _, err := u.players.Execute(ctx); err != nil {
		return fmt.Errorf("kept the rule set %q, but could not play the rules: %w", in.Name, err)
	}

	return nil
}
