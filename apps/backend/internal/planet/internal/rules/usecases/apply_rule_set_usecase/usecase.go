package apply_rule_set_usecase

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rules"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Catalog interface {
	Set(name rules.SetName) (rules.Set, error)
}

type Changes interface {
	Change(ctx context.Context, request rules.RequestID) (rules.Change, error)
	Save(ctx context.Context, change rules.Change) error
}

type Rules interface {
	Enforce(set rules.Set)
}

type In struct {
	Request rules.RequestID
	Set     rules.SetName
	Source  rules.Source
}

func New(catalog Catalog, changes Changes, inForce Rules, clock cptime.Clock) *UseCase {
	return &UseCase{catalog: catalog, changes: changes, inForce: inForce, clock: clock}
}

type UseCase struct {
	catalog Catalog
	changes Changes
	inForce Rules
	clock   cptime.Clock

	// One at a time, so the rules in force are always the change kept last.
	mu sync.Mutex
}

func (u *UseCase) Execute(ctx context.Context, in In) error {
	u.mu.Lock()
	defer u.mu.Unlock()

	set, err := u.catalog.Set(in.Set)
	if err != nil {
		return fmt.Errorf("cannot apply request %q: %w", in.Request, err)
	}

	kept, err := u.changes.Change(ctx, in.Request)
	if err != nil {
		return fmt.Errorf("could not tell whether request %q was applied: %w", in.Request, err)
	}

	change, err := rules.NewChange(in.Request, set, in.Source, u.clock.Now(), kept)
	if errors.Is(err, rules.ErrAppliedAlready) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("cannot apply request %q: %w", in.Request, err)
	}

	if err := u.changes.Save(ctx, change); err != nil {
		return fmt.Errorf("could not keep request %q: %w", in.Request, err)
	}
	u.inForce.Enforce(set)

	return nil
}
