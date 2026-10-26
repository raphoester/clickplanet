package save_rule_set_usecase

import (
	"context"
	"errors"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/rules"
)

var ErrPlaying = errors.New("this rule set plays now: save the rules under another name, and the schedule plays it")

type Sets interface {
	Save(ctx context.Context, set rules.Snapshot) error
}

type Playing interface {
	Plays(name rules.SetName) bool
}

func New(sets Sets, parts rules.Parts, playing Playing) *UseCase {
	return &UseCase{sets: sets, parts: parts, playing: playing}
}

// UseCase keeps a set as an operator wrote it. The rules in force change only through the schedule, so a set that plays
// now is refused.
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

	if u.playing.Plays(in.Name) {
		return fmt.Errorf("cannot keep the rule set %q: %w", in.Name, ErrPlaying)
	}

	if err := u.sets.Save(ctx, set.TakeSnapshot()); err != nil {
		return fmt.Errorf("could not keep the rule set %q: %w", in.Name, err)
	}

	return nil
}
