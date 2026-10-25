package restore_rules_usecase

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rules"
)

type Changes interface {
	Latest(ctx context.Context) (rules.Change, error)
}

type Catalog interface {
	SetOf(change rules.Change) (rules.Set, error)
}

type Rules interface {
	Enforce(set rules.Set)
}

func New(changes Changes, catalog Catalog, inForce Rules) *UseCase {
	return &UseCase{changes: changes, catalog: catalog, inForce: inForce}
}

type UseCase struct {
	changes Changes
	catalog Catalog
	inForce Rules
}

func (u *UseCase) Execute(ctx context.Context) error {
	latest, err := u.changes.Latest(ctx)
	if err != nil {
		return fmt.Errorf("could not read the rule set applied last: %w", err)
	}

	set, err := u.catalog.SetOf(latest)
	if err != nil {
		return fmt.Errorf("the rule set applied last is not in rules.sets: %w", err)
	}
	u.inForce.Enforce(set)

	return nil
}
