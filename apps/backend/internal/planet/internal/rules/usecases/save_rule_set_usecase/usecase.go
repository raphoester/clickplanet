package save_rule_set_usecase

import (
	"context"
	"errors"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rules"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
)

type Sets interface {
	Set(ctx context.Context, name rules.SetName) (rules.Set, error)
	Names(ctx context.Context) ([]rules.SetName, error)
	Save(ctx context.Context, set rules.Set) error
}

type Playing interface {
	Execute(ctx context.Context) (taken rules.Snapshot, took bool, err error)
}

func New(transactioner cpatomic.Transactioner, sets Sets, parts rules.Parts, playing Playing) *UseCase {
	return &UseCase{transactioner: transactioner, sets: sets, parts: parts, playing: playing}
}

type UseCase struct {
	transactioner cpatomic.Transactioner
	sets          Sets
	parts         rules.Parts
	playing       Playing
}

func (u *UseCase) Execute(ctx context.Context, in rules.Snapshot) error {
	if err := u.transactioner.InTx(ctx, func(ctx context.Context) error { return u.keep(ctx, in) }); err != nil {
		return err //nolint:wrapcheck // keep says which set.
	}

	if _, _, err := u.playing.Execute(ctx); err != nil {
		return fmt.Errorf("kept the rule set %q, but could not play the set chosen last again: %w", in.Name, err)
	}

	return nil
}

func (u *UseCase) keep(ctx context.Context, in rules.Snapshot) error {
	defaults := in
	if in.Name != rules.Default {
		kept, err := u.sets.Set(ctx, rules.Default)
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

	if !set.Is(rules.Default) {
		return nil
	}
	return u.everySetPlays(ctx)
}

// A default that leaves another set unable to play is refused, as that set would fail when the schedule picks it.
func (u *UseCase) everySetPlays(ctx context.Context) error {
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
