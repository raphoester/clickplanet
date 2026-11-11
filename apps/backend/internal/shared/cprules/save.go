package cprules

import (
	"context"
	"errors"
	"fmt"

	"google.golang.org/protobuf/proto"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
)

func NewSaveUseCase[M proto.Message](
	transactioner cpatomic.Transactioner, sets Store[M], parts Parts[M], playing Replaying[M],
) *SaveUseCase[M] {
	return &SaveUseCase[M]{transactioner: transactioner, sets: sets, parts: parts, playing: playing}
}

// SaveUseCase keeps the operator's set, checked over the default kept, then plays the set chosen last again.
type SaveUseCase[M proto.Message] struct {
	transactioner cpatomic.Transactioner
	sets          Store[M]
	parts         Parts[M]
	playing       Replaying[M]
}

func (u *SaveUseCase[M]) Execute(ctx context.Context, in Snapshot[M]) error {
	if err := u.transactioner.InTx(ctx, func(ctx context.Context) error { return u.keep(ctx, in) }); err != nil {
		return err //nolint:wrapcheck // keep says which set.
	}

	if _, _, err := u.playing.Execute(ctx); err != nil {
		return fmt.Errorf("kept the rule set %q, but could not play the set chosen last again: %w", in.Name, err)
	}

	return nil
}

func (u *SaveUseCase[M]) keep(ctx context.Context, in Snapshot[M]) error {
	defaults := in
	if in.Name != Default {
		kept, err := u.sets.Set(ctx, Default)
		if err != nil {
			return fmt.Errorf("cannot keep the rule set %q without default: %w", in.Name, err)
		}
		defaults = kept.TakeSnapshot()
	}

	set, err := in.Restore(u.parts, defaults)
	if err != nil {
		return fmt.Errorf("cannot keep the rule set %q: %w", in.Name, err)
	}
	if err := u.sets.Save(ctx, set); err != nil {
		return fmt.Errorf("could not keep the rule set %q: %w", in.Name, err)
	}

	if !set.Is(Default) {
		return nil
	}
	return u.everySetPlays(ctx)
}

// A default that leaves another set unable to play is refused, as that set would fail when the schedule picks it.
func (u *SaveUseCase[M]) everySetPlays(ctx context.Context) error {
	names, err := u.sets.Names(ctx)
	if err != nil {
		return fmt.Errorf("could not read the rule sets default plays under: %w", err)
	}

	var unplayable []error
	for _, name := range names {
		if _, err := u.sets.Set(ctx, name); err != nil {
			unplayable = append(unplayable, err)
		}
	}
	if len(unplayable) > 0 {
		return fmt.Errorf("cannot keep this default: %w", errors.Join(unplayable...))
	}

	return nil
}
