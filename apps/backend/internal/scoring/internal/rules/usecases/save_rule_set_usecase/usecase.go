package save_rule_set_usecase

import (
	"context"
	"errors"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/rules"
)

var ErrPlaying = errors.New("this rule set plays now: save the rules under another name, and the schedule plays it")

type Sets interface {
	Save(ctx context.Context, set rules.Set) error
}

type Playing interface {
	Plays(name rules.SetName) bool
}

func New(sets Sets, playing Playing) *UseCase {
	return &UseCase{sets: sets, playing: playing}
}

// UseCase keeps an operator's set, and refuses the set that plays now: only the schedule changes what plays.
type UseCase struct {
	sets    Sets
	playing Playing
}

func (u *UseCase) Execute(ctx context.Context, in rules.Snapshot) error {
	set, err := in.Restore()
	if err != nil {
		return fmt.Errorf("cannot keep the rule set %q: %w", in.Name, err)
	}

	if u.playing.Plays(in.Name) {
		return fmt.Errorf("cannot keep the rule set %q: %w", in.Name, ErrPlaying)
	}

	if err := u.sets.Save(ctx, set); err != nil {
		return fmt.Errorf("could not keep the rule set %q: %w", in.Name, err)
	}

	return nil
}
